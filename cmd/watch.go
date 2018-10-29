package cmd

import (
	"fmt"
	util2 "github.com/redhat-developer/odo/pkg/odo/util"
	"net/url"
	"os"
	"runtime"

	"github.com/redhat-developer/odo/pkg/application"
	"github.com/redhat-developer/odo/pkg/component"
	"github.com/redhat-developer/odo/pkg/project"
	"github.com/redhat-developer/odo/pkg/util"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

var (
	ignores []string
	delay   int
)

var watchCmd = &cobra.Command{
	Use:   "watch [component name]",
	Short: "Watch for changes, update component on change",
	Long:  `Watch for changes, update component on change.`,
	Example: `  # Watch for changes in directory for current component
  odo watch

  # Watch for changes in directory for component called frontend 
  odo watch frontend
	`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		stdout := os.Stdout
		client := util2.GetOcClient()
		projectName := project.GetCurrent(client)
		applicationName, err := application.GetCurrent(client)
		util2.CheckError(err, "Unable to get current application.")

		var componentName string
		if len(args) == 0 {
			var err error
			glog.V(4).Info("No component name passed, assuming current component")
			componentName, err = component.GetCurrent(client, applicationName, projectName)
			util2.CheckError(err, "")
			if componentName == "" {
				fmt.Println("No component is set as active.")
				fmt.Println("Use 'odo component set <component name> to set and existing component as active or call this command with component name as and argument.")
				os.Exit(1)
			}
		} else {
			componentName = args[0]
		}

		sourceType, sourcePath, err := component.GetComponentSource(client, componentName, applicationName, projectName)
		util2.CheckError(err, "Unable to get source for %s component.", componentName)

		if sourceType != "binary" && sourceType != "local" {
			fmt.Printf("Watch is supported by binary and local components only and source type of component %s is %s\n", componentName, sourceType)
			os.Exit(1)
		}

		u, err := url.Parse(sourcePath)
		util2.CheckError(err, "Unable to parse source %s from component %s.", sourcePath, componentName)

		if u.Scheme != "" && u.Scheme != "file" {
			fmt.Printf("Component %s has invalid source path %s.", componentName, u.Scheme)
			os.Exit(1)
		}
		watchPath := util.ReadFilePath(u, runtime.GOOS)

		err = component.WatchAndPush(client, componentName, applicationName, watchPath, stdout, ignores, delay)
		util2.CheckError(err, "Error while trying to watch %s", watchPath)
	},
}

func init() {
	// ignore git as it can change even if no source file changed
	// for example some plugins providing git info in PS1 doing that
	watchCmd.Flags().StringSliceVar(&ignores, "ignore", []string{".*\\.git.*"}, "Files or folders to be ignored via regular expressions.")
	watchCmd.Flags().IntVar(&delay, "delay", 1, "Time in seconds between a detection of code change and push.delay=0 means changes will be pushed as soon as they are detected which can cause performance issues")
	// Add a defined annotation in order to appear in the help menu
	watchCmd.Annotations = map[string]string{"command": "component"}
	watchCmd.SetUsageTemplate(cmdUsageTemplate)

	rootCmd.AddCommand(watchCmd)
}
