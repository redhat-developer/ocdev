package component

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/golang/glog"
	"github.com/openshift/odo/pkg/component"
	"github.com/openshift/odo/pkg/config"
	"github.com/openshift/odo/pkg/log"
	"github.com/openshift/odo/pkg/occlient"
	"github.com/openshift/odo/pkg/odo/genericclioptions"
	odoutil "github.com/openshift/odo/pkg/odo/util"
	"github.com/openshift/odo/pkg/project"
	"github.com/openshift/odo/pkg/util"
	"github.com/pkg/errors"
)

// CommonPushOptions has data needed for all pushes
type CommonPushOptions struct {
	ignores []string
	show    bool

	sourceType       config.SrcType
	sourcePath       string
	componentContext string
	client           *occlient.Client
	localConfigInfo  *config.LocalConfigInfo

	pushConfig bool
	pushSource bool

	*genericclioptions.Context
}

// NewCommonPushOptions instantiates a commonPushOptions object
func NewCommonPushOptions() *CommonPushOptions {
	return &CommonPushOptions{
		show: false,
	}
}

// ResolveSrcAndConfigFlags sets all pushes if none is asked
func (cpo *CommonPushOptions) ResolveSrcAndConfigFlags() {
	// If neither config nor source flag is passed, update both config and source to the component
	if !cpo.pushConfig && !cpo.pushSource {
		cpo.pushConfig = true
		cpo.pushSource = true
	}
}

func (cpo *CommonPushOptions) createCmpIfNotExistsAndApplyCmpConfig(stdout io.Writer) error {
	if !cpo.pushConfig {
		// Not the case of component creation or updation(with new config)
		// So nothing to do here and hence return from here
		return nil
	}

	cmpName := cpo.localConfigInfo.GetName()
	appName := cpo.localConfigInfo.GetApplication()

	// First off, we check to see if the component exists. This is ran each time we do `odo push`
	s := log.Spinner("Checking component")
	defer s.End(false)
	isCmpExists, err := component.Exists(cpo.Context.Client, cmpName, appName)
	if err != nil {
		return errors.Wrapf(err, "failed to check if component %s exists or not", cmpName)
	}
	s.End(true)

	// Output the "new" section (applying changes)
	log.Info("\nConfiguration changes")

	// If the component does not exist, we will create it for the first time.
	if !isCmpExists {

		s = log.Spinner("Creating component")
		defer s.End(false)

		// Classic case of component creation
		if err = component.CreateComponent(cpo.Context.Client, *cpo.localConfigInfo, cpo.componentContext, stdout); err != nil {
			log.Errorf(
				"Failed to create component with name %s. Please use `odo config view` to view settings used to create component. Error: %+v",
				cmpName,
				err,
			)
			os.Exit(1)
		}

		s.End(true)
	}

	// Apply config
	err = component.ApplyConfig(cpo.Context.Client, *cpo.localConfigInfo, stdout, isCmpExists)
	if err != nil {
		odoutil.LogErrorAndExit(err, "Failed to update config to component deployed")
	}

	return nil
}

// ResolveProject completes the push options as needed
func (cpo *CommonPushOptions) ResolveProject(prjName string) (err error) {

	// check if project exist
	isPrjExists, err := project.Exists(cpo.Context.Client, prjName)
	if err != nil {
		return errors.Wrapf(err, "failed to check if project with name %s exists", prjName)
	}
	if !isPrjExists {
		log.Successf("Creating project %s", prjName)
		err = project.Create(cpo.Context.Client, prjName, true)
		if err != nil {
			log.Errorf("Failed creating project %s", prjName)
			return errors.Wrapf(
				err,
				"project %s does not exist. Failed creating it.Please try after creating project using `odo project create <project_name>`",
				prjName,
			)
		}
		log.Successf("Successfully created project %s", prjName)
	}
	cpo.Context.Client.Namespace = prjName
	return
}

// SetSourceInfo sets up source information
func (cpo *CommonPushOptions) SetSourceInfo() (err error) {
	cpo.sourceType = cpo.localConfigInfo.GetSourceType()

	glog.V(4).Infof("SourceLocation: %s", cpo.localConfigInfo.GetSourceLocation())

	// Get SourceLocation here...
	cpo.sourcePath, err = cpo.localConfigInfo.GetOSSourcePath()
	if err != nil {
		return errors.Wrap(err, "unable to retrieve absolute path to source location")
	}

	glog.V(4).Infof("Source Path: %s", cpo.sourcePath)
	return
}

// Push pushes changes as per set options
func (cpo *CommonPushOptions) Push() (err error) {
	stdout := color.Output

	cmpName := cpo.localConfigInfo.GetName()
	appName := cpo.localConfigInfo.GetApplication()

	err = cpo.createCmpIfNotExistsAndApplyCmpConfig(stdout)
	if err != nil {
		return
	}

	if !cpo.pushSource {
		// If source is not requested for update, return
		return nil
	}

	log.Infof("\nPushing to component %s of type %s", cmpName, cpo.sourceType)

	// Get SourceLocation here...
	cpo.sourcePath, err = cpo.localConfigInfo.GetOSSourcePath()
	if err != nil {
		return errors.Wrap(err, "unable to retrieve OS source path to source location")
	}

	switch cpo.sourceType {
	case config.LOCAL:
		glog.V(4).Infof("Copying directory %s to pod", cpo.sourcePath)
		err = component.PushLocal(
			cpo.Context.Client,
			cmpName,
			appName,
			cpo.sourcePath,
			os.Stdout,
			[]string{},
			[]string{},
			true,
			util.GetAbsGlobExps(cpo.sourcePath, cpo.ignores),
			cpo.show,
		)

		if err != nil {
			return errors.Wrapf(err, fmt.Sprintf("Failed to push component: %v", cmpName))
		}

	case config.BINARY:

		// We will pass in the directory, NOT filepath since this is a binary..
		binaryDirectory := filepath.Dir(cpo.sourcePath)

		glog.V(4).Infof("Copying binary file %s to pod", cpo.sourcePath)
		err = component.PushLocal(
			cpo.Context.Client,
			cmpName,
			appName,
			binaryDirectory,
			os.Stdout,
			[]string{cpo.sourcePath},
			[]string{},
			true,
			util.GetAbsGlobExps(cpo.sourcePath, cpo.ignores),
			cpo.show,
		)

		if err != nil {
			return errors.Wrapf(err, fmt.Sprintf("Failed to push component: %v", cmpName))
		}

		// we don't need a case for building git components
		// the build happens before deployment

		return errors.Wrapf(err, fmt.Sprintf("failed to push component: %v", cmpName))
	}

	log.Success("Changes successfully pushed to component")
	return
}
