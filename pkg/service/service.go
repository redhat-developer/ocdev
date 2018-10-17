package service

import (
	"strings"

	"sort"

	"github.com/pkg/errors"
	applabels "github.com/redhat-developer/odo/pkg/application/labels"
	componentlabels "github.com/redhat-developer/odo/pkg/component/labels"
	"github.com/redhat-developer/odo/pkg/occlient"
	"github.com/redhat-developer/odo/pkg/util"
)

// ServiceInfo holds all important information about one service
type ServiceInfo struct {
	Name   string
	Type   string
	Status string
}

// ListCatalog lists all the available service types
func ListCatalog(client *occlient.Client) ([]occlient.Service, error) {

	clusterServiceClasses, err := client.GetClusterServiceClassExternalNamesAndPlans()
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get cluster serviceClassExternalName")
	}

	// Sorting service classes alphabetically
	// Reference: https://golang.org/pkg/sort/#example_Slice
	sort.Slice(clusterServiceClasses, func(i, j int) bool {
		return clusterServiceClasses[i].Name < clusterServiceClasses[j].Name
	})

	return clusterServiceClasses, nil
}

// Search searches for the services
func Search(client *occlient.Client, name string) ([]occlient.Service, error) {
	var result []occlient.Service
	serviceList, err := ListCatalog(client)
	if err != nil {
		return nil, errors.Wrap(err, "unable to list services")
	}

	// do a partial search in all the services
	for _, service := range serviceList {
		if strings.Contains(service.Name, name) {
			result = append(result, service)
		}
	}

	return result, nil
}

// CreateService creates new service from serviceCatalog
func CreateService(client *occlient.Client, serviceName string, serviceType string, servicePlan string, parameters []string, applicationName string) error {
	labels := componentlabels.GetLabels(serviceName, applicationName, true)
	// save service type as label
	labels[componentlabels.ComponentTypeLabel] = serviceType
	mapOfParameters := util.ConvertKeyValueStringToMap(parameters)
	err := client.CreateServiceInstance(serviceName, serviceType, servicePlan, mapOfParameters, labels)
	if err != nil {
		return errors.Wrap(err, "unable to create service instance")

	}
	return nil
}

// DeleteService will delete the service with the provided `name`
func DeleteService(client *occlient.Client, name string, applicationName string) error {

	labels := componentlabels.GetLabels(name, applicationName, false)
	err := client.DeleteServiceInstance(labels)
	if err != nil {
		return errors.Wrapf(err, "unable to retrieve list of services")
	}
	return nil

}

// List lists all the deployed services
func List(client *occlient.Client, applicationName string, projectName string) ([]ServiceInfo, error) {
	labels := map[string]string{
		applabels.ApplicationLabel: applicationName,
	}

	//since, service is associated with application, it consist of application label as well
	// which we can give as a selector
	applicationSelector := util.ConvertLabelsToSelector(labels)

	// get service instance list based on given selector
	serviceInstanceList, err := client.GetServiceInstanceList(projectName, applicationSelector)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to list services")
	}

	var services []ServiceInfo
	// Iterate through serviceInstanceList and add to service
	for _, elem := range serviceInstanceList {
		services = append(services, ServiceInfo{Name: elem.Labels[componentlabels.ComponentLabel], Type: elem.Labels[componentlabels.ComponentTypeLabel], Status: elem.Status.Conditions[0].Reason})
	}

	return services, nil
}

// GetSvcByType returns the matching (by type) service or nil of there are no matches
func GetSvcByType(client *occlient.Client, serviceType string) (*occlient.Service, error) {
	catalogList, err := ListCatalog(client)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to list catalog")
	}

	for _, supported := range catalogList {
		if serviceType == supported.Name {
			return &supported, nil
		}
	}
	return nil, nil
}

// SvcExists Checks whether a service with the given name exists in the current application or not
// serviceName is the service name to perform check for
// The first returned parameter is a bool indicating if a service with the given name already exists or not
// The second returned parameter is the error that might occurs while execution
func SvcExists(client *occlient.Client, serviceName, applicationName, projectName string) (bool, error) {

	serviceList, err := List(client, applicationName, projectName)
	if err != nil {
		return false, errors.Wrap(err, "unable to get the service list")
	}
	for _, service := range serviceList {
		if service.Name == serviceName {
			return true, nil
		}
	}
	return false, nil
}
