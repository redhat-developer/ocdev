package genericclioptions

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/openshift/odo/pkg/version"

	"github.com/openshift/odo/pkg/odo/cli/ui"
	"gopkg.in/AlecAivazis/survey.v1"

	"github.com/openshift/odo/pkg/preference"
	"github.com/openshift/odo/pkg/segment"
	scontext "github.com/openshift/odo/pkg/segment/context"
	"k8s.io/klog"

	"github.com/openshift/odo/pkg/log"
	"github.com/openshift/odo/pkg/odo/util"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Runnable interface {
	Complete(name string, cmd *cobra.Command, args []string) error
	Validate() error
	Run(cmd *cobra.Command) error
}

func GenericRun(o Runnable, cmd *cobra.Command, args []string) {
	var err error
	var startTime time.Time
	cfg, _ := preference.New()
	disableTelemetry, _ := strconv.ParseBool(os.Getenv(segment.DisableTelemetryEnv))

	// Prompt the user to consent for telemetry if a value is not set already
	// Skip prompting if the preference command is called
	// This prompt has been placed here so that it does not prompt the user when they call --help

	if !cfg.IsSet(preference.ConsentTelemetrySetting) && cmd.Parent().Name() != "preference" {
		if !segment.RunningInTerminal() {
			klog.V(4).Infof("Skipping telemetry question because there is no terminal (tty)\n")
		} else if disableTelemetry {
			klog.V(4).Infof("Skipping telemetry question due to %s=%t\n", segment.DisableTelemetryEnv, disableTelemetry)
		} else {
			var consentTelemetry bool
			prompt := &survey.Confirm{Message: "Help odo improve by allowing it to collect usage data. Read about our privacy statement: https://developers.redhat.com/article/tool-data-collection. You can change your preference later by changing the ConsentTelemetry preference.", Default: false}
			err = survey.AskOne(prompt, &consentTelemetry, nil)
			ui.HandleError(err)
			if err == nil {
				if err1 := cfg.SetConfiguration(preference.ConsentTelemetrySetting, strconv.FormatBool(consentTelemetry)); err1 != nil {
					klog.V(4).Info(err1.Error())
				}
			}
		}
	}
	// set value for telemetry status in context so that we do not need to call IsTelemetryEnabled every time to check its status
	scontext.SetTelemetryStatus(cmd.Context(), segment.IsTelemetryEnabled(cfg))

	startTime = time.Now()

	// CheckMachineReadableOutput
	// fixes / checks all related machine readable output functions
	CheckMachineReadableOutputCommand(cmd)

	// LogErrorAndExit is used so that we get -o (jsonoutput) for cmds which have json output implemented
	util.LogErrorAndExit(checkConflictingFlags(cmd, args), "")
	// Run completion, validation and run.
	// Only upload data to segment for completion and validation if a non-nil error is returned.
	err = o.Complete(cmd.Name(), cmd, args)
	if err != nil {
		startTelemetry(cmd, err, startTime)
	}
	util.LogErrorAndExit(err, "")

	err = o.Validate()
	if err != nil {
		startTelemetry(cmd, err, startTime)
	}
	util.LogErrorAndExit(err, "")

	err = o.Run(cmd)
	startTelemetry(cmd, err, startTime)
	util.LogErrorAndExit(err, "")
}

// startTelemetry uploads the data to segment if user has consented to usage data collection and the command is not telemetry
// TODO: move this function to a more suitable place, preferably pkg/segment
func startTelemetry(cmd *cobra.Command, err error, startTime time.Time) {
	if scontext.GetTelemetryStatus(cmd.Context()) && !strings.Contains(cmd.CommandPath(), "telemetry") {
		uploadData := &segment.TelemetryData{
			Event: cmd.CommandPath(),
			Properties: segment.TelemetryProperties{
				Duration:      time.Since(startTime).Milliseconds(),
				Success:       err == nil,
				Tty:           segment.RunningInTerminal(),
				Version:       fmt.Sprintf("odo %v (%v)", version.VERSION, version.GITCOMMIT),
				CmdProperties: scontext.GetContextProperties(cmd.Context()),
			},
		}
		if err != nil {
			uploadData.Properties.Error = segment.SetError(err)
			uploadData.Properties.ErrorType = segment.ErrorType(err)
		}
		data, err1 := json.Marshal(uploadData)
		if err1 != nil {
			klog.V(4).Infof("Failed to marshall telemetry data. %q", err1.Error())
		}
		command := exec.Command(os.Args[0], "telemetry", string(data))
		if err1 = command.Start(); err1 != nil {
			klog.V(4).Infof("Failed to start the telemetry process. Error: %q", err1.Error())
			return
		}
		if err1 = command.Process.Release(); err1 != nil {
			klog.V(4).Infof("Failed to release the process. %q", err1.Error())
			return
		}
	}
}

