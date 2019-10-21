package extract

import (
	"fmt"
	"os"
	"strings"

	"github.com/brian1917/illumioapi"
	"github.com/brian1917/workloader/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// PCE global variable
var pce illumioapi.PCE
var err error
var pStatus []string
var outDir string
var debug bool

// ExtractCmd extracts PCE objects
var ExtractCmd = &cobra.Command{
	Use:    "extract",
	Short:  "Extract PCE objects.",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {

		pce, err = utils.GetPCE(false)
		if err != nil {
			utils.Log(1, err.Error())
		}

		// Get the debug value from viper
		debug = viper.Get("debug").(bool)

		extract()
	},
}

func labels() {

	// Get all labels
	labels, lablesAPI, err := pce.GetAllLabels()
	if err != nil {
		utils.Log(1, err.Error())
	}

	// Create the file
	labelsFile, err := os.Create(fmt.Sprintf("%s/labels.json", outDir))
	if err != nil {
		utils.Log(1, err.Error())
	}

	// Write the file
	_, err = labelsFile.WriteString(lablesAPI.RespBody)
	if err != nil {
		utils.Log(1, err.Error())
	}
	// Close the file
	labelsFile.Close()

	// Update stdout
	fmt.Printf("Exported %d labels.\r\n", len(labels))
}

func workloads() {
	// Create directory
	os.Mkdir(fmt.Sprintf("%s/workloads", outDir), 0700)
	fmt.Println("Created temporary directory for extract.")

	// Start by getting all workloads
	wklds, _, err := pce.GetAllWorkloads()
	if err != nil {
		utils.Log(1, err.Error())
	}
	// Iterate through each workload
	for i, w := range wklds {
		// Get the workload so we can include service details that GetAllWorkloads does not have
		w, a, err := pce.GetWkldByHref(w.Href)
		if err != nil {
			utils.Log(1, err.Error())
		}
		// Create the file
		wkldFile, err := os.Create(fmt.Sprintf("%s/workloads/%s.json", outDir, strings.TrimPrefix(w.Href, fmt.Sprintf("/orgs/%d/workloads/", pce.Org))))
		if err != nil {
			utils.Log(1, err.Error())
		}
		// Write the file
		_, err = wkldFile.WriteString(a.RespBody)
		if err != nil {
			utils.Log(1, err.Error())
		}
		// CLose the file
		wkldFile.Close()
		// Update progress
		fmt.Printf("\rExported %d of %d workloads (%d%%).", i, len(wklds), i*100/len(wklds))
	}
	// Update stdout
	fmt.Printf("\r                                                      ")
	fmt.Printf("\rExported %d workloads.\r\n", len(wklds))
}

func services() {
	for _, p := range pStatus {
		// Reset the services API and then call it for each provision status
		servicesAPI := illumioapi.APIResponse{}
		svcs, servicesAPI, err := pce.GetAllServices(p)
		if err != nil {
			utils.Log(1, err.Error())
		}
		// Create the file
		servicesFile, err := os.Create(fmt.Sprintf("%s/%s_services.json", outDir, p))
		if err != nil {
			utils.Log(1, err.Error())
		}
		// Write the file
		_, err = servicesFile.WriteString(servicesAPI.RespBody)
		if err != nil {
			utils.Log(1, err.Error())
		}
		// Close the file
		servicesFile.Close()
		//Update
		fmt.Printf("Exported %d %s services.\r\n", len(svcs), p)
	}
}

func ipLists() {
	for _, p := range pStatus {
		// Reset the services API and then call it for each provision status
		ipListAPI := illumioapi.APIResponse{}
		ipLists := []illumioapi.IPList{}
		if p == "draft" {
			ipLists, ipListAPI, err = pce.GetAllDraftIPLists()
			if err != nil {
				utils.Log(1, err.Error())
			}
		} else {
			ipLists, ipListAPI, err = pce.GetAllActiveIPLists()
			if err != nil {
				utils.Log(1, err.Error())
			}
		}
		if len(ipLists) > 0 {
			// Create the file
			ipListsFile, err := os.Create(fmt.Sprintf("%s/%s_iplists.json", outDir, p))
			if err != nil {
				utils.Log(1, err.Error())
			}
			// Write the file
			_, err = ipListsFile.WriteString(ipListAPI.RespBody)
			if err != nil {
				utils.Log(1, err.Error())
			}
			//Update
			fmt.Printf("Exported %d %s IP Lists.\r\n", len(ipLists), p)
			// Close file
			ipListsFile.Close()
		} else {
			fmt.Printf("No %s IP lists to export.\r\n", p)
		}
	}
}

func virtualServices() {
	for _, p := range pStatus {
		// Reset the services API and then call it for each provision status
		vsAPI := illumioapi.APIResponse{}
		vs, vsAPI, err := pce.GetAllVirtualServices(p)
		if err != nil {
			utils.Log(1, err.Error())
		}

		if len(vs) > 0 {
			// Create the file
			virtualServicesFile, err := os.Create(fmt.Sprintf("%s/%s_virtualservices.json", outDir, p))
			if err != nil {
				utils.Log(1, err.Error())
			}
			// Write the file
			_, err = virtualServicesFile.WriteString(vsAPI.RespBody)
			if err != nil {
				utils.Log(1, err.Error())
			}
			// Close the file
			virtualServicesFile.Close()
			//Update
			fmt.Printf("Exported %d %s virtual services.\r\n", len(vs), p)
		} else {
			fmt.Printf("No %s virtual services to export.\r\n", p)
		}
	}
}

func labelGroups() {
	for _, p := range pStatus {
		// Reset the services API and then call it for each provision status
		lgAPI := illumioapi.APIResponse{}
		lg, lgAPI, err := pce.GetAllLabelGroups(p)
		if err != nil {
			utils.Log(1, err.Error())
		}

		if len(lg) > 0 {
			// Create the file
			lgFile, err := os.Create(fmt.Sprintf("%s/%s_labelgroups.json", outDir, p))
			if err != nil {
				utils.Log(1, err.Error())
			}
			// Write the file
			_, err = lgFile.WriteString(lgAPI.RespBody)
			if err != nil {
				utils.Log(1, err.Error())
			}
			// Close the file
			lgFile.Close()
			//Update
			fmt.Printf("Exported %d %s label groups.\r\n", len(lg), p)
		} else {
			fmt.Printf("No %s label groups to export.\r\n", p)
		}
	}
}

func ruleSets() {
	for _, p := range pStatus {
		// Reset the services API and then call it for each provision status
		rsAPI := illumioapi.APIResponse{}
		rs, rsAPI, err := pce.GetAllRuleSets(p)
		if err != nil {
			utils.Log(1, err.Error())
		}

		if len(rs) > 0 {
			// Create the file
			rsFile, err := os.Create(fmt.Sprintf("%s/%s_rulesets.json", outDir, p))
			if err != nil {
				utils.Log(1, err.Error())
			}
			// Write the file
			_, err = rsFile.WriteString(rsAPI.RespBody)
			if err != nil {
				utils.Log(1, err.Error())
			}
			// Close the file
			rsFile.Close()
			//Update
			fmt.Printf("Exported %d %s rulesets.\r\n", len(rs), p)
		} else {
			fmt.Printf("No %s rulesets to export.\r\n", p)
		}
	}
}

func traffic() {
	t, err := pce.IterateTrafficJString(true)
	if err != nil {
		utils.Log(1, err.Error())
	}

	if len(t) > 0 {
		// Create the file
		tFile, err := os.Create(fmt.Sprintf("%s/traffic.json", outDir))
		if err != nil {
			utils.Log(1, err.Error())
		}
		// Write the file
		_, err = tFile.WriteString(t)
		if err != nil {
			utils.Log(1, err.Error())
		}
		// Close the file
		tFile.Close()
	} else {
		fmt.Println("No traffic to export.")
	}
}

func extract() {

	// Log start of command
	utils.Log(0, "extract command started")

	// Set outdir
	outDir = "pce-extract"

	// Log output directory
	d, err := os.Getwd()
	if err != nil {
		utils.Log(1, err.Error())
	}
	fullPathOutDir := fmt.Sprintf("%s%s%s", d, string(os.PathSeparator), outDir)
	utils.Log(0, fmt.Sprintf("temp pce-extract folder set to %s", fullPathOutDir))

	// Check if directory exists and remove it
	if _, err := os.Stat(outDir); os.IsNotExist(err) {
		utils.Log(0, fmt.Sprintf("%s does not already exist. creating it.", fullPathOutDir))
	} else {
		utils.Log(0, fmt.Sprintf("%s exists. removing it and creating new.", fullPathOutDir))
		err := os.RemoveAll(outDir)
		if err != nil {
			utils.Log(1, err.Error())
		}
	}

	// Make the directory for the extract
	if err := os.Mkdir(outDir, 0700); err != nil {
		utils.Log(1, err.Error())
	}
	utils.Log(0, fmt.Sprintf("created %s", fullPathOutDir))

	// Set provision status for objects that require it
	pStatus = []string{"draft", "active"}

	// Extract objects
	workloads()
	labels()
	services()
	ipLists()
	virtualServices()
	labelGroups()
	ruleSets()
	traffic()

	// Zip the extract folder
	zipit(outDir, "pce-extract.zip")

	fmt.Println("pce-extract.zip created.")
	utils.Log(0, fmt.Sprintf("%s%spce-extract.zip created", fullPathOutDir, string(os.PathSeparator)))

	// Remove the created directory
	err = os.RemoveAll(outDir)
	if err != nil {
		fmt.Println(err)
	}
	utils.Log(0, fmt.Sprintf("%s removed", fullPathOutDir))

	fmt.Println("Temporary directory removed.")

	// Log start of command
	utils.Log(0, "extract command completed")

}