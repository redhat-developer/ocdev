/*
	This file contains code for various service backends supported by odo. Different backends have different logics for
	Complete, Validate and Run functions. These are covered in this file.
*/
package service

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/openshift/odo/pkg/log"
	svc "github.com/openshift/odo/pkg/service"
	olm "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// DynamicCRD holds the original CR obtained from the Operator (a CSV), or user
// (when they use --from-file flag), and few other attributes that are likely
// to be used to validate a CRD before creating a service from it
type DynamicCRD struct {
	// contains the CR as obtained from CSV or user
	OriginalCRD map[string]interface{}
}

func NewDynamicCRD() *DynamicCRD {
	return &DynamicCRD{}
}

// validateMetadataInCRD validates if the CRD has metadata.name field and returns an error
func (d *DynamicCRD) validateMetadataInCRD() error {
	metadata, ok := d.OriginalCRD["metadata"].(map[string]interface{})
	if !ok {
		// this condition is satisfied if there's no metadata at all in the provided CRD
		return fmt.Errorf("couldn't find \"metadata\" in the yaml; need metadata start the service")
	}

	if _, ok := metadata["name"].(string); ok {
		// found the metadata.name; no error
		return nil
	}
	return fmt.Errorf("couldn't find metadata.name in the yaml; provide a name for the service")
}

// setServiceName modifies the CRD to contain user provided name on the CLI
// instead of using the default one in almExample
func (d *DynamicCRD) setServiceName(name string) {
	metaMap := d.OriginalCRD["metadata"].(map[string]interface{})

	for k := range metaMap {
		if k == "name" {
			metaMap[k] = name
			return
		}
		// if metadata doesn't have 'name' field, we set it up
		metaMap["name"] = name
	}
}

// getServiceNameFromCRD fetches the service name from metadata.name field of the CRD
func (d *DynamicCRD) getServiceNameFromCRD() (string, error) {
	metadata, ok := d.OriginalCRD["metadata"].(map[string]interface{})
	if !ok {
		// this condition is satisfied if there's no metadata at all in the provided CRD
		return "", fmt.Errorf("couldn't find \"metadata\" in the yaml; need metadata.name to start the service")
	}

	if name, ok := metadata["name"].(string); ok {
		// found the metadata.name; no error
		return name, nil
	}
	return "", fmt.Errorf("couldn't find metadata.name in the yaml; provide a name for the service")
}

// This CompleteServiceCreate contains logic to complete the "odo service create" call for the case of Operator backend
func (b *OperatorBackend) CompleteServiceCreate(o *CreateOptions, cmd *cobra.Command, args []string) (err error) {
	// since interactive mode is not supported for Operators yet, set it to false
	o.interactive = false

	// if user has just used "odo service create", simply return
	if o.fromFile == "" && len(args) == 0 {
		return
	}

	// if user wants to create service from file and use a name given on CLI
	if o.fromFile != "" {
		if len(args) == 1 {
			o.ServiceName = args[0]
		}
		return
	}

	// split the name provided on CLI and populate servicetype & customresource
	o.ServiceType, b.CustomResource, err = svc.SplitServiceKindName(args[0])
	if err != nil {
		return fmt.Errorf("invalid service name, use the format <operator-type>/<crd-name>")
	}

	// if two args are given, first is service type and second one is service name
	if len(args) == 2 {
		o.ServiceName = args[1]
	}

	return nil
}

