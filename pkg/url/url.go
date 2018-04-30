package url

import (
	"fmt"

	routev1 "github.com/openshift/api/route/v1"
	"github.com/pkg/errors"
	applicationlabels "github.com/redhat-developer/odo/pkg/application/labels"
	componentlabels "github.com/redhat-developer/odo/pkg/component/labels"
	"github.com/redhat-developer/odo/pkg/occlient"
	"github.com/redhat-developer/odo/pkg/util"

	log "github.com/sirupsen/logrus"
)

type URL struct {
	Name     string
	URL      string
	Protocol string
}

// Delete deletes a URL
func Delete(client *occlient.Client, name string) error {
	return client.DeleteRoute(name)
}

// Create creates a URL
func Create(client *occlient.Client, componentName string, applicationName string) (*URL, error) {

	labels := componentlabels.GetLabels(componentName, applicationName, false)

	// Namespace the component
	namespacedOCObject, err := util.NamespaceOpenShiftObject(componentName, applicationName)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to create namespaced name")
	}

	route, err := client.CreateRoute(namespacedOCObject, labels)

	if err != nil {
		return nil, errors.Wrap(err, "unable to create route")
	}

	return &URL{
		Name:     route.Name,
		URL:      route.Spec.Host,
		Protocol: getProtocol(*route),
	}, nil
}

// List lists the URLs in an application. The results can further be narrowed
// down if a component name is provided, which will only list URLs for the
// given component
func List(client *occlient.Client, componentName string, applicationName string) ([]URL, error) {

	labelSelector := fmt.Sprintf("%v=%v", applicationlabels.ApplicationLabel, applicationName)

	if componentName != "" {
		labelSelector = labelSelector + fmt.Sprintf(",%v=%v", componentlabels.ComponentLabel, componentName)
	}

	log.Debugf("Listing routes with label selector: %v", labelSelector)
	routes, err := client.ListRoutes(labelSelector)
	if err != nil {
		return nil, errors.Wrap(err, "unable to list route names")
	}

	var urls []URL
	for _, r := range routes {
		urls = append(urls, URL{
			Name:     r.Name,
			URL:      r.Spec.Host,
			Protocol: getProtocol(r),
		})
	}

	return urls, nil
}

func getProtocol(route routev1.Route) string {
	if route.Spec.TLS != nil {
		return "https"
	} else {
		return "http"
	}
}

func GetUrlString(url URL) string {
	return url.Protocol + "://" + url.URL
}
