package wkldreplicate

import (
	"fmt"
	"strings"
	"time"

	"github.com/brian1917/illumioapi"
	"github.com/brian1917/workloader/cmd/wkldexport"
	"github.com/brian1917/workloader/cmd/wkldimport"
	"github.com/brian1917/workloader/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var pceList, skipSources, outputFileName string
var updatePCE, noPrompt bool

func init() {
	WkldReplicate.Flags().StringVarP(&pceList, "pce-list", "p", "", "comma-separated list of pce names (not fqdns). see workloader pce-list for options.")
	WkldReplicate.Flags().StringVarP(&skipSources, "skip-source", "s", "", "comma-separated list of pce names (not fqdns) to skip as a source. the pces still received workloads from other pces.")
	WkldReplicate.Flags().StringVar(&outputFileName, "output-file", "", "optionally specify the name of the output file location. default is current location with a timestamped filename. there will be a prefix added to each provided filename.")
}

// WkldReplicate runs the wkld-replicate command
var WkldReplicate = &cobra.Command{
	Use:   "wkld-replicate",
	Short: "Replicate workloads between multiple PCEs.",
	Long: `
Replicate workloads between multiple PCEs.

All PCEs must have the same label types. Any customer label types must be added to all PCEs.

Managed and unmanaged workloads are replicated across all PCEs. The command creates and deletes unmanaged workloads. Unmanaged workloads are deleted in the following scenarios:
1. The managed workload it was replicated from is unpaired.
2. The original unmanaged workload it was replicated from is deleted.`,
	Run: func(cmd *cobra.Command, args []string) {

		// Get the debug value from viper
		updatePCE = viper.Get("update_pce").(bool)
		noPrompt = viper.Get("no_prompt").(bool)
		wkldReplicate()
	},
}

type replicateWkld struct {
	pce      illumioapi.PCE
	workload illumioapi.Workload
}

func labelSlice(w illumioapi.Workload, pce illumioapi.PCE, labelKeys []string) (labelSlice []string) {
	for _, k := range labelKeys {
		label := w.GetLabelByKey(k, pce.Labels)
		if label.Key == "" {
			labelSlice = append(labelSlice, "wkld-replicate-remove")
		} else {
			labelSlice = append(labelSlice, label.Value)
		}
	}
	return labelSlice
}

func wkldReplicate() {

	// Create a slice to hold our target PCEs
	var pces []illumioapi.PCE

	// Create a map to to hold the PCE names to verify skip PCEs
	pceNameMap := make(map[string]bool)

	// Process the input PCEs
	utils.LogInfo("getting pces and labels...", true)
	for _, pce := range strings.Split(strings.Replace(pceList, " ", "", -1), ",") {
		p, err := utils.GetPCEbyName(pce, true)
		if err != nil {
			utils.LogError(err.Error())
		}
		pces = append(pces, p)
		pceNameMap[pce] = true
	}

	// Process PCEs that should be skiped as the source
	skipPCENameMap := make(map[string]bool)
	if skipSources != "" {
		for _, pce := range strings.Split(strings.Replace(skipSources, " ", "", -1), ",") {
			if !pceNameMap[pce] {
				utils.LogError("%s is not in the pce list. skipped pces must also be in the pce list")
			}
			skipPCENameMap[pce] = true
		}
	}

	// Create maps for workloads
	managedWkldMap := make(map[string]replicateWkld)
	unmanagedWkldMap := make(map[string]replicateWkld)

	// Determine if any PCEs don't support MT4L
	legacyPCE := false
	for _, p := range pces {
		if p.Version.Major < 22 || (p.Version.Major == 22 && p.Version.Minor < 5) {
			legacyPCE = true
			break
		}
	}

	// Get the label keys
	labelKeys := []string{}
	if !legacyPCE {
		labelDimensions, api, err := pces[0].GetLabelDimensions(nil)
		utils.LogAPIResp("GetLabelDimensions", api)
		if err != nil {
			utils.LogError(err.Error())
		}
		for _, ld := range labelDimensions {
			labelKeys = append(labelKeys, ld.Key)
		}
		utils.LogInfo(fmt.Sprintf("used %s to discover label dimensions: %s", pces[0].FriendlyName, strings.Join(labelKeys, ",")), true)
	} else {
		labelKeys = append(labelKeys, "role", "app", "env", "loc")
	}

	// Start the csv data
	wkldImportCsvData := [][]string{append(append([]string{"source", wkldexport.HeaderHostname, wkldexport.HeaderDescription}, labelKeys...), wkldexport.HeaderInterfaces, wkldexport.HeaderExternalDataSet, wkldexport.HeaderExternalDataReference)}
	wkldDeleteCsvdata := [][]string{{"href", "pce_fqdn", "pce_name"}}
	deleteHrefMap := make(map[string][]string)

	// Iterate through the PCEs and do initial processing of workloads
	for _, p := range pces {

		// If it's  a skip source, skip it
		if skipPCENameMap[p.FriendlyName] {
			continue
		}

		// Start the delete slice
		deleteHrefMap[p.FQDN] = []string{}

		// Get the workloads
		utils.LogInfo(fmt.Sprintf("getting workloads for %s (%s)", p.FriendlyName, p.FQDN), true)
		_, a, err := p.GetWklds(nil)
		utils.LogAPIResp("GetWkld", a)
		if err != nil {
			utils.LogError(err.Error())
		}

		// Reset counters
		managedWkldCnt := 0
		unmanagedWkldnt := 0
		unmanagedOwned := 0
		unmanagedNotOwned := 0

		// Iterate over all managed and unmanaged workloads separately
		for _, w := range p.WorkloadsSlice {
			if w.Hostname == "" {
				utils.LogError(fmt.Sprintf("%s - href: %s - name: %s - wkld-replicate requires hostnames on all workloads. one option to quickly fix is to use wkld-export, edit the csv to have unique hostnames, and use wkld-import to apply.", p.FQDN, w.Href, w.Name))
			}

			// Start with managed worklodas
			if w.GetMode() != "unmanaged" {
				// Put it in the map
				managedWkldMap[p.FQDN+w.Hostname] = replicateWkld{pce: p, workload: w}
				managedWkldCnt++

				// Edit the external data reference section
				w.ExternalDataSet = utils.StrToPtr("wkld-replicate")
				w.ExternalDataReference = utils.StrToPtr(p.FQDN + "-managed-wkld-" + w.Href)

				// Add to the CSV output
				newRow := append([]string{p.FriendlyName, w.Hostname, fmt.Sprintf("managed ven on %s", p.FQDN)}, labelSlice(w, p, labelKeys)...)
				newRow = append(newRow, strings.Join(wkldexport.InterfaceToString(w, true), ";"), utils.PtrToStr(w.ExternalDataSet), utils.PtrToStr(w.ExternalDataReference))
				wkldImportCsvData = append(wkldImportCsvData, newRow)
			}

			// Unmanaged - just put in the map. Needs additional processing below before being added to CSV slice.
			if w.GetMode() == "unmanaged" {
				unmanagedWkldnt++
				unmanagedWkldMap[p.FQDN+w.Hostname] = replicateWkld{pce: p, workload: w}
				if strings.Contains(utils.PtrToStr(w.ExternalDataReference), p.FQDN) || utils.PtrToStr(w.ExternalDataReference) == "" {
					unmanagedOwned++
				} else {
					unmanagedNotOwned++
				}
			}
		}
		// Log information
		utils.LogInfo(fmt.Sprintf("%s (%s) - workload counts:", p.FriendlyName, p.FQDN), true)
		utils.LogInfo(fmt.Sprintf("%d total workloads", len(p.WorkloadsSlice)), true)
		utils.LogInfo(fmt.Sprintf("%d managed workloads", managedWkldCnt), true)
		utils.LogInfo(fmt.Sprintf("%d unmanaged workloads (%d owned by this pce and %d not owned by this pce)", unmanagedWkldnt, unmanagedOwned, unmanagedNotOwned), true)
		utils.LogInfo(fmt.Sprintf("%d contributions (managed + unmanaged owned by this pce)", managedWkldCnt+unmanagedOwned), true)
		utils.LogInfo("------------------------------", true)
	}

	// Iterate through all the unmanaged workloads
	for _, wkld := range unmanagedWkldMap {
		// If it's not in the dataset yet, update the external data reference and add it to the csv
		if utils.PtrToStr(wkld.workload.ExternalDataSet) != "wkld-replicate" {
			wkld.workload.ExternalDataSet = utils.StrToPtr("wkld-replicate")
			wkld.workload.ExternalDataReference = utils.StrToPtr(wkld.pce.FQDN + "-unmanaged-wkld-" + wkld.workload.Href)
			newRow := append([]string{wkld.pce.FriendlyName, wkld.workload.Hostname, fmt.Sprintf("unmanaged workload on %s", wkld.pce.FQDN)}, labelSlice(wkld.workload, wkld.pce, labelKeys)...)
			newRow = append(newRow, strings.Join(wkldexport.InterfaceToString(wkld.workload, true), ";"), utils.PtrToStr(wkld.workload.ExternalDataSet), utils.PtrToStr(wkld.workload.ExternalDataReference))
			wkldImportCsvData = append(wkldImportCsvData, newRow)
			continue
		}

		// If we are here, the external dataset equals wkld-replicate and need to validate it still should exist

		// If it's ext data references shows it's owned by the same PCE, keep it.
		if wkld.pce.FQDN == strings.Split(utils.PtrToStr(wkld.workload.ExternalDataReference), "-unmanaged-wkld-")[0] {
			newRow := append([]string{wkld.pce.FriendlyName, wkld.workload.Hostname, fmt.Sprintf("unmanaged workload on %s", wkld.pce.FQDN)}, labelSlice(wkld.workload, wkld.pce, labelKeys)...)
			newRow = append(newRow, strings.Join(wkldexport.InterfaceToString(wkld.workload, true), ";"), utils.PtrToStr(wkld.workload.ExternalDataSet), utils.PtrToStr(wkld.workload.ExternalDataReference))
			wkldImportCsvData = append(wkldImportCsvData, newRow)
			continue
		}

		// If ext data reference shows it's a managed workload and that manage workload doesn't exist any more, remove it.
		if strings.Contains(utils.PtrToStr(wkld.workload.ExternalDataReference), "-managed-wkld-") {
			if _, exists := managedWkldMap[strings.Split(utils.PtrToStr(wkld.workload.ExternalDataReference), "-managed-wkld-")[0]+wkld.workload.Hostname]; !exists {
				wkldDeleteCsvdata = append(wkldDeleteCsvdata, []string{wkld.workload.Href, wkld.pce.FQDN, wkld.pce.FriendlyName})
				deleteHrefMap[wkld.pce.FQDN] = append(deleteHrefMap[wkld.pce.FQDN], wkld.workload.Href)
			}
			continue
		}

		// If the ext data reference shows it's owned by an unmanaged workload in a separate pce (already validated it's not same PCE above),
		if strings.Contains(utils.PtrToStr(wkld.workload.ExternalDataReference), "-unmanaged-wkld-") {
			if _, exists := unmanagedWkldMap[strings.Split(utils.PtrToStr(wkld.workload.ExternalDataReference), "-unmanaged-wkld-")[0]+wkld.workload.Hostname]; !exists {
				wkldDeleteCsvdata = append(wkldDeleteCsvdata, []string{wkld.workload.Href, wkld.pce.FQDN, wkld.pce.FriendlyName})
				deleteHrefMap[wkld.pce.FQDN] = append(deleteHrefMap[wkld.pce.FQDN], wkld.workload.Href)
			}
		}
	}

	// Export the wkld-import CSV
	var wkldCsvFileName string
	if len(wkldImportCsvData) > 1 {
		if outputFileName == "" {
			wkldCsvFileName = fmt.Sprintf("workloader-wkld-replicate-wkld-import-%s.csv", time.Now().Format("20060102_150405"))
		} else {
			wkldCsvFileName = "wkld-import-" + outputFileName
		}
		utils.WriteOutput(wkldImportCsvData, wkldImportCsvData, wkldCsvFileName)
		utils.LogInfo(fmt.Sprintf("%d workloads to be imported", len(wkldImportCsvData)-1), true)
	}

	// Export the wklds to delete
	var deleteCsvFileName string
	if len(wkldDeleteCsvdata) > 1 {
		if outputFileName == "" {
			deleteCsvFileName = fmt.Sprintf("workloader-wkld-replicate-wkld-delete-%s.csv", time.Now().Format("20060102_150405"))
		} else {
			deleteCsvFileName = "wkld-delete-" + outputFileName
		}
		utils.WriteOutput(wkldDeleteCsvdata, wkldDeleteCsvdata, deleteCsvFileName)
		utils.LogInfo(fmt.Sprintf("%d workloads to be deleted", len(wkldDeleteCsvdata)-1), true)
	}

	utils.LogInfo("------------------------------", true)

	// If updatePCE is disabled, we are just going to alert the user what will happen and log
	if !updatePCE {
		utils.LogInfo("see workloader.log for more details. to do the import, run again using --update-pce flag.", true)
		utils.LogEndCommand("wkld-replicate")
		return
	}

	// If updatePCE is set, but not noPrompt, we will prompt the user.
	if updatePCE && !noPrompt {
		var prompt string
		fmt.Printf("\r\n%s [PROMPT] - do you want to run the replicate (yes/no)? ", time.Now().Format("2006-01-02 15:04:05 "))
		fmt.Scanln(&prompt)
		if strings.ToLower(prompt) != "yes" {
			utils.LogInfo("prompt denied", true)
			utils.LogEndCommand("wkld-replicate")
			return
		}
	}

	// Run the actions against PCEs
	for _, p := range pces {
		if len(wkldImportCsvData) > 1 {
			utils.LogInfo(fmt.Sprintf("running wkld-import for %s (%s) with %s", p.FriendlyName, p.FQDN, wkldCsvFileName), true)
			wkldimport.ImportWkldsFromCSV(wkldimport.Input{
				PCE:             p,
				ImportFile:      wkldCsvFileName,
				RemoveValue:     "wkld-replicate-remove",
				Umwl:            true,
				UpdatePCE:       true,
				NoPrompt:        true,
				UpdateWorkloads: true,
			})
		}

		// Delete the hrefs
		if len(wkldDeleteCsvdata) > 1 {
			utils.LogInfo(fmt.Sprintf("running delete api for %s (%s)", p.FriendlyName, p.FQDN), true)
			for _, deleteHref := range deleteHrefMap[p.FQDN] {
				a, err := p.DeleteHref(deleteHref)
				utils.LogAPIResp("DeleteHref", a)
				if err != nil {
					utils.LogError(err.Error())
				}
				utils.LogInfo(fmt.Sprintf("%s is in %s delete - %d", deleteHref, p.FQDN, a.StatusCode), true)
			}
		}

		utils.LogInfo("------------------------------", true)
	}

	utils.LogEndCommand("wkld-replicate")

}
