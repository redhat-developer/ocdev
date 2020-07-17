package common

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/openshift/odo/pkg/devfile/parser/data"
	"github.com/openshift/odo/pkg/devfile/parser/data/common"
	"k8s.io/klog"
)

// getCommand iterates through the devfile commands and returns the devfile command associated with the group
// commands mentioned via the flags are passed via commandName, empty otherwise
func getCommand(data data.DevfileData, commandName string, groupType common.DevfileCommandGroupType) (supportedCommand common.DevfileCommand, err error) {

	var command common.DevfileCommand

	if commandName == "" {
		command, err = getCommandFromDevfile(data, groupType)
	} else if commandName != "" {
		command, err = getCommandFromFlag(data, groupType, commandName)
	}

	return command, err
}

// getCommandFromDevfile iterates through the devfile commands and returns the command associated with the group
func getCommandFromDevfile(data data.DevfileData, groupType common.DevfileCommandGroupType) (supportedCommand common.DevfileCommand, err error) {
	commands := data.GetCommands()
	var onlyCommand common.DevfileCommand

	// validate the command groups before searching for a command match
	// if the command groups are invalid, err out
	// we only validate when the push command flags are absent,
	// since we know the command kind from the push flags
	err = validateCommandsForGroup(data, groupType)
	if err != nil {
		return common.DevfileCommand{}, err
	}

	for _, command := range commands {
		cmdGroup := command.GetGroup()
		if cmdGroup != nil && cmdGroup.Kind == groupType {
			if cmdGroup.IsDefault {
				return command, validateCommand(data, command)
			} else if reflect.DeepEqual(onlyCommand, common.DevfileCommand{}) {
				// return the only remaining command for the group if there is no default command
				// NOTE: we return outside the for loop since the next iteration can have a default command
				onlyCommand = command
			}
		}
	}

	// if default command is not found return the first command found for the matching type.
	if !reflect.DeepEqual(onlyCommand, common.DevfileCommand{}) {
		return onlyCommand, validateCommand(data, onlyCommand)
	}

	msg := fmt.Sprintf("the command group of kind \"%v\" is not found in the devfile", groupType)
	// if run command or test command is not found in devfile then it is an error
	if groupType == common.RunCommandGroupType || groupType == common.TestCommandGroupType {
		err = fmt.Errorf(msg)
	} else {
		klog.V(4).Info(msg)
	}

	return
}

// getCommandFromFlag iterates through the devfile commands and returns the command specified associated with the group
func getCommandFromFlag(data data.DevfileData, groupType common.DevfileCommandGroupType, commandName string) (supportedCommand common.DevfileCommand, err error) {
	commands := data.GetCommands()

	for _, command := range commands {
		if command.GetID() == commandName {

			// Update Group only custom commands (specified by odo flags)
			command = updateCommandGroupIfReqd(groupType, command)

			// we have found the command with name, its groupType Should match to the flag
			// e.g --build-command "mybuild"
			// exec:
			//   id: mybuild
			//   group:
			//     kind: build
			cmdGroup := command.GetGroup()
			if cmdGroup != nil && cmdGroup.Kind != groupType {
				return command, fmt.Errorf("command group mismatched, command %s is of group %v in devfile.yaml", commandName, command.Exec.Group.Kind)
			}

			return command, validateCommand(data, command)
		}
	}

	// if any command specified via flag is not found in devfile then it is an error.
	err = fmt.Errorf("the command \"%v\" is not found in the devfile", commandName)

	return
}

// validateCommandsForGroup validates the commands in a devfile for a group
// 1. multiple commands belonging to a single group should have at least one default
// 2. multiple commands belonging to a single group cannot have more than one default
func validateCommandsForGroup(data data.DevfileData, groupType common.DevfileCommandGroupType) error {

	commands := getCommandsByGroup(data, groupType)

	defaultCommandCount := 0

	if len(commands) > 1 {
		for _, command := range commands {
			if command.GetGroup().IsDefault {
				defaultCommandCount++
			}
		}
	} else {
		// if there is only one command, it is the default command for the group
		defaultCommandCount = 1
	}

	if defaultCommandCount == 0 {
		return fmt.Errorf("there should be exactly one default command for command group %v, currently there is no default command", groupType)
	} else if defaultCommandCount > 1 {
		return fmt.Errorf("there should be exactly one default command for command group %v, currently there is more than one default command", groupType)
	}

	return nil
}

