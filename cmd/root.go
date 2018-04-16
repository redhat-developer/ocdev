package cmd

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/redhat-developer/odo/pkg/notify"
	"github.com/redhat-developer/odo/pkg/occlient"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// Global variables
var (
	GlobalVerbose bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "odo",
	Short: "OpenShift CLI for Developers",
	Long:  `OpenShift CLI for Developers`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {

		// Add extra logging when verbosity is passed
		if GlobalVerbose {
			//TODO
			log.SetLevel(log.DebugLevel)
		}

	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {

	updateInfo := make(chan string)
	go getLatestReleaseInfo(updateInfo)

	checkError(rootCmd.Execute(), "")

	select {
	case message := <-updateInfo:
		fmt.Printf(message)
	default:
		log.Debug("Could not get the latest release information in time. Never mind, exiting gracefully :)")
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.odo.yaml)")

	rootCmd.PersistentFlags().BoolVarP(&GlobalVerbose, "verbose", "v", false, "Verbose output")
}

func getLatestReleaseInfo(info chan<- string) {
	newTag, err := notify.CheckLatestReleaseTag(VERSION)
	if err != nil {
		// The error is intentionally not being handled because we don't want
		// to stop the execution of the program because of this failure
		log.Debugf("Error checking if newer odo release is available: %v", err)
	}
	if len(newTag) > 0 {
		info <- "---\n" +
			"A newer version of odo (version: " + fmt.Sprint(newTag) + ") is available.\n" +
			"Update using your package manager, or run\n" +
			"curl " + notify.InstallScriptURL + " | sh\n" +
			"to update manually, or visit https://github.com/redhat-developer/odo/releases\n" +
			"---\n"
	}
}

func getOcClient() *occlient.Client {
	client, err := occlient.New()
	checkError(err, "")
	return client
}

// checkError prints the cause of the given error and exits the code with an
// exit code of 1.
// If the context is provided, then that is printed, if not, then the cause is
// detected using errors.Cause(err)
func checkError(err error, context string, a ...interface{}) {
	if err != nil {
		log.Debugf("Error:\n%v", err)
		if context == "" {
			fmt.Println(errors.Cause(err))
		} else {
			fmt.Printf(fmt.Sprintf("%s\n", context), a...)
		}

		os.Exit(1)
	}
}
