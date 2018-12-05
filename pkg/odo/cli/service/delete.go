package service

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/redhat-developer/odo/pkg/odo/genericclioptions"
	"github.com/redhat-developer/odo/pkg/odo/util"
	"github.com/redhat-developer/odo/pkg/odo/util/completion"
	svc "github.com/redhat-developer/odo/pkg/service"
	"github.com/spf13/cobra"
	ktemplates "k8s.io/kubernetes/pkg/kubectl/cmd/templates"
	"strings"
)

const deleteRecommendedCommandName = "delete"

var (
	deleteExample = ktemplates.Examples(`
    # Delete the service named 'mysql-persistent'
    %[1]s mysql-persistent`)

	deleteLongDesc = ktemplates.LongDesc(`
List all services in the current application`)
)

type ServiceDeleteOptions struct {
	serviceForceDeleteFlag bool
	serviceName            string
	*genericclioptions.Context
}

// NewServiceDeleteOptions creates a new ServiceDeleteOptions instance
func NewServiceDeleteOptions() *ServiceDeleteOptions {
	return &ServiceDeleteOptions{}
}

// Complete completes ServiceDeleteOptions after they've been created
func (o *ServiceDeleteOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	o.Context = genericclioptions.NewContext(cmd)
	o.serviceName = args[0]

	return
}

// Validate validates the ServiceDeleteOptions based on completed values
func (o *ServiceDeleteOptions) Validate() (err error) {
	exists, err := svc.SvcExists(o.Client, o.serviceName, o.Application)
	if err != nil {
		return fmt.Errorf("unable to delete service because Service Catalog is not enabled in your cluster:\n%v", err)
	}
	if !exists {
		return fmt.Errorf("Service with the name %s does not exist in the current application\n", o.serviceName)
	}
	return
}

// Run contains the logic for the odo service delete command
func (o *ServiceDeleteOptions) Run() (err error) {
	var confirmDeletion string
	if o.serviceForceDeleteFlag {
		confirmDeletion = "y"
	} else {
		fmt.Printf("Are you sure you want to delete %v from %v? [y/N] ", o.serviceName, o.Application)
		_, _ = fmt.Scanln(&confirmDeletion)
	}
	if strings.ToLower(confirmDeletion) == "y" {
		err = svc.DeleteService(o.Client, o.serviceName, o.Application)
		if err != nil {
			return fmt.Errorf("unable to delete service %s:\n%v", o.serviceName, err)
		}
		fmt.Printf("Service %s from application %s has been deleted\n", o.serviceName, o.Application)
	} else {
		fmt.Printf("Aborting deletion of service: %v\n", o.serviceName)
	}
	return
}

// NewCmdServiceDelete implements the odo service delete command.
func NewCmdServiceDelete(name, fullName string) *cobra.Command {
	o := NewServiceDeleteOptions()
	serviceDeleteCmd := &cobra.Command{
		Use:     name + " <service_name>",
		Short:   "Delete an existing service",
		Long:    deleteLongDesc,
		Example: fmt.Sprintf(deleteExample, fullName),
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			glog.V(4).Infof("service delete called\n args: %#v", strings.Join(args, " "))
			util.CheckError(o.Complete(name, cmd, args), "")
			util.CheckError(o.Validate(), "")
			util.CheckError(o.Run(), "")
		},
	}
	serviceDeleteCmd.Flags().BoolVarP(&o.serviceForceDeleteFlag, "force", "f", false, "Delete service without prompting")
	completion.RegisterCommandHandler(serviceDeleteCmd, completion.ServiceCompletionHandler)
	return serviceDeleteCmd
}
