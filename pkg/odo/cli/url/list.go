package url

import (
	"fmt"
	"os"
	"text/tabwriter"

	routev1 "github.com/openshift/api/route/v1"
	clicomponent "github.com/openshift/odo/pkg/odo/cli/component"
	odoutil "github.com/openshift/odo/pkg/odo/util"

	"github.com/openshift/odo/pkg/localConfigProvider"
	"github.com/openshift/odo/pkg/log"
	"github.com/openshift/odo/pkg/machineoutput"
	"github.com/openshift/odo/pkg/odo/genericclioptions"
	"github.com/openshift/odo/pkg/odo/util/completion"
	"github.com/openshift/odo/pkg/url"
	"github.com/spf13/cobra"
	ktemplates "k8s.io/kubectl/pkg/util/templates"
)

const listRecommendedCommandName = "list"

var (
	urlListShortDesc = `List URLs`
	urlListLongDesc  = ktemplates.LongDesc(`Lists all the available URLs which can be used to access the components.`)
	urlListExample   = ktemplates.Examples(` # List the available URLs
  %[1]s
	`)
)

// ListOptions encapsulates the options for the odo url list command
type ListOptions struct {
	componentContext string
	*genericclioptions.Context
	client url.Client
}

// NewURLListOptions creates a new URLCreateOptions instance
func NewURLListOptions() *ListOptions {
	return &ListOptions{}
}

// Complete completes ListOptions after they've been Listed
func (o *ListOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	o.Context, err = genericclioptions.New(genericclioptions.CreateParameters{
		Cmd:              cmd,
		DevfilePath:      clicomponent.DevfilePath,
		ComponentContext: o.componentContext,
	})

	if err != nil {
		return err
	}

	routeSupported, err := o.Context.Client.IsRouteSupported()
	if err != nil {
		return err
	}

	o.client = url.NewClient(url.ClientOptions{
		LocalConfigProvider: o.Context.LocalConfigProvider,
		OCClient:            *o.Context.Client,
		IsRouteSupported:    routeSupported,
	})
	return
}

// Validate validates the ListOptions based on completed values
func (o *ListOptions) Validate() (err error) {
	return odoutil.CheckOutputFlag(o.OutputFlag)
}

// Run contains the logic for the odo url list command
func (o *ListOptions) Run(cmd *cobra.Command) (err error) {
	componentName := o.Context.LocalConfigProvider.GetName()
	urls, err := o.client.List()
	if err != nil {
		return err
	}
	if log.IsJSON() {
		machineoutput.OutputSuccess(urls)
	} else {
		if len(urls.Items) == 0 {
			return fmt.Errorf("no URLs found for component %v. Refer `odo url create -h` to add one", componentName)
		}

		log.Infof("Found the following URLs for component %v", componentName)
		tabWriterURL := tabwriter.NewWriter(os.Stdout, 5, 2, 3, ' ', tabwriter.TabIndent)
		fmt.Fprintln(tabWriterURL, "NAME", "\t", "STATE", "\t", "URL", "\t", "PORT", "\t", "SECURE", "\t", "KIND")

		// are there changes between local and cluster states?
		outOfSync := false
		for _, u := range urls.Items {
			if u.Spec.Kind == localConfigProvider.ROUTE {
				fmt.Fprintln(tabWriterURL, u.Name, "\t", u.Status.State, "\t", url.GetURLString(u.Spec.Protocol, u.Spec.Host, "", o.Context.LocalConfigInfo.Exists()), "\t", u.Spec.Port, "\t", u.Spec.Secure, "\t", u.Spec.Kind)
			} else {
				fmt.Fprintln(tabWriterURL, u.Name, "\t", u.Status.State, "\t", url.GetURLString(url.GetProtocol(routev1.Route{}, url.ConvertIngressURLToIngress(u, o.EnvSpecificInfo.GetName())), "", u.Spec.Host, false), "\t", u.Spec.Port, "\t", u.Spec.Secure, "\t", u.Spec.Kind)
			}
			if u.Status.State != url.StateTypePushed {
				outOfSync = true
			}
		}
		tabWriterURL.Flush()
		if outOfSync {
			log.Info("There are local changes. Please run 'odo push'.")
		}
	}

	return
}

// NewCmdURLList implements the odo url list command.
func NewCmdURLList(name, fullName string) *cobra.Command {
	o := NewURLListOptions()
	urlListCmd := &cobra.Command{
		Use:         name,
		Short:       urlListShortDesc,
		Long:        urlListLongDesc,
		Example:     fmt.Sprintf(urlListExample, fullName),
		Args:        cobra.NoArgs,
		Annotations: map[string]string{"machineoutput": "json"},
		Run: func(cmd *cobra.Command, args []string) {
			genericclioptions.GenericRun(o, cmd, args)
		},
	}
	genericclioptions.AddContextFlag(urlListCmd, &o.componentContext)
	completion.RegisterCommandFlagHandler(urlListCmd, "context", completion.FileCompletionHandler)

	return urlListCmd
}
