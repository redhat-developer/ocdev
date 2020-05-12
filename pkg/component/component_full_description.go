package component

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openshift/odo/pkg/config"
	"github.com/openshift/odo/pkg/log"
	"github.com/openshift/odo/pkg/occlient"
	"github.com/openshift/odo/pkg/odo/util/experimental"
	"github.com/openshift/odo/pkg/storage"
	urlpkg "github.com/openshift/odo/pkg/url"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ComponentFullDescriptionSpec repersents complete desciption of the component
type ComponentFullDescriptionSpec struct {
	App        string              `json:"app,omitempty"`
	Type       string              `json:"type,omitempty"`
	Source     string              `json:"source,omitempty"`
	SourceType string              `json:"sourceType,omitempty"`
	URL        urlpkg.URLList      `json:"urls,omitempty"`
	Storage    storage.StorageList `json:"storages,omitempty"`
	Env        []corev1.EnvVar     `json:"env,omitempty"`
	Ports      []string            `json:"ports,omitempty"`
}

// ComponentFullDescription describes a component fully
type ComponentFullDescription struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ComponentFullDescriptionSpec `json:"spec,omitempty"`
	Status            ComponentStatus              `json:"status,omitempty"`
}

// copyFromComponentDescription copies over all fields from Component that can be copied
func (cfd *ComponentFullDescription) copyFromComponentDesc(component *Component) error {
	d, err := json.Marshal(component)
	if err != nil {
		return err
	}
	return json.Unmarshal(d, cfd)
}

// loadURLSFromClientAndLocalConfig loads url information from localconfig and cluster
func (cfd *ComponentFullDescription) loadURLSFromClientAndLocalConfig(client *occlient.Client, localConfigInfo *config.LocalConfigInfo, componentName string, applicationName string) error {
	urls, err := urlpkg.List(client, localConfigInfo, componentName, applicationName)
	if err != nil {
		return err
	}

	cfd.Spec.URL = urls
	return nil
}

// loadStoragesFromClientAndLocalConfig collects information about storages in localconfig and cluster.
func (cfd *ComponentFullDescription) loadStoragesFromClientAndLocalConfig(client *occlient.Client, localConfigInfo *config.LocalConfigInfo, componentName string, applicationName string, componentDesc *Component) error {
	var storages storage.StorageList
	var err error
	// if component is pushed call ListWithState which gets storages from localconfig and cluster
	// this result is already in mc readable form
	if componentDesc.Status.State == StateTypePushed {
		storages, err = storage.ListStorageWithState(client, localConfigInfo, componentName, applicationName)
		if err != nil {
			return err
		}
	} else {
		// otherwise simply fetch storagelist from localconfig
		storageLocal, err := localConfigInfo.StorageList()
		if err != nil {
			return err
		}
		// convert to machine readable format
		storages = storage.ConvertListLocalToMachine(storageLocal)
	}
	cfd.Spec.Storage = storages
	return nil
}

// fillEmptyFields fills any fields that are empty in componentfulldescription
func (cfd *ComponentFullDescription) fillEmptyFields(componentDesc Component, componentName string, applicationName string, projectName string) {
	//fix missing names in case it in not in description
	if len(cfd.Name) <= 0 {
		cfd.Name = componentName
	}

	if len(cfd.Namespace) <= 0 {
		cfd.Namespace = projectName
	}

	if len(cfd.Kind) <= 0 {
		cfd.Kind = "Component"
	}

	if len(cfd.APIVersion) <= 0 {
		cfd.APIVersion = "odo.openshift.io/v1alpha1"
	}

	if len(cfd.Spec.App) <= 0 {
		cfd.Spec.App = applicationName
	}
	cfd.Spec.Ports = componentDesc.Spec.Ports
}

// NewComponentFullDescriptionFromClientAndLocalConfig gets the complete description of the component from both localconfig and cluster
func NewComponentFullDescriptionFromClientAndLocalConfig(client *occlient.Client, localConfigInfo *config.LocalConfigInfo, componentName string, applicationName string, projectName string) (*ComponentFullDescription, error) {
	cfd := &ComponentFullDescription{}
	state := GetComponentState(client, componentName, applicationName)
	componentDesc, err := GetComponentFromConfig(localConfigInfo)
	if err != nil {
		return cfd, err
	}
	err = cfd.copyFromComponentDesc(&componentDesc)
	if err != nil {
		return cfd, err
	}
	cfd.Status.State = state
	if state == StateTypePushed {
		componentDescFromCluster, err := GetComponent(client, componentName, applicationName, projectName)
		if err != nil {
			return cfd, err
		}
		cfd.Spec.Env = componentDescFromCluster.Spec.Env
		cfd.Spec.Type = componentDescFromCluster.Spec.Type
		cfd.Spec.SourceType = componentDescFromCluster.Spec.SourceType
	}

	cfd.fillEmptyFields(componentDesc, componentName, applicationName, projectName)

	err = cfd.loadURLSFromClientAndLocalConfig(client, localConfigInfo, componentName, applicationName)
	if err != nil {
		return cfd, err
	}

	err = cfd.loadStoragesFromClientAndLocalConfig(client, localConfigInfo, componentName, applicationName, &componentDesc)
	if err != nil {
		return cfd, err
	}
	return cfd, nil
}

