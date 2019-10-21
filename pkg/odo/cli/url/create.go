package url

import (
	"fmt"

	"github.com/golang/glog"

	clicomponent "github.com/openshift/odo/pkg/odo/cli/component"
	"github.com/openshift/odo/pkg/odo/genericclioptions/printtemplates"

	"github.com/openshift/odo/pkg/config"
	"github.com/openshift/odo/pkg/log"
	"github.com/openshift/odo/pkg/odo/genericclioptions"
	"github.com/openshift/odo/pkg/odo/util/completion"
	"github.com/openshift/odo/pkg/url"
	"github.com/pkg/errors"

	"github.com/openshift/odo/pkg/util"
	"github.com/spf13/cobra"

	ktemplates "k8s.io/kubernetes/pkg/kubectl/cmd/templates"
)

const createRecommendedCommandName = "create"

var (
	urlCreateShortDesc = `Create a URL for a component`
	urlCreateLongDesc  = ktemplates.LongDesc(`Create a URL for a component.

	The created URL can be used to access the specified component from outside the OpenShift cluster.
	`)
	urlCreateExample = ktemplates.Examples(`  # Create a URL with a specific name by automatically detecting the port used by the component
	%[1]s example

	# Create a URL for the current component with a specific port
	%[1]s --port 8080
  
	# Create a URL with a specific name and port
	%[1]s example --port 8080
	  `)
)

// URLCreateOptions encapsulates the options for the odo url create command
type URLCreateOptions struct {
	componentContext string
	urlName          string
	urlPort          int
	componentPort    int
	now              bool
	*clicomponent.CommonPushOptions
}

// NewURLCreateOptions creates a new UrlCreateOptions instance
func NewURLCreateOptions() *URLCreateOptions {
	return &URLCreateOptions{CommonPushOptions: clicomponent.NewCommonPushOptions()}
}

// Complete completes UrlCreateOptions after they've been Created
func (o *URLCreateOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	o.Context = genericclioptions.NewContext(cmd)
	o.LocalConfigInfo, err = config.NewLocalConfigInfo(o.componentContext)
	if err != nil {
		return err
	}
	o.componentPort, err = url.GetValidPortNumber(o.Component(), o.urlPort, o.LocalConfigInfo.GetPorts())
	if err != nil {
		return err
	}
	if len(args) == 0 {
		o.urlName = url.GetURLName(o.Component(), o.componentPort)
	} else {
		o.urlName = args[0]
	}
	if o.now {
		o.ResolveSrcAndConfigFlags()
		err = o.ResolveProject(o.Context.Project)
		if err != nil {
			return err
		}
	}
	return
}

// Validate validates the UrlCreateOptions based on completed values
func (o *URLCreateOptions) Validate() (err error) {

	// Check if exist
	for _, localUrl := range o.LocalConfigInfo.GetUrl() {
		if o.urlName == localUrl.Name {
			return fmt.Errorf("the url %s already exists in the application: %s", o.urlName, o.Application)
		}
	}

	// Check if url name is more than 63 characters long
	if len(o.urlName) > 63 {
		return fmt.Errorf("url name must be shorter than 63 characters")
	}

	if !util.CheckOutputFlag(o.OutputFlag) {
		return fmt.Errorf("given output format %s is not supported", o.OutputFlag)
	}

	return
}

// Run contains the logic for the odo url create command
func (o *URLCreateOptions) Run() (err error) {
	err = o.LocalConfigInfo.SetConfiguration("url", config.ConfigUrl{Name: o.urlName, Port: o.componentPort})
	if err != nil {
		return errors.Wrapf(err, "failed to persist the component settings to config file")
	}
	log.Successf("URL %s created for component: %v\n", o.urlName, o.Component())
	if o.now {
		o.Context, _, err = genericclioptions.UpdatedContext(o.Context)
		if o.LocalConfigInfo == nil {
			fmt.Println("Ooops local config is nil")
		}
		glog.V(4).Infof("Reloaded context info %#v", o)

		if err != nil {
			return errors.Wrap(err, "unable to retrieve updated local config")
		}
		err = o.SetSourceInfo()
		if err != nil {
			return errors.Wrap(err, "unable to set source information")
		}
		err = o.Push()
		if err != nil {
			return errors.Wrapf(err, "failed to push the changes")
		}
	} else {
		fmt.Print(printtemplates.PushMessage("create", "URL", false))
	}
	return nil
}

// NewCmdURLCreate implements the odo url create command.
func NewCmdURLCreate(name, fullName string) *cobra.Command {
	o := NewURLCreateOptions()
	urlCreateCmd := &cobra.Command{
		Use:     name + " [url name]",
		Short:   urlCreateShortDesc,
		Long:    urlCreateLongDesc,
		Example: fmt.Sprintf(urlCreateExample, fullName),
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			genericclioptions.GenericRun(o, cmd, args)
		},
	}
	urlCreateCmd.Flags().IntVarP(&o.urlPort, "port", "", -1, "port number for the url of the component, required in case of components which expose more than one service port")
	// Add `--now` flag
	genericclioptions.AddNowFlag(urlCreateCmd, &o.now)
	genericclioptions.AddContextFlag(urlCreateCmd, &o.componentContext)
	completion.RegisterCommandFlagHandler(urlCreateCmd, "context", completion.FileCompletionHandler)
	return urlCreateCmd
}