// validateCommand validates the given command
// 1. command has to be of type exec or composite
// 2. component should be present
// 4. command must have group
func validateCommand(data data.DevfileData, command common.DevfileCommand) (err error) {

	// type must be exec or composite
	if command.Exec == nil && command.Composite == nil {
		return fmt.Errorf("command must be of type \"exec\" or \"composite\"")
	}

	// If the command is a composite command, need to validate that it is valid
	if command.Composite != nil {
		return validateCompositeCommand(data, command.Composite)
	}

	// component must be specified
	if command.Exec.Component == "" {
		return fmt.Errorf("exec commands must reference a component")
	}

	// must specify a command
	if command.Exec.CommandLine == "" {
		return fmt.Errorf("exec commands must have a command")
	}

	// must map to a supported component
	components := GetSupportedComponents(data)

	isComponentValid := false
	for _, component := range components {
		if command.Exec.Component == component.Container.Name {
			isComponentValid = true
		}
	}
	if !isComponentValid {
		return fmt.Errorf("the command does not map to a supported component")
	}

	return
}

// validateCompositeCommand checks that the specified composite command is valid
func validateCompositeCommand(data data.DevfileData, compositeCommand *common.Composite) error {
	if compositeCommand.Group != nil && compositeCommand.Group.Kind == common.RunCommandGroupType {
		return fmt.Errorf("composite commands of run Kind are not supported currently")
	}

	commandsMap := GetCommandsMap(data.GetCommands())

	// Loop over the commands and validate that each command points to a command that's in the devfile
	for _, command := range compositeCommand.Commands {
		if strings.ToLower(command) == compositeCommand.Id {
			return fmt.Errorf("the composite command %q cannot reference itself", compositeCommand.Id)
		}

		_, ok := commandsMap[strings.ToLower(command)]
		if !ok {
			return fmt.Errorf("the command %q mentioned in the composite command %q does not exist in the devfile", command, compositeCommand.Id)
		}
	}
	return nil
}

// GetInitCommand iterates through the components in the devfile and returns the init command
func GetInitCommand(data data.DevfileData, devfileInitCmd string) (initCommand common.DevfileCommand, err error) {

	return getCommand(data, devfileInitCmd, common.InitCommandGroupType)
}

// GetBuildCommand iterates through the components in the devfile and returns the build command
func GetBuildCommand(data data.DevfileData, devfileBuildCmd string) (buildCommand common.DevfileCommand, err error) {

	return getCommand(data, devfileBuildCmd, common.BuildCommandGroupType)
}

// GetDebugCommand iterates through the components in the devfile and returns the debug command
func GetDebugCommand(data data.DevfileData, devfileDebugCmd string) (debugCommand common.DevfileCommand, err error) {
	return getCommand(data, devfileDebugCmd, common.DebugCommandGroupType)
}

// GetRunCommand iterates through the components in the devfile and returns the run command
func GetRunCommand(data data.DevfileData, devfileRunCmd string) (runCommand common.DevfileCommand, err error) {

	return getCommand(data, devfileRunCmd, common.RunCommandGroupType)
}

// GetTestCommand iterates through the components in the devfile and returns the test command
func GetTestCommand(data data.DevfileData, devfileTestCmd string) (runCommand common.DevfileCommand, err error) {

	return getCommand(data, devfileTestCmd, common.TestCommandGroupType)
}

