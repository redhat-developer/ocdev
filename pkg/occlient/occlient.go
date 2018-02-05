package occlient

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// getOcBinary returns full path to oc binary
// first it looks for env variable KUBECTL_PLUGINS_CALLER (run as oc plugin)
// than looks for env variable OC_BIN (set manualy by user)
// at last it tries to find oc in default PATH
func getOcBinary() (string, error) {
	log.Debug("getOcBinary - searching for oc binary")

	var ocPath string

	envKubectlPluginCaller := os.Getenv("KUBECTL_PLUGINS_CALLER")
	envOcBin := os.Getenv("OC_BIN")

	log.Debugf("envKubectlPluginCaller = %s\n", envKubectlPluginCaller)
	log.Debugf("envOcBin = %s\n", envOcBin)

	if len(envKubectlPluginCaller) > 0 {
		log.Debug("using path from KUBECTL_PLUGINS_CALLER")
		ocPath = envKubectlPluginCaller
	} else if len(envOcBin) > 0 {
		log.Debug("using path from OC_BIN")
		ocPath = envOcBin
	} else {
		path, err := exec.LookPath("oc")
		if err != nil {
			log.Debug("oc binary not found in PATH")
			return "", err
		}
		log.Debug("using oc from PATH")
		ocPath = path
	}
	log.Debug("using oc from %s", ocPath)

	if _, err := os.Stat(ocPath); err != nil {
		return "", err
	}

	return ocPath, nil
}

type OcCommand struct {
	args   []string
	data   *string
	format string
}

// runOcCommands executes oc
// args - command line arguments to be passed to oc ('-o json' is added by default if data is not nil)
// data - is a pointer to a string, if set than data is given to command to stdin ('-f -' is added to args as default)
func runOcComamnd(command *OcCommand) ([]byte, error) {

	ocpath, err := getOcBinary()
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(ocpath, command.args...)

	// if data is not set assume that it is get command
	if len(command.format) > 0 {
		cmd.Args = append(cmd.Args, "-o", command.format)
	}
	if command.data != nil {
		// data is given, assume this is crate or apply command
		// that takes data from stdin
		cmd.Args = append(cmd.Args, "-f", "-")

		// Read from stdin
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return nil, err
		}

		// Write to stdin
		go func() {
			defer stdin.Close()
			_, err := io.WriteString(stdin, *command.data)
			if err != nil {
				fmt.Printf("can't write to stdin %v\n", err)
			}
		}()
	}

	// Execute the actual command
	var stdOut, stdErr bytes.Buffer
	cmd.Stdout = &stdOut
	cmd.Stderr = &stdErr

	log.Debugf("running oc command with arguments: %s\n", strings.Join(cmd.Args, " "))

	err = cmd.Run()
	if err != nil {
		outputMessage := ""
		if stdErr.Len() != 0 {
			outputMessage = stdErr.String()
		}
		if stdOut.Len() != 0 {
			outputMessage = fmt.Sprintf("\n%s", stdErr.String())
		}

		if outputMessage != "" {
			return nil, fmt.Errorf("failed to execute oc command\n %s", outputMessage)
		}
		return nil, err
	}

	if stdErr.Len() != 0 {
		return nil, fmt.Errorf("Error output:\n%s", stdErr.String())
	}

	return stdOut.Bytes(), nil

}

func GetCurrentProjectName() (string, error) {
	// We need to run `oc project` because it returns an error when project does
	// not exist, while `oc project -q` does not return an error, it simply
	// returns the project name
	_, err := runOcComamnd(&OcCommand{
		args: []string{"project"},
	})
	if err != nil {
		return "", errors.Wrap(err, "unable to get current project info")
	}

	output, err := runOcComamnd(&OcCommand{
		args: []string{"project", "-q"},
	})
	if err != nil {
		return "", errors.Wrap(err, "unable to get current project name")
	}

	return strings.TrimSpace(string(output)), nil
}

func CreateNewProject(name string) error {
	_, err := runOcComamnd(&OcCommand{
		args: []string{"new-project", name},
	})
	if err != nil {
		return errors.Wrap(err, "unable to create new project")
	}
	return nil
}

// addLabelsToArgs adds labels from map to args as a new argument in format that oc requires
// --labels label1=value1,label2=value2
func addLabelsToArgs(labels map[string]string, args []string) []string {
	if labels != nil {
		var labelsString []string
		for key, value := range labels {
			labelsString = append(labelsString, fmt.Sprintf("%s=%s", key, value))
		}
		args = append(args, "--labels")
		args = append(args, strings.Join(labelsString, ","))
	}

	return args
}

// NewAppS2I create new application  using S2I with source in git repository
func NewAppS2I(name string, builderImage string, gitUrl string, labels map[string]string) (string, error) {
	args := []string{
		"new-app",
		fmt.Sprintf("%s~%s", builderImage, gitUrl),
		"--name", name,
	}

	args = addLabelsToArgs(labels, args)

	output, err := runOcComamnd(&OcCommand{args: args})
	if err != nil {
		return "", err
	}
	return string(output[:]), nil

}

// NewAppS2I create new application  using S2I from local directory
func NewAppS2IEmpty(name string, builderImage string, labels map[string]string) (string, error) {

	// there is no way to create binary builds using 'oc new-app' other than passing it directory that is not a git repository
	// this is why we are creating empty directory and using is a a source

	tmpDir, err := ioutil.TempDir("", "fakeSource")
	if err != nil {
		return "", errors.Wrap(err, "unable to create tmp directory to use it as a source for build")
	}
	defer os.Remove(tmpDir)

	args := []string{
		"new-app",
		fmt.Sprintf("%s~%s", builderImage, tmpDir),
		"--name", name,
	}

	args = addLabelsToArgs(labels, args)

	output, err := runOcComamnd(&OcCommand{args: args})
	if err != nil {
		return "", err
	}

	return string(output[:]), nil
}

func StartBuildFromDir(name string, dir string) (string, error) {
	args := []string{
		"start-build",
		name,
		"--from-dir", dir,
		"--follow",
	}

	// TODO: build progress is not shown
	output, err := runOcComamnd(&OcCommand{args: args})
	if err != nil {
		return "", err
	}

	return string(output[:]), nil

}

// Delete calls oc delete
// kind is always required (can be set to 'all')
// name can be omitted if labels are set, in that case set name to ''
// if you want to delete object just by its name set labels to nil
func Delete(kind string, name string, labels map[string]string) (string, error) {

	args := []string{
		"delete",
		kind,
	}

	if len(name) > 0 {
		args = append(args, name)
	}

	if labels != nil {
		var labelsString []string
		for key, value := range labels {
			labelsString = append(labelsString, fmt.Sprintf("%s=%s", key, value))
		}
		args = append(args, "--selector")
		args = append(args, strings.Join(labelsString, ","))
	}

	output, err := runOcComamnd(&OcCommand{args: args})
	if err != nil {
		return "", err
	}

	return string(output[:]), nil

}
