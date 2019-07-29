package catalog

import (
	"fmt"
	"strings"

	"github.com/golang/glog"
	imagev1 "github.com/openshift/api/image/v1"
	"github.com/openshift/odo/pkg/occlient"
	"github.com/pkg/errors"
)

type CatalogImage struct {
	Name          string
	Namespace     string
	AllTags       []string
	NonHiddenTags []string
}

// List lists all the available component types
func List(client *occlient.Client) ([]CatalogImage, error) {

	catalogList, err := getDefaultBuilderImages(client)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get image streams")
	}

	if len(catalogList) == 0 {
		return nil, errors.New("unable to retrieve any catalog images from the OpenShift cluster")
	}

	return catalogList, nil
}

// Search searches for the component
func Search(client *occlient.Client, name string) ([]string, error) {
	var result []string
	componentList, err := List(client)
	if err != nil {
		return nil, errors.Wrap(err, "unable to list components")
	}

	// do a partial search in all the components
	for _, component := range componentList {
		// we only show components that contain the search term and that have at least non-hidden tag
		// since a component with all hidden tags is not shown in the odo catalog list components either
		if strings.Contains(component.Name, name) && len(component.NonHiddenTags) > 0 {
			result = append(result, component.Name)
		}
	}

	return result, nil
}

// Exists returns true if the given component type and the version are valid, false if not
func Exists(client *occlient.Client, componentType string, componentVersion string) (bool, error) {
	imageStream, err := client.GetImageStream("", componentType, componentVersion)
	if err != nil {
		return false, errors.Wrapf(err, "unable to get from catalog")
	}
	if imageStream == nil {
		return false, nil
	}
	return true, nil
}

// getDefaultBuilderImages returns the default builder images available in the
// openshift and the current namespaces
func getDefaultBuilderImages(client *occlient.Client) ([]CatalogImage, error) {

	var imageStreams []imagev1.ImageStream
	currentNamespace := client.GetCurrentProjectName()

	// Fetch imagestreams from default openshift namespace
	openshiftNSImageStreams, openshiftNSISFetchError := client.GetImageStreams(occlient.OpenShiftNameSpace)
	if openshiftNSISFetchError != nil {
		// Tolerate the error as it might only be a partial failure
		// We may get the imagestreams from other Namespaces
		//err = errors.Wrapf(openshiftNSISFetchError, "unable to get Image Streams from namespace %s", occlient.OpenShiftNameSpace)
		// log it for debugging purposes
		glog.V(4).Infof("Unable to get Image Streams from namespace %s. Error %s", occlient.OpenShiftNameSpace, openshiftNSISFetchError.Error())
	}

	// Fetch imagestreams from current namespace
	currentNSImageStreams, currentNSISFetchError := client.GetImageStreams(currentNamespace)
	// If failure to fetch imagestreams from current namespace, log the failure for debugging purposes
	if currentNSISFetchError != nil {
		// Tolerate the error as it is totally a valid scenario to not have any imagestreams in current namespace
		// log it for debugging purposes
		glog.V(4).Infof("Unable to get Image Streams from namespace %s. Error %s", currentNamespace, currentNSISFetchError.Error())
	}

	// If failure fetching imagestreams from both namespaces, error out
	if openshiftNSISFetchError != nil && currentNSISFetchError != nil {
		return nil, errors.Wrapf(
			fmt.Errorf("%s.\n%s", openshiftNSISFetchError, currentNSISFetchError),
			"Failed to fetch imagestreams from both openshift and %s namespaces.\nPlease ensure that a builder imagestream of required version for the component exists in either openshift or %s namespaces",
			currentNamespace,
			currentNamespace,
		)
	}

	// Resultant imagestreams is list of imagestreams from current and openshift namespaces
	imageStreams = append(imageStreams, openshiftNSImageStreams...)
	imageStreams = append(imageStreams, currentNSImageStreams...)

	// create a map from name (builder image name + tag) to the ImageStreamTag
	// we need this in order to filter out hidden tags
	imageStreamTagMap := make(map[string]imagev1.ImageStreamTag)

	currentNSImageStreamTags, currentNSImageStreamTagsErr := client.GetImageStreamTags(currentNamespace)
	openshiftNSImageStreamTags, openshiftNSImageStreamTagsErr := client.GetImageStreamTags(occlient.OpenShiftNameSpace)

	// If failure fetching imagestreamtags from both namespaces, error out
	if currentNSImageStreamTagsErr != nil && openshiftNSImageStreamTagsErr != nil {
		return nil, errors.Wrapf(
			fmt.Errorf("%s.\n%s", currentNSImageStreamTagsErr, openshiftNSImageStreamTagsErr),
			"Failed to fetch imagestreamtags from both openshift and %s namespaces.\nPlease ensure that a builder imagestream of required version for the component exists in either openshift or %s namespaces",
			currentNamespace,
			currentNamespace,
		)
	}

	// create a map from name to ImageStreamTag out of all the ImageStreamTag objects we collect
	var imageStreamTags []imagev1.ImageStreamTag
	imageStreamTags = append(imageStreamTags, currentNSImageStreamTags...)
	imageStreamTags = append(imageStreamTags, openshiftNSImageStreamTags...)
	for _, imageStreamTag := range imageStreamTags {
		imageStreamTagMap[imageStreamTag.Name] = imageStreamTag
	}

	var builderImages []CatalogImage

	// Get builder images from the available imagestreams
	for _, imageStream := range imageStreams {
		var allTags []string
		var hiddenTags []string
		buildImage := false

		for _, tagReference := range imageStream.Spec.Tags {
			allTags = append(allTags, tagReference.Name)

			// Check to see if it is a "builder" image
			if _, ok := tagReference.Annotations["tags"]; ok {
				for _, t := range strings.Split(tagReference.Annotations["tags"], ",") {
					// If the tagReference has "builder" then we will add the image to the list
					if t == "builder" {
						buildImage = true
					}
				}
			}

		}

		// Append to the list of images if a "builder" tag was found
		if buildImage {
			// We need to gauge the ImageStreamTag of each potential builder image, because it might contain
			// the 'hidden' tag. If so, this builder image is deprecated and should not be offered to the user
			// as candidate
			for _, tag := range allTags {
				imageStreamTag := imageStreamTagMap[imageStream.Name+":"+tag]
				if _, ok := imageStreamTag.Annotations["tags"]; ok {
					for _, t := range strings.Split(imageStreamTag.Annotations["tags"], ",") {
						// If the tagReference has "builder" then we will add the image to the list
						if t == "hidden" {
							glog.V(5).Infof("Tag: %v of builder: %v is marked as hidden and therefore will be excluded", tag, imageStream.Name)
							hiddenTags = append(hiddenTags, tag)
						}
					}
				}

			}

			catalogImage := CatalogImage{
				Name:          imageStream.Name,
				Namespace:     imageStream.Namespace,
				AllTags:       allTags,
				NonHiddenTags: getAllNonHiddenTags(allTags, hiddenTags),
			}
			builderImages = append(builderImages, catalogImage)
			glog.V(5).Infof("Found builder image: %#v", catalogImage)
		}

	}

	return builderImages, nil
}

func getAllNonHiddenTags(allTags []string, hiddenTags []string) []string {
	result := make([]string, 0, len(allTags))
	for _, t1 := range allTags {
		found := false
		for _, t2 := range hiddenTags {
			if t1 == t2 {
				found = true
				break
			}
		}

		if !found {
			result = append(result, t1)
		}
	}
	return result
}
