package component

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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

	pushConfig  bool
	pushSource  bool
	forceBuild  bool
	isCmpExists bool

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

	// Output the "new" section (applying changes)
	log.Info("\nConfiguration changes")

	// If the component does not exist, we will create it for the first time.
	if !cpo.isCmpExists {

		// Classic case of component creation
		if err := component.CreateComponent(cpo.Context.Client, *cpo.localConfigInfo, cpo.componentContext, stdout); err != nil {
			log.Errorf(
				"Failed to create component with name %s. Please use `odo config view` to view settings used to create component. Error: %+v",
				cmpName,
				err,
			)
			os.Exit(1)
		}
	}

	// Apply config
	err := component.ApplyConfig(cpo.Context.Client, *cpo.localConfigInfo, stdout, cpo.isCmpExists)
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

	deletedFiles := []string{}
	changedFiles := []string{}
	isForcePush := false

	stdout := color.Output

	cmpName := cpo.localConfigInfo.GetName()
	appName := cpo.localConfigInfo.GetApplication()

	if cpo.componentContext == "" {
		cpo.componentContext = strings.Trim(filepath.Dir(cpo.localConfigInfo.Filename), ".odo")
	}

	err = cpo.createCmpIfNotExistsAndApplyCmpConfig(stdout)
	if err != nil {
		return
	}

	if !cpo.pushSource {
		// If source is not requested for update, return
		return nil
	}

	log.Infof("\nPushing to component %s of type %s", cmpName, cpo.sourceType)

	if !cpo.forceBuild && cpo.sourceType != config.GIT {
		absIgnoreRules := util.GetAbsGlobExps(cpo.sourcePath, cpo.ignores)

		spinner := log.NewStatus(log.GetStdout())
		defer spinner.End(true)
		if cpo.isCmpExists {
			spinner.Start("Checking file changes for pushing", false)
		} else {
			// if the component doesn't exist, we don't check for changes in the files
			// thus we show a different message
			spinner.Start("Checking files for pushing", false)
		}

		// run the indexer and find the modified/added/deleted/renamed files
		filesChanged, filesDeleted, err := util.RunIndexer(cpo.componentContext, absIgnoreRules)
		spinner.End(true)

		if err != nil {
			return err
		}

		if cpo.isCmpExists {
			// apply the glob rules from the .gitignore/.odo file
			// and ignore the files on which the rules apply and filter them out
			filesChangedFiltered, filesDeletedFiltered := filterIgnores(filesChanged, filesDeleted, absIgnoreRules)

			// Remove the relative file directory from the list of deleted files
			// in order to make the changes correctly within the OpenShift pod
			deletedFiles, err = util.RemoveRelativePathFromFiles(filesDeletedFiltered, cpo.sourcePath)
			if err != nil {
				return err
			}
			glog.V(4).Infof("List of files to be deleted: +%v", deletedFiles)
			changedFiles = filesChangedFiltered

			if len(filesChangedFiltered) == 0 && len(filesDeletedFiltered) == 0 {
				// no file was modified/added/deleted/renamed, thus return without building
				log.Success("No file changes detected, skipping build. Use the '-f' flag to force the build.")
				return nil
			}
		}
	}

	if cpo.forceBuild || !cpo.isCmpExists {
		isForcePush = true
	}

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
			changedFiles,
			deletedFiles,
			isForcePush,
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
			deletedFiles,
			isForcePush,
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

// filterIgnores applies the glob rules on the filesChanged and filesDeleted and filters them
// returns the filtered results which match any of the glob rules
func filterIgnores(filesChanged, filesDeleted, absIgnoreRules []string) (filesChangedFiltered, filesDeletedFiltered []string) {
	for _, file := range filesChanged {
		match, err := util.IsGlobExpMatch(file, absIgnoreRules)
		if err != nil {
			continue
		}
		if !match {
			filesChangedFiltered = append(filesChangedFiltered, file)
		}
	}

	for _, file := range filesDeleted {
		match, err := util.IsGlobExpMatch(file, absIgnoreRules)
		if err != nil {
			continue
		}
		if !match {
			filesDeletedFiltered = append(filesDeletedFiltered, file)
		}
	}
	return filesChangedFiltered, filesDeletedFiltered
}