// Print prints the complete information of component onto stdout (Note: longterm this function should not need to access any parameters, but just print the information in struct)
func (cfd *ComponentFullDescription) Print(client *occlient.Client) error {
	// TODO: remove the need to client here print should just deal with printing
	log.Describef("Component Name: ", cfd.GetName())
	log.Describef("Type: ", cfd.Spec.Type)

	// Source
	if cfd.Spec.Source != "" {
		log.Describef("Source: ", cfd.Spec.Source)
	}

	// Env
	if cfd.Spec.Env != nil {

		// Retrieve all the environment variables
		var output string
		for _, env := range cfd.Spec.Env {
			output += fmt.Sprintf(" · %v=%v\n", env.Name, env.Value)
		}

		// Cut off the last newline and output
		if len(output) > 0 {
			output = output[:len(output)-1]
			log.Describef("Environment Variables:\n", output)
		}

	}

	// Storage
	if len(cfd.Spec.Storage.Items) > 0 {

		// Gather the output
		var output string
		for _, store := range cfd.Spec.Storage.Items {
			output += fmt.Sprintf(" · %v of size %v mounted to %v\n", store.Name, store.Spec.Size, store.Spec.Path)
		}

		// Cut off the last newline and output
		if len(output) > 0 {
			output = output[:len(output)-1]
			log.Describef("Storage:\n", output)
		}

	}

	// URL
	if len(cfd.Spec.URL.Items) > 0 {
		var output string

		if !experimental.IsExperimentalModeEnabled() {
			// if the component is not pushed
			for i, componentURL := range cfd.Spec.URL.Items {
				if componentURL.Status.State == urlpkg.StateTypePushed {
					output += fmt.Sprintf(" · %v exposed via %v\n", urlpkg.GetURLString(componentURL.Spec.Protocol, componentURL.Spec.Host, "", experimental.IsExperimentalModeEnabled()), componentURL.Spec.Port)
				} else {
					var p string
					if i >= len(cfd.Spec.Ports) {
						p = cfd.Spec.Ports[len(cfd.Spec.Ports)-1]
					} else {
						p = cfd.Spec.Ports[i]
					}
					output += fmt.Sprintf(" · URL named %s will be exposed via %v\n", componentURL.Name, p)
				}
			}
		}
		// Cut off the last newline and output
		if len(output) > 0 {
			output = output[:len(output)-1]
			log.Describef("URLs:\n", output)
		}

	}

	// Linked components
	if len(cfd.Status.LinkedComponents) > 0 {

		// Gather the output
		var output string
		for name, ports := range cfd.Status.LinkedComponents {
			if len(ports) > 0 {
				output += fmt.Sprintf(" · %v - Port(s): %v\n", name, strings.Join(ports, ","))
			} else {
				output += fmt.Sprintf(" · %v\n", name)
			}
		}

		// Cut off the last newline and output
		if len(output) > 0 {
			output = output[:len(output)-1]
			log.Describef("Linked Components:\n", output)
		}

	}

	// Linked services
	if len(cfd.Status.LinkedServices) > 0 {

		// Gather the output
		var output string
		for _, linkedService := range cfd.Status.LinkedServices {

			// Let's also get the secrets / environment variables that are being passed in.. (if there are any)
			secrets, err := client.GetSecret(linkedService, cfd.GetNamespace())
			if err != nil {
				return err
			}

			if len(secrets.Data) > 0 {
				// Iterate through the secrets to throw in a string
				var secretOutput string
				for i := range secrets.Data {
					secretOutput += fmt.Sprintf("    · %v\n", i)
				}

				if len(secretOutput) > 0 {
					// Cut off the last newline
					secretOutput = secretOutput[:len(secretOutput)-1]
					output += fmt.Sprintf(" · %s\n   Environment Variables:\n%s\n", linkedService, secretOutput)
				}

			} else {
				output += fmt.Sprintf(" · %s\n", linkedService)
			}

		}

		if len(output) > 0 {
			// Cut off the last newline and output
			output = output[:len(output)-1]
			log.Describef("Linked Services:\n", output)
		}

	}
	return nil
}
