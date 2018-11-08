package completion

import (
	"github.com/posener/complete"
	"github.com/redhat-developer/odo/pkg/application"
	"github.com/redhat-developer/odo/pkg/odo/genericclioptions"
	"github.com/redhat-developer/odo/pkg/service"
	"github.com/spf13/cobra"
)

// ServiceCompletionHandler provides service name completion for the current project and application
var ServiceCompletionHandler = func(cmd *cobra.Command, args complete.Args, context *genericclioptions.Context) (completions []string) {
	completions = make([]string, 0)

	services, err := service.List(context.Client, context.Application)
	if err != nil {
		return completions
	}

	for _, class := range services {
		completions = append(completions, class.Name)
	}

	return completions
}

// ServiceClassCompletionHandler provides catalog service class name completion
var ServiceClassCompletionHandler = func(cmd *cobra.Command, args complete.Args, context *genericclioptions.Context) (completions []string) {
	completions = make([]string, 0)
	services, err := context.Client.GetClusterServiceClasses()
	if err != nil {
		return completions
	}

	for _, class := range services {
		completions = append(completions, class.Spec.ExternalName)
	}

	return completions
}

// AppCompletionHandler provides completion for the app commands
var AppCompletionHandler = func(args complete.Args, client *occlient.Client) (completions []string) {
	completions = make([]string, 0)
	util.GetAndSetNamespace(client)

	applications, err := application.List(client)
	if err != nil {
		return completions
	}

	for _, app := range applications {
		completions = append(completions, app.Name)
	}
	return completions
}

// FileCompletionHandler provides suggestions for files and directories
var FileCompletionHandler = func(args complete.Args, client *occlient.Client) (completions []string) {
	completions = append(completions, complete.PredictFiles("*").Predict(args)...)
	return completions
}
