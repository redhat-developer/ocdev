package describe

import (
	"encoding/json"
	"fmt"

	"github.com/openshift/odo/pkg/log"
	"github.com/openshift/odo/pkg/machineoutput"
	"github.com/openshift/odo/pkg/service"
	svc "github.com/openshift/odo/pkg/service"
	olm "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type operatorBackend struct {
	// the operator name
	OperatorType   string
	CustomResource string
	CSV            olm.ClusterServiceVersion
	CR             *olm.CRDDescription
}

func NewOperatorBackend() *operatorBackend {
	return &operatorBackend{}
}

func (ohb *operatorBackend) CompleteDescribeService(dso *DescribeServiceOptions, args []string) error {
	oprType, CR, err := service.SplitServiceKindName(args[0])
	if err != nil {
		return err
	}
	// we check if the cluster supports ClusterServiceVersion or not.
	isCSVSupported, err := svc.IsCSVSupported()
	if err != nil {
		// if there is an error checking it, we return the error.
		return err
	}
	// if its not supported then we return an error
	if !isCSVSupported {
		return errors.New("it seems the cluster doesn't support Operators. Please install OLM and try again")
	}
	ohb.OperatorType = oprType
	ohb.CustomResource = CR
	return nil
}

func (ohb *operatorBackend) ValidateDescribeService(dso *DescribeServiceOptions) error {
	var err error
	if ohb.OperatorType == "" || ohb.CustomResource == "" {
		return errors.New("invalid service name, use the format <operator-type>/<crd-name>")
	}
	// make sure that CSV of the specified OperatorType exists
	ohb.CSV, err = dso.KClient.GetClusterServiceVersion(ohb.OperatorType)
	if err != nil {
		// error only occurs when OperatorHub is not installed.
		// k8s does't have it installed by default but OCP does
		return err
	}

	// Get the specific CR that matches "kind"
	crs := dso.KClient.GetCustomResourcesFromCSV(&ohb.CSV)

	var cr *olm.CRDDescription
	hasCR := false
	for _, custRes := range *crs {
		c := custRes
		if c.Kind == ohb.CustomResource {
			cr = &c
			hasCR = true
			break
		}
	}
	if !hasCR {
		return fmt.Errorf("the %q resource doesn't exist in specified %q operator", ohb.CustomResource, ohb.OperatorType)
	}

	ohb.CR = cr
	return nil

}

func (ohb *operatorBackend) RunDescribeService(dso *DescribeServiceOptions) error {

	if dso.isExample {
		almExample, err := svc.GetAlmExample(ohb.CSV, ohb.CustomResource, ohb.OperatorType)
		if err != nil {
			return err
		}
		if log.IsJSON() {
			jsonExample := service.NewOperatorExample(almExample)
			jsonCR, err := json.MarshalIndent(jsonExample, "", "  ")
			if err != nil {
				return err
			}

			fmt.Println(string(jsonCR))

		} else {
			yamlCR, err := yaml.Marshal(almExample)
			if err != nil {
				return err
			}

			log.Info(string(yamlCR))
		}
		return nil
	}

	if log.IsJSON() {
		machineoutput.OutputSuccess(svc.ConvertCRDToJSONRepr(ohb.CR))
		return nil
	}
	output, err := yaml.Marshal(svc.ConvertCRDToRepr(ohb.CR))
	if err != nil {
		return err
	}

	fmt.Print(string(output))
	return nil
}
