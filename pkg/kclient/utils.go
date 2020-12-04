package kclient

import (
	"strconv"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

// GetInputEnvVarsFromStrings generates corev1.EnvVar values from the array of string key=value pairs
// envVars is the array containing the key=value pairs
func GetInputEnvVarsFromStrings(envVars []string) ([]corev1.EnvVar, error) {
	var inputEnvVars []corev1.EnvVar
	var keys = make(map[string]int)
	for _, env := range envVars {
		splits := strings.SplitN(env, "=", 2)
		if len(splits) < 2 {
			return nil, errors.New("invalid syntax for env, please specify a VariableName=Value pair")
		}
		_, ok := keys[splits[0]]
		if ok {
			return nil, errors.Errorf("multiple values found for VariableName: %s", splits[0])
		}

		keys[splits[0]] = 1

		inputEnvVars = append(inputEnvVars, corev1.EnvVar{
			Name:  splits[0],
			Value: splits[1],
		})
	}
	return inputEnvVars, nil
}

// getErrorMessageFromEvents generates a error message from the given events
func getErrorMessageFromEvents(failedEvents map[string]corev1.Event) strings.Builder {
	// Create an output table
	tableString := &strings.Builder{}
	table := tablewriter.NewWriter(tableString)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")

	// Header
	table.SetHeader([]string{"Name", "Count", "Reason", "Message"})

	// List of events
	for name, event := range failedEvents {
		table.Append([]string{name, strconv.Itoa(int(event.Count)), event.Reason, event.Message})
	}

	// Here we render the table as well as a helpful error message
	table.Render()

	return *tableString
}
