package validate

import (
	devfilev1 "github.com/devfile/api/v2/pkg/apis/workspaces/v1alpha2"
)

// validateCommands validates the devfile commands:
// 1. checks if its either an exec or composite command
// 2. checks if the composite is a non run kind command
func validateCommands(commandsMap map[string]devfilev1.Command) (err error) {

	for _, command := range commandsMap {
		err = validateCommand(command)
		if err != nil {
			return err
		}
	}

	return
}

// validateCommand validates the given command
// 1. command has to be of type exec or composite,
// 2. if composite command, it should not be of kind run
func validateCommand(command devfilev1.Command) (err error) {

	// devfile command type for odo must be exec, apply or composite
	if command.Exec == nil && command.Apply == nil && command.Composite == nil {
		return &UnsupportedOdoCommandError{commandId: command.Id}
	}

	// If the command is a composite command, need to validate that it is valid
	if command.Composite != nil {
		return validateCompositeCommand(command)
	}

	return
}

// validateCompositeCommand checks that the specified composite command is valid in odo ie; it should not be of kind run
func validateCompositeCommand(command devfilev1.Command) error {
	if command.Composite.Group != nil && command.Composite.Group.Kind == devfilev1.RunCommandGroupKind {
		return &CompositeRunKindError{}
	}

	return nil
}
