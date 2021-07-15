package validate

import "github.com/devfile/api/v2/pkg/apis/workspaces/v1alpha2"

func validatePreStart(preStart []string) (err error) {
	return nil
}

func validateEvents(events v1alpha2.Events) (err error) {

	if err := validatePreStart(events.PreStart); err != nil {
		return err
	}

	return nil
}
