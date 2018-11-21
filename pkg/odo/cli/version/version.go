package version

import (
	"fmt"
	"github.com/redhat-developer/odo/pkg/notify"
	"github.com/redhat-developer/odo/pkg/odo/genericclioptions"
	"github.com/redhat-developer/odo/pkg/odo/util"
	"os"
	"strings"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

var (
	// VERSION  is version number that will be displayed when running ./odo version
	VERSION = "v0.0.16"

	// GITCOMMIT is hash of the commit that wil be displayed when running ./odo version
	// this will be overwritten when running  build like this: go build -ldflags="-X github.com/redhat-developer/odo/cmd.GITCOMMIT=$(GITCOMMIT)"
	// HEAD is default indicating that this was not set during build
	GITCOMMIT = "HEAD"
)

var clientFlag bool

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the client version information",
	Long:  "Print the client version information",
	Example: `  # Print the client version of Odo
  odo version
	`,
	Run: func(cmd *cobra.Command, args []string) {

		// If verbose mode is enabled, dump all KUBECLT_* env variables
		// this is usefull for debuging oc plugin integration
		for _, v := range os.Environ() {
			if strings.HasPrefix(v, "KUBECTL_") {
				glog.V(4).Info(v)
			}
		}

		fmt.Println("odo " + VERSION + " (" + GITCOMMIT + ")")

		if !clientFlag {
			// Lets fetch the info about the server
			serverInfo, err := genericclioptions.ClientWithConnectionCheck(cmd, true).GetServerVersion()
			util.CheckError(err, "")
			// make sure we only include Openshift info if we actually have it
			openshiftStr := ""
			if len(serverInfo.OpenShiftVersion) > 0 {
				openshiftStr = fmt.Sprintf("OpenShift: %v\n", serverInfo.OpenShiftVersion)
			}
			fmt.Printf("\n"+
				"Server: %v\n"+
				"%v"+
				"Kubernetes: %v\n",
				serverInfo.Address,
				openshiftStr,
				serverInfo.KubernetesVersion)
		}
	},
}

func NewCmdVersion() *cobra.Command {
	// Add a defined annotation in order to appear in the help menu
	versionCmd.Annotations = map[string]string{"command": "utility"}
	versionCmd.SetUsageTemplate(util.CmdUsageTemplate)
	versionCmd.Flags().BoolVar(&clientFlag, "client", false, "Client version only (no server required).")

	return versionCmd
}
func GetLatestReleaseInfo(info chan<- string) {
	newTag, err := notify.CheckLatestReleaseTag(VERSION)
	if err != nil {
		// The error is intentionally not being handled because we don't want
		// to stop the execution of the program because of this failure
		glog.V(4).Infof("Error checking if newer odo release is available: %v", err)
	}
	if len(newTag) > 0 {
		info <- "---\n" +
			"A newer version of odo (version: " + fmt.Sprint(newTag) + ") is available.\n" +
			"Update using your package manager, or run\n" +
			"curl " + notify.InstallScriptURL + " | sh\n" +
			"to update manually, or visit https://github.com/redhat-developer/odo/releases\n" +
			"---\n" +
			"If you wish to disable the update notifications, you can disable it by running\n" +
			"'odo utils config set UpdateNotification false'\n"
	}
}
