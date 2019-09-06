# Workloader

## Description
Workloader is a tool that helps discover, label, and manage workloads in an Illumio PCE. Functionality includes:

## Installation

### Have Go installed?
Run `go get github.com/brian1917/workloader` or to update `go get -u github.com/brian1917/workloader`. This will download the repo and depenendies and install in your Go path to start using.

### Don't have Go installed?
Download the binary for your operating system from the `bin` folder of this repository. Direct links for each operating systems are here: [Mac](https://github.com/brian1917/workloader/raw/master/bin/workloader-mac), [Linux](https://github.com/brian1917/workloader/raw/master/bin/workloader-linux), and [Windows](https://github.com/brian1917/workloader/raw/master/bin/workloader-win.exe). No other installation required.


## Usage
`workloader -h`

```
Workloader is a tool that helps discover, label, and manage workloads in an Illumio PCE.

  Usage:
        workloader [command]

  Login Commands:
        login         Verifies existing login or generates a pce.yaml file for authentication used for all other commands.
        logout        Removes pce.yaml file and optionally removes all workloader generated API keys from PCE.

  Import/Export Commands:
        export        Create a CSV export of all workloads in the PCE.
        import        Create and assign labels to a workload from a CSV file. Use the --umwl flag to create and label unmanaged workloads from the same CSV.
        flowupload    Upload flows from CSV file to the PCE.
          
  Automated Labeling Commands:
        traffic       Find and label unmanaged workloads and label existing workloads based on Explorer traffic and an input CSV.
        subnet        Assign environment and location labels based on a workload's network.
        hostparse     Label workloads by parsing hostnames from provided regex functions.

  Workload Management Commands:
        compatibility Generate a compatibility report for all Idle workloads.
        mode          Change the state of workloads based on a CSV input.
        upgrade       Upgrade the VEN installed on workloads by labels or an input hostname list.

  Reporting Commands:
        mislabel      Display workloads that have no intra App-Group communications to identify potentially mislabled workloads.
        flowsummary   Summarize flows from explorer. Two subcommands: appgroup and env
        dupecheck     Identifies duplicate hostnames and IP addresses in the PCE.
  
Use "workloader [command] --help" for more information about a command.
```

Each command has its own help menu (e.g., `workloader login -h`)