// checkConflictingFlags checks for conflicting flags. Currently --context cannot be provided
// with either --app, --project and --component as that information can be fetched from the local
// config.
func checkConflictingFlags(cmd *cobra.Command, args []string) error {

	// we allow providing --context with --app and --project in case of `odo create` or `odo component create`
	if cmd.Name() == "create" {
		if cmd.HasParent() {
			if cmd.Parent().Name() == "odo" || cmd.Parent().Name() == "component" {
				return nil
			}
		}
	}
	app := stringFlagLookup(cmd, "app")
	project := stringFlagLookup(cmd, "project")
	context := stringFlagLookup(cmd, "context")
	component := stringFlagLookup(cmd, "component")
	all, _ := strconv.ParseBool(stringFlagLookup(cmd, "all"))
	// TODO: Move this to a method under DeleteOptions, similar to CreateOptions.checkConflictingFlags
	if cmd.Name() == "delete" {
		if cmd.HasParent() {
			if cmd.Parent().Name() == "odo" || cmd.Parent().Name() == "component" {
				var componentName string
				if len(args) > 0 {
					componentName = args[0]
				}
				if (context != "") && (project != "" || componentName != "") {
					return fmt.Errorf("cannot provide --project or component name when --context is provided")
				}
				if project != "" && componentName == "" && app == "" {
					return fmt.Errorf("cannot provide --project without --app and component name")
				}
				if all && ((componentName != "" && app != "" && project != "") || (componentName != "")) {
					return fmt.Errorf("cannot provide --all when component name, --app and --project are provided")
				}
			}
		}
	}
	if (context != "") && (app != "" || project != "" || component != "") {
		return fmt.Errorf("cannot provide --app, --project or --component flag when --context is provided")
	}
	return nil
}

func stringFlagLookup(cmd *cobra.Command, flagName string) string {
	flag := cmd.Flags().Lookup(flagName)
	// a check to make sure if the flag is not defined we return blank
	if flag == nil {
		return ""
	}
	return flag.Value.String()
}

// CheckMachineReadableOutputCommand performs machine-readable output functions required to
// have it work correctly
func CheckMachineReadableOutputCommand(cmd *cobra.Command) {

	// Get the needed values
	outputFlag := pflag.Lookup("o")
	hasFlagChanged := outputFlag != nil && outputFlag.Changed
	machineOutput := cmd.Annotations["machineoutput"]

	// Check the valid output
	if hasFlagChanged && outputFlag.Value.String() != "json" {
		log.Error("Please input a valid output format for -o, available format: json")
		os.Exit(1)
	}

	// Check that if -o json has been passed, that the command actually USES json.. if not, error out.
	if hasFlagChanged && outputFlag.Value.String() == "json" && machineOutput == "" {

		// By default we "disable" logging, so activate it so that the below error can be shown.
		_ = flag.Set("o", "")

		// Output the error
		log.Error("Machine readable output is not yet implemented for this command")
		os.Exit(1)
	}

	// Before running anything, we will make sure that no verbose output is made
	// This is a HACK to manually override `-v 4` to `-v 0` (in which we have no klog.V(0) in our code...
	// in order to have NO verbose output when combining both `-o json` and `-v 4` so json output
	// is not malformed / mixed in with normal logging
	if log.IsJSON() {
		_ = flag.Set("v", "0")
	} else {
		// Override the logging level by the value (if set) by the ODO_LOG_LEVEL env
		// The "-v" flag set on command line will take precedence over ODO_LOG_LEVEL env
		v := flag.CommandLine.Lookup("v").Value.String()
		if level, ok := os.LookupEnv("ODO_LOG_LEVEL"); ok && v == "0" {
			_ = flag.CommandLine.Set("v", level)
		}
	}
}
