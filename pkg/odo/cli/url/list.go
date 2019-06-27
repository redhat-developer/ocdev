package url

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/openshift/odo/pkg/odo/genericclioptions/printtemplates"

	"github.com/openshift/odo/pkg/config"
	"github.com/openshift/odo/pkg/log"
	"github.com/openshift/odo/pkg/odo/genericclioptions"
	"github.com/openshift/odo/pkg/odo/util"
	"github.com/openshift/odo/pkg/odo/util/completion"
	"github.com/openshift/odo/pkg/url"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	ktemplates "k8s.io/kubernetes/pkg/kubectl/cmd/templates"
)

const listRecommendedCommandName = "list"

var (
	urlListShortDesc = `List URLs`
	urlListLongDesc  = ktemplates.LongDesc(`Lists all the available URLs which can be used to access the components.`)
	urlListExample   = ktemplates.Examples(` # List the available URLs
  %[1]s
	`)
)

// URLListOptions encapsulates the options for the odo url list command
type URLListOptions struct {
	localConfigInfo  *config.LocalConfigInfo
	componentContext string
	*genericclioptions.Context
}

// NewURLListOptions creates a new UrlCreateOptions instance
func NewURLListOptions() *URLListOptions {
	return &URLListOptions{}
}

// Complete completes UrlListOptions after they've been Listed
func (o *URLListOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	o.Context = genericclioptions.NewContext(cmd)
	o.localConfigInfo, err = config.NewLocalConfigInfo(o.componentContext)
	if err != nil {
		return errors.Wrap(err, "failed intiating local config")
	}
	return
}

// Validate validates the UrlListOptions based on completed values
func (o *URLListOptions) Validate() (err error) {
	return util.CheckOutputFlag(o.OutputFlag)
}

// Run contains the logic for the odo url list command
func (o *URLListOptions) Run() (err error) {

	urls, err := url.List(o.Client, o.Component(), o.Application)
	if err != nil {
		return err
	}

	localUrls := o.localConfigInfo.GetUrl()

	if o.OutputFlag == "json" {
		out, err := json.Marshal(urls)
		if err != nil {
			return err
		}
		fmt.Println(string(out))
	} else {

		if len(urls.Items) == 0 && len(localUrls) == 0 {
			return fmt.Errorf("no URLs found for component %v in application %v", o.Component(), o.Application)
		} else {
			// cm=create message dm=delete message. If these flags are set then push message needs to be printed
			var cm, dm bool
			log.Infof("Found the following URLs for component %v in application %v:", o.Component(), o.Application)
			tabWriterURL := tabwriter.NewWriter(os.Stdout, 5, 2, 3, ' ', tabwriter.TabIndent)
			//create headers
			fmt.Fprintln(tabWriterURL, "NAME", "\t", "IN CONFIG", "\t", "URL", "\t", "PORT")

			// Check everything in config and match with URL
			for _, i := range localUrls {
				var present bool
				for _, u := range urls.Items {
					if i.Name == u.Name {
						fmt.Fprintln(tabWriterURL, u.Name, "\t", "Present", "\t", url.GetURLString(u.Spec.Protocol, u.Spec.Host), "\t", u.Spec.Port)
						present = true
					}
				}
				if !present {
					cm = true
					fmt.Fprintln(tabWriterURL, i.Name, "\t", "Present", "\t", "<not created on cluster>", "\t", i.Port)
				}
			}

			// Now reverse, check urls not in config and print their information
			// Hence need to be deleted
			for _, u := range urls.Items {
				// Search if url is present in local config
				var found bool
				for _, i := range localUrls {
					if i.Name == u.Name {
						found = true
						break
					}
				}
				// If not found then we need to print its info but say its not in config
				if !found {
					dm = true
					fmt.Fprintln(tabWriterURL, u.Name, "\t", "Absent", "\t", url.GetURLString(u.Spec.Protocol, u.Spec.Host), "\t", u.Spec.Port)
				}
			}
			tabWriterURL.Flush()
			if cm && dm {
				fmt.Print(printtemplates.PushMessage("create/delete", "URLs"))
			} else if cm {
				fmt.Print(printtemplates.PushMessage("create", "URLs"))
			} else if dm {
				fmt.Print(printtemplates.PushMessage("delete", "URLs"))
			}
		}

	}
	return
}

// NewCmdURLList implements the odo url list command.
func NewCmdURLList(name, fullName string) *cobra.Command {
	o := NewURLListOptions()
	urlListCmd := &cobra.Command{
		Use:     name,
		Short:   urlListShortDesc,
		Long:    urlListLongDesc,
		Example: fmt.Sprintf(urlListExample, fullName),
		Args:    cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			genericclioptions.GenericRun(o, cmd, args)
		},
	}
	genericclioptions.AddOutputFlag(urlListCmd)
	genericclioptions.AddContextFlag(urlListCmd, &o.componentContext)
	completion.RegisterCommandFlagHandler(urlListCmd, "context", completion.FileCompletionHandler)

	return urlListCmd
}