// ValidateAndGetPushDevfileCommands validates the build and the run command,
// if provided through odo push or else checks the devfile for devBuild and devRun.
// It returns the build and run commands if its validated successfully, error otherwise.
func ValidateAndGetPushDevfileCommands(data data.DevfileData, devfileInitCmd, devfileBuildCmd, devfileRunCmd string) (commandMap PushCommandsMap, err error) {
	var emptyCommand common.DevfileCommand
	commandMap = NewPushCommandMap()

	isInitCommandValid, isBuildCommandValid, isRunCommandValid := false, false, false

	initCommand, initCmdErr := GetInitCommand(data, devfileInitCmd)

	isInitCmdEmpty := reflect.DeepEqual(emptyCommand, initCommand)
	if isInitCmdEmpty && initCmdErr == nil {
		// If there was no init command specified through odo push and no default init command in the devfile, default validate to true since the init command is optional
		isInitCommandValid = true
		klog.V(4).Infof("No init command was provided")
	} else if !isInitCmdEmpty && initCmdErr == nil {
		isInitCommandValid = true
		commandMap[common.InitCommandGroupType] = initCommand
		klog.V(4).Infof("Init command: %v", initCommand.GetID())
	}

	buildCommand, buildCmdErr := GetBuildCommand(data, devfileBuildCmd)

	isBuildCmdEmpty := reflect.DeepEqual(emptyCommand, buildCommand)
	if isBuildCmdEmpty && buildCmdErr == nil {
		// If there was no build command specified through odo push and no default build command in the devfile, default validate to true since the build command is optional
		isBuildCommandValid = true
		klog.V(4).Infof("No build command was provided")
	} else if !reflect.DeepEqual(emptyCommand, buildCommand) && buildCmdErr == nil {
		isBuildCommandValid = true
		commandMap[common.BuildCommandGroupType] = buildCommand
		klog.V(4).Infof("Build command: %v", buildCommand.GetID())
	}

	runCommand, runCmdErr := GetRunCommand(data, devfileRunCmd)
	if runCmdErr == nil && !reflect.DeepEqual(emptyCommand, runCommand) {
		isRunCommandValid = true
		commandMap[common.RunCommandGroupType] = runCommand
		klog.V(4).Infof("Run command: %v", runCommand.GetID())
	}

	// If either command had a problem, return an empty list of commands and an error
	if !isInitCommandValid || !isBuildCommandValid || !isRunCommandValid {
		commandErrors := ""
		if initCmdErr != nil {
			commandErrors += fmt.Sprintf("\n%s", initCmdErr.Error())
		}
		if buildCmdErr != nil {
			commandErrors += fmt.Sprintf("\n%s", buildCmdErr.Error())
		}
		if runCmdErr != nil {
			commandErrors += fmt.Sprintf("\n%s", runCmdErr.Error())
		}
		return commandMap, fmt.Errorf(commandErrors)
	}

	return commandMap, nil
}

// Need to update group on custom commands specified by odo flags
func updateCommandGroupIfReqd(groupType common.DevfileCommandGroupType, command common.DevfileCommand) common.DevfileCommand {
	// Update Group only for exec commands
	// Update Group only when Group is not nil, devfile v2 might contain group for custom commands.
	if command.Exec != nil && command.Exec.Group == nil {
		command.Exec.Group = &common.Group{Kind: groupType}
		return command
	}
	return command
}

// ValidateAndGetDebugDevfileCommands validates the debug command
func ValidateAndGetDebugDevfileCommands(data data.DevfileData, devfileDebugCmd string) (pushDebugCommand common.DevfileCommand, err error) {
	var emptyCommand common.DevfileCommand

	isDebugCommandValid := false
	debugCommand, debugCmdErr := GetDebugCommand(data, devfileDebugCmd)
	if debugCmdErr == nil && !reflect.DeepEqual(emptyCommand, debugCommand) {
		isDebugCommandValid = true
		klog.V(4).Infof("Debug command: %v", debugCommand.Exec.Id)
	}

	if !isDebugCommandValid {
		commandErrors := ""
		if debugCmdErr != nil {
			commandErrors += debugCmdErr.Error()
		}
		return common.DevfileCommand{}, fmt.Errorf(commandErrors)
	}

	return debugCommand, nil
}

// ValidateAndGetTestDevfileCommands validates the test command
func ValidateAndGetTestDevfileCommands(data data.DevfileData, devfileTestCmd string) (testCommand common.DevfileCommand, err error) {
	var emptyCommand common.DevfileCommand
	isTestCommandValid := false
	testCommand, testCmdErr := GetTestCommand(data, devfileTestCmd)
	if testCmdErr == nil && !reflect.DeepEqual(emptyCommand, testCommand) {
		isTestCommandValid = true
		klog.V(4).Infof("Test command: %v", testCommand.GetID())
	}

	if !isTestCommandValid && testCmdErr != nil {
		return common.DevfileCommand{}, testCmdErr
	}

	return testCommand, nil
}
