package component

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/redhat-developer/ocdev/pkg/application"
	"github.com/redhat-developer/ocdev/pkg/config"
	"github.com/redhat-developer/ocdev/pkg/occlient"
)

// componentLabel is a label key used to identify component
const componentLabel = "app.kubernetes.io/component-name"

// getComponentLabels return labels that should be applied to every object
// additional labels are used only for creating object
// if you are creating something use additional=true
// if you need labels to filter component that use additional=false
func getComponentLabels(componentName string, additional bool) (map[string]string, error) {
	currentApplication, err := application.GetCurrent()
	if err != nil {
		return nil, errors.Wrap(err, "unable to get current application")
	}
	labels := map[string]string{
		application.ApplicationLabel: currentApplication,
		componentLabel:               componentName,
	}

	if additional {
		for _, additionalLabel := range application.AdditionalApplicationLabels {
			labels[additionalLabel] = currentApplication
		}
	}

	return labels, nil
}

func CreateFromGit(name string, ctype string, url string) (string, error) {
	labels, err := getComponentLabels(name, true)
	if err != nil {
		return "", errors.Wrapf(err, "unable to activate component %s created from git", name)
	}

	output, err := occlient.NewAppS2I(name, ctype, url, labels)
	if err != nil {
		return "", err
	}

	if err = SetCurrent(name); err != nil {
		return "", errors.Wrapf(err, "unable to activate component %s created from git", name)
	}
	return output, nil
}

func CreateEmpty(name string, ctype string) (string, error) {
	labels, err := getComponentLabels(name, true)
	if err != nil {
		return "", errors.Wrapf(err, "unable to activate component %s created from git", name)
	}

	output, err := occlient.NewAppS2IEmpty(name, ctype, labels)
	if err != nil {
		return "", err
	}
	if err = SetCurrent(name); err != nil {
		return "", errors.Wrapf(err, "unable to activate empty component %s", name)
	}

	return output, nil
}

func CreateFromDir(name string, ctype, dir string) (string, error) {
	output, err := CreateEmpty(name, ctype)
	if err != nil {
		return "", errors.Wrap(err, "unable to get create empty component")
	}

	// TODO: it might not be ideal to print to stdout here
	fmt.Println(output)
	fmt.Println("please wait, building application...")

	output, err = occlient.StartBuild(name, dir)
	if err != nil {
		return "", err
	}
	fmt.Println(output)

	return "", nil

}

// Delete whole component
func Delete(name string) (string, error) {
	labels, err := getComponentLabels(name, false)
	if err != nil {
		return "", errors.Wrapf(err, "unable to activate component %s created from git", name)
	}

	output, err := occlient.Delete("all", "", labels)
	if err != nil {
		return "", errors.Wrap(err, "unable to delete component")
	}

	return output, nil
}

func SetCurrent(name string) error {
	cfg, err := config.New()
	if err != nil {
		return errors.Wrap(err, "unable to get config")
	}

	currentAppliction, err := application.GetCurrent()
	if err != nil {
		return errors.Wrap(err, "unable to get current application")
	}

	cfg.SetActiveComponent(name, currentAppliction)
	if err != nil {
		return errors.Wrapf(err, "unable to set current component %s", name)
	}

	return nil
}

func GetCurrent() (string, error) {
	cfg, err := config.New()
	if err != nil {
		return "", errors.Wrap(err, "unable to get config")
	}
	currentApplication, err := application.GetCurrent()
	if err != nil {
		return "", errors.Wrap(err, "unable to get active application")
	}

	currentProject, err := occlient.GetCurrentProjectName()
	if err != nil {
		return "", errors.Wrap(err, "unable to get current  component")
	}

	currentComponent := cfg.GetActiveComponent(currentApplication, currentProject)
	if currentComponent == "" {
		return "", errors.New("no component is set as active")
	}
	return currentComponent, nil

}

func Push(name string, dir string) (string, error) {
	output, err := occlient.StartBuild(name, dir)
	if err != nil {
		return "", errors.Wrap(err, "unable to start build")
	}
	return output, nil
}