func (b *OperatorBackend) ValidateServiceCreate(o *CreateOptions) (err error) {
	d := NewDynamicCRD()
	// if the user wants to create service from a file, we check for
	// existence of file and validate if the requested operator and CR
	// exist on the cluster
	if o.fromFile != "" {
		if _, err := os.Stat(o.fromFile); err != nil {
			return errors.Wrap(err, "unable to find specified file")
		}

		// Parse the file to find Operator and CR info
		fileContents, err := ioutil.ReadFile(o.fromFile)
		if err != nil {
			return err
		}

		err = yaml.Unmarshal(fileContents, &d.OriginalCRD)
		if err != nil {
			return err
		}

		// Check if the operator and the CR exist on cluster
		var csv olm.ClusterServiceVersion
		b.CustomResource, csv, err = svc.GetCSV(o.KClient, d.OriginalCRD)
		if err != nil {
			return err
		}

		// all is well, let's populate the fields required for creating operator backed service
		b.group, b.version, b.resource, err = svc.GetGVRFromOperator(csv, b.CustomResource)
		if err != nil {
			return err
		}

		err = d.validateMetadataInCRD()
		if err != nil {
			return err
		}

		if o.ServiceName != "" && !o.DryRun {
			// First check if service with provided name already exists
			svcFullName := strings.Join([]string{b.CustomResource, o.ServiceName}, "/")
			exists, err := svc.OperatorSvcExists(o.KClient, svcFullName)
			if err != nil {
				return err
			}
			if exists {
				return fmt.Errorf("service %q already exists; please provide a different name or delete the existing service first", svcFullName)
			}

			d.setServiceName(o.ServiceName)
		} else {
			o.ServiceName, err = d.getServiceNameFromCRD()
			if err != nil {
				return err
			}
		}

		// CRD is valid. We can use it further to create a service from it.
		b.CustomResourceDefinition = d.OriginalCRD

		return nil
	} else if b.CustomResource != "" {
		// make sure that CSV of the specified ServiceType exists
		csv, err := o.KClient.GetClusterServiceVersion(o.ServiceType)
		if err != nil {
			// error only occurs when OperatorHub is not installed.
			// k8s does't have it installed by default but OCP does
			return err
		}

		almExample, err := svc.GetAlmExample(csv, b.CustomResource, o.ServiceType)
		if err != nil {
			return err
		}

		d.OriginalCRD = almExample

		b.group, b.version, b.resource, err = svc.GetGVRFromOperator(csv, b.CustomResource)
		if err != nil {
			return err
		}

		if o.ServiceName != "" && !o.DryRun {
			// First check if service with provided name already exists
			svcFullName := strings.Join([]string{b.CustomResource, o.ServiceName}, "/")
			exists, err := svc.OperatorSvcExists(o.KClient, svcFullName)
			if err != nil {
				return err
			}
			if exists {
				return fmt.Errorf("service %q already exists; please provide a different name or delete the existing service first", svcFullName)
			}

			d.setServiceName(o.ServiceName)
		}

		err = d.validateMetadataInCRD()
		if err != nil {
			return err
		}

		// CRD is valid. We can use it further to create a service from it.
		b.CustomResourceDefinition = d.OriginalCRD

		if o.ServiceName == "" {
			o.ServiceName, err = d.getServiceNameFromCRD()
			if err != nil {
				return err
			}
		}

		return nil
	} else {
		// This block is executed only when user has neither provided a
		// file nor a valid `odo service create <operator-name>` to start
		// the service from an Operator. So we raise an error because the
		// correct way is to execute:
		// `odo service create <operator-name>/<crd-name>`

		return fmt.Errorf("please use a valid command to start an Operator backed service; desired format: %q", "odo service create <operator-name>/<crd-name>")
	}
}

func (b *OperatorBackend) RunServiceCreate(o *CreateOptions) (err error) {
	s := &log.Status{}

	// in case of an Operator backed service, name of the service is
	// provided by the yaml specification in alm-examples. It might also
	// happen that a user wants to spin up Service Catalog based service in
	// spite of having 4.x cluster mode but we're not supporting
	// interacting with both Operator Hub and Service Catalog on 4.x. So
	// the user won't get to see service name in the log message
	if !o.DryRun {
		log.Infof("Deploying service %q of type: %q", o.ServiceName, b.CustomResource)
		s = log.Spinner("Deploying service")
		defer s.End(false)
	}

	// if cluster has resources of type CSV and o.CustomResource is not
	// empty, we're expected to create an Operator backed service
	if o.DryRun {
		// if it's dry run, only print the alm-example (o.CustomResourceDefinition) and exit
		jsonCR, err := json.MarshalIndent(b.CustomResourceDefinition, "", "  ")
		if err != nil {
			return err
		}

		// convert json to yaml
		yamlCR, err := yaml.JSONToYAML(jsonCR)
		if err != nil {
			return err
		}

		log.Info(string(yamlCR))

		return nil
	} else {
		err = svc.CreateOperatorService(o.KClient, b.group, b.version, b.resource, b.CustomResourceDefinition)
		if err != nil {
			// TODO: logic to remove CRD info from devfile because service creation failed.
			return err
		} else {
			s.End(true)
			log.Successf(`Service %q was created`, o.ServiceName)
		}

		crdYaml, err := yaml.Marshal(b.CustomResourceDefinition)
		if err != nil {
			return err
		}

		err = svc.AddKubernetesComponentToDevfile(string(crdYaml), o.ServiceName, o.EnvSpecificInfo.GetDevfileObj())
		if err != nil {
			return err
		}
	}
	s.End(true)

	return
}

func (b *OperatorBackend) ServiceExists(o *DeleteOptions) (bool, error) {
	return svc.OperatorSvcExists(o.KClient, o.serviceName)
}

func (b *OperatorBackend) DeleteService(o *DeleteOptions, name string, application string) error {
	err := svc.DeleteOperatorService(o.KClient, o.serviceName)
	if err != nil {
		return err
	}

	// "name" is of the form CR-Name/Instance-Name so we split it
	// we ignore the error because the function used below is called in the call to "DeleteOperatorService" above.
	_, instanceName, _ := svc.SplitServiceKindName(name)

	err = svc.DeleteKubernetesComponentFromDevfile(instanceName, o.EnvSpecificInfo.GetDevfileObj())
	if err != nil {
		return errors.Wrap(err, "failed to delete service from the devfile")
	}

	return nil
}
