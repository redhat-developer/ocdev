package completion

import (
	"github.com/posener/complete"
	"github.com/redhat-developer/odo/pkg/application"
	"github.com/redhat-developer/odo/pkg/odo/genericclioptions"
	"github.com/redhat-developer/odo/pkg/project"
	"github.com/redhat-developer/odo/pkg/service"
	"github.com/redhat-developer/odo/pkg/url"
	"github.com/spf13/cobra"
)

// ServiceCompletionHandler provides service name completion for the current project and application
var ServiceCompletionHandler = func(cmd *cobra.Command, args parsedArgs, context *genericclioptions.Context) (completions []string) {
	completions = make([]string, 0)

	services, err := service.List(context.Client, context.Application)
	if err != nil {
		return completions
	}

	for _, class := range services {
		completions = append(completions, class.Name)
	}

	return
}

// ServiceClassCompletionHandler provides catalog service class name completion
var ServiceClassCompletionHandler = func(cmd *cobra.Command, args parsedArgs, context *genericclioptions.Context) (completions []string) {
	completions = make([]string, 0)
	services, err := context.Client.GetClusterServiceClasses()
	if err != nil {
		return completions
	}

	for _, class := range services {
		completions = append(completions, class.Spec.ExternalName)
	}

	return
}

// AppCompletionHandler provides completion for the app commands
var AppCompletionHandler = func(cmd *cobra.Command, args parsedArgs, context *genericclioptions.Context) (completions []string) {
	completions = make([]string, 0)

	applications, err := application.List(context.Client)
	if err != nil {
		return completions
	}

	for _, app := range applications {
		completions = append(completions, app.Name)
	}
	return
}

// FileCompletionHandler provides suggestions for files and directories
var FileCompletionHandler = func(cmd *cobra.Command, args parsedArgs, context *genericclioptions.Context) (completions []string) {
	completions = append(completions, complete.PredictFiles("*").Predict(args.original)...)
	return
}

// ProjectNameCompletionHandler provides project name completion
var ProjectNameCompletionHandler = func(cmd *cobra.Command, args parsedArgs, context *genericclioptions.Context) (completions []string) {
	completions = make([]string, 0)
	projects, err := project.List(context.Client)
	if err != nil {
		return completions
	}

	for _, project := range projects {
		// we found the project name in the list which means
		// that the project name has been already selected by the user so no need to suggest more
		if val, ok := args.commands[project.Name]; ok && val {
			return nil
		}
		completions = append(completions, project.Name)
	}
	return completions
}

// URLCompletionHandler provides completion for the url commands
var URLCompletionHandler = func(cmd *cobra.Command, args parsedArgs, context *genericclioptions.Context) (completions []string) {
	completions = make([]string, 0)

	urls, err := url.List(context.Client, context.Component(), context.Application)
	if err != nil {
		return completions
	}

	for _, url := range urls {
		// we found the url in the list which means
		// that the url name has been already selected by the user so no need to suggest more
		if val, ok := args.commands[url.Name]; ok && val {
			return nil
		}
		completions = append(completions, url.Name)
	}
	return
}
