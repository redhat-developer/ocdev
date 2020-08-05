package parser

import (
	"fmt"
	"strings"

	"github.com/openshift/odo/pkg/application/labels"
	"github.com/openshift/odo/pkg/config"
	"github.com/openshift/odo/pkg/devfile/parser/data/common"
	"github.com/openshift/odo/pkg/util"
	"github.com/pkg/errors"
)

// This file contains all the parameters that can be change via odo config

const (
	Name              = "Name"
	Ports             = "Ports"
	Memory            = "Memory"
	PortsDescription  = "Ports to be opened in all component containers"
	MemoryDescription = "The Maximum memory all the component containers can consume"
	NameDescription   = "The name of the component"
)

var (
	supportedDevfileParameterDescriptions = map[string]string{
		Name:   NameDescription,
		Ports:  PortsDescription,
		Memory: MemoryDescription,
	}

	lowerCaseDevfileParameters = util.GetLowerCaseParameters(GetDevfileSupportedParameters())
)

func (d DevfileObj) SetConfiguration(parameter string, value interface{}) error {

	strValue, _ := value.(string)
	if parameter, ok := AsDevfileSupportedParameter(parameter); ok {
		switch parameter {
		case "name":
			return d.SetMetadataName(strValue)
		case "ports":
			arrValue := strings.Split(strValue, ",")
			return d.SetPorts(arrValue...)
		case "memory":
			return d.SetMemory(strValue)
		}

	}
	return errors.Errorf("unknown parameter :'%s' is not a configurable parameter in the devfile", parameter)

}

func (d DevfileObj) DeleteConfiguration(parameter string) error {
	if parameter, ok := AsDevfileSupportedParameter(parameter); ok {
		switch parameter {
		case "name":
			return d.SetMetadataName("")
		case "ports":
			return d.RemovePorts()
		case "memory":
			return d.SetMemory("")
		}
	}
	return errors.Errorf("unknown parameter :'%s' is not a configurable parameter in the devfile", parameter)
}

func (d DevfileObj) IsSet(parameter string) bool {

	if parameter, ok := AsDevfileSupportedParameter(parameter); ok {
		switch parameter {
		case "name":
			return d.GetMetadataName() != ""
		case "ports":
			return d.hasPorts()
		case "memory":
			return d.GetMemory() != ""
		}
	}
	return false

}

func (d DevfileObj) SetMetadataName(name string) error {
	metadata := d.Data.GetMetadata()
	d.Data.SetMetadata(name, metadata.Version)
	return d.WriteYamlDevfile()
}

func (d DevfileObj) AddEnvVars(otherList config.EnvVarList) error {
	components := d.Data.GetComponents()
	for _, component := range components {
		if component.Container != nil {
			currentlist := config.NewEnvVarListFromDevfileEnv(component.Container.Env)
			component.Container.Env = currentlist.Merge(otherList).ToDevfileEnv()
			d.Data.UpdateComponent(component)
		}
	}
	return d.WriteYamlDevfile()
}

func (d DevfileObj) RemoveEnvVars(keys []string) error {
	components := d.Data.GetComponents()
	for _, component := range components {
		if component.Container != nil {

			currentlist := config.NewEnvVarListFromDevfileEnv(component.Container.Env)
			envList, err := config.RemoveEnvVarsFromList(currentlist, keys)
			if err != nil {
				return err
			}
			component.Container.Env = envList.ToDevfileEnv()
			d.Data.UpdateComponent(component)
		}
	}
	return d.WriteYamlDevfile()
}

func (d DevfileObj) SetPorts(ports ...string) error {
	components := d.Data.GetComponents()
	endpoints, err := portsToEndpoints(ports...)
	if err != nil {
		return err
	}
	for _, component := range components {
		if component.Container != nil {
			component.Container.Endpoints = endpoints
			d.Data.UpdateComponent(component)
		}
	}
	return d.WriteYamlDevfile()
}

func (d DevfileObj) RemovePorts() error {
	components := d.Data.GetComponents()
	for _, component := range components {
		if component.Container != nil {
			component.Container.Endpoints = nil
			d.Data.UpdateComponent(component)
		}
	}
	return d.WriteYamlDevfile()
}

func (d DevfileObj) hasPorts() bool {
	components := d.Data.GetComponents()
	for _, component := range components {
		if len(component.Container.Endpoints) > 0 {
			return true
		}
	}
	return false
}

func (d DevfileObj) SetMemory(memory string) error {
	components := d.Data.GetComponents()
	for _, component := range components {
		if component.Container != nil {
			component.Container.MemoryLimit = memory
			d.Data.UpdateComponent(component)
		}
	}
	return d.WriteYamlDevfile()
}
func (d DevfileObj) GetMemory() string {
	components := d.Data.GetComponents()
	for _, component := range components {
		if component.Container.MemoryLimit != "" {
			return component.Container.MemoryLimit
		}
	}
	return ""
}

func (d DevfileObj) GetMetadataName() string {
	return d.Data.GetMetadata().Name
}

// FormatDevfileSupportedParameters outputs supported parameters and their description
func FormatDevfileSupportedParameters() (result string) {
	for _, v := range GetDevfileSupportedParameters() {
		result = result + v + " - " + supportedDevfileParameterDescriptions[v] + "\n"
	}
	return "\nAvailable Devfile Parameters:\n" + result
}

// AsDevfileSupportedParameter returns the parameter in lower case and a boolean indicating if it is a supported parameter
func AsDevfileSupportedParameter(param string) (string, bool) {
	lower := strings.ToLower(param)
	return lower, lowerCaseDevfileParameters[lower]
}

// GetDevfileSupportedParameters returns the name of the supported global parameters
func GetDevfileSupportedParameters() []string {
	return util.GetSortedKeys(supportedDevfileParameterDescriptions)
}

func portsToEndpoints(ports ...string) ([]common.Endpoint, error) {
	var endpoints []common.Endpoint
	conPorts, err := util.GetContainerPortsFromStrings(ports)
	if err != nil {
		return nil, err
	}
	for _, port := range conPorts {

		endpoint := common.Endpoint{
			// this is added to differentiate between endpoint created by the user vs devfile creator
			Attributes: map[string]string{
				labels.OdoManagedBy: "odo",
			},
			Name:       fmt.Sprintf("port-%d", port.ContainerPort),
			TargetPort: port.ContainerPort,
		}
		endpoints = append(endpoints, endpoint)
	}
	return endpoints, nil

}
