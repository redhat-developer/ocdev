package occlient

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const ocRequestTimeout = 1 * time.Second

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
	if !isServerUp() {
		return nil, errors.New("server is down")
	}
	if !isLoggedIn() {
		return nil, errors.New("please log in to the cluster")
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

	log.Debugf("running oc command with arguments: %s\n", strings.Join(cmd.Args, " "))

	output, err := cmd.CombinedOutput()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return nil, errors.Wrapf(err, "command: %v failed to run:\n%v", cmd.Args, string(output))
		}
		return nil, errors.Wrap(err, "unable to get combined output")
	}

	return output, nil
}

func isLoggedIn() bool {
	ocpath, err := getOcBinary()
	if err != nil {
		log.Debug(errors.Wrap(err, "unable to find oc binary"))
		return false
	}

	cmd := exec.Command(ocpath, "whoami")
	output, err := cmd.CombinedOutput()
	if err != nil && strings.Contains(string(output), "system:anonymous") {
		log.Debug(errors.Wrap(err, "error running command"))
		log.Debugf("Output is: %v", output)
		return false
	}
	return true
}

func isServerUp() bool {

	ocpath, err := getOcBinary()
	if err != nil {
		log.Debug(errors.Wrap(err, "unable to find oc binary"))
		return false
	}

	cmd := exec.Command(ocpath, "whoami", "--show-server")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Debug(errors.Wrap(err, "error running command"))
		return false
	}

	server := strings.TrimSpace(string(output))
	u, err := url.Parse(server)
	if err != nil {
		log.Debug(errors.Wrap(err, "unable to parse url"))
		return false
	}

	log.Debugf("Trying to connect to server %v - %v", u.Host)
	_, connectionError := net.DialTimeout("tcp", u.Host, time.Duration(ocRequestTimeout))
	if connectionError != nil {
		log.Debug(errors.Wrap(connectionError, "unable to connect to server"))
		return false
	}
	log.Debugf("Server %v is up", server)
	return true
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

func GetProjects() (string, error) {
	output, err := runOcComamnd(&OcCommand{
		args:   []string{"get", "project"},
		format: "custom-columns=NAME:.metadata.name",
	})
	if err != nil {
		return "", errors.Wrap(err, "unable to get projects")
	}
	return strings.Join(strings.Split(string(output), "\n")[1:], "\n"), nil
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

func NewAppSearch(query string) (string, error) {
	args := []string{
		"new-app",
		"--search",
		"--image-stream",
		query,
	}
	// TODO: Since we only support ImageStreams right now, it's okay to use
	// TODO: --image-stream. Later on we need to parse this better.

	output, err := runOcComamnd(&OcCommand{args: args})
	if err != nil {
		return "", errors.Wrap(err, "unable to run command")
	}
	return string(output[:]), nil
}

func StartBuild(name string, dir string) (string, error) {
	args := []string{
		"start-build",
		name,
		"--follow",
	}
	if len(dir) > 0 {
		args = append(args, "--from-dir", dir)
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

func DeleteProject(name string) error {
	_, err := runOcComamnd(&OcCommand{
		args: []string{"delete", "project", name},
	})
	if err != nil {
		return errors.Wrap(err, "unable to delete project")
	}
	return nil
}

type VolumeConfig struct {
	Name             *string
	Size             *string
	DeploymentConfig *string
	Path             *string
}

type VolumeOpertaions struct {
	Add    bool
	Remove bool
	List   bool
}

func SetVolumes(config *VolumeConfig, operations *VolumeOpertaions) (string, error) {
	args := []string{
		"set",
		"volumes",
		"dc", *config.DeploymentConfig,
		"--type", "pvc",
	}
	if config.Name != nil {
		args = append(args, "--name", *config.Name)
	}
	if config.Size != nil {
		args = append(args, "--claim-size", *config.Size)
	}
	if config.Path != nil {
		args = append(args, "--mount-path", *config.Path)
	}
	if operations.Add {
		args = append(args, "--add")
	}
	if operations.Remove {
		args = append(args, "--remove", "--confirm")
	}
	if operations.List {
		args = append(args, "--all")
	}
	output, err := runOcComamnd(&OcCommand{
		args: args,
	})
	if err != nil {
		return "", errors.Wrap(err, "unable to perform volume operations")
	}
	return string(output), nil
}

// GetLabelValues get label values from all object that are labeled with given label
// returns slice of uniq label values
func GetLabelValues(project string, label string) ([]string, error) {
	// get all object that have given label
	// and show just label values separated by ,
	args := []string{
		"get", "all",
		"--selector", label,
		"--namespace", project,
		"-o", "go-template={{range .items}}{{range $key, $value := .metadata.labels}}{{if eq $key \"" + label + "\"}}{{$value}},{{end}}{{end}}{{end}}",
	}

	output, err := runOcComamnd(&OcCommand{args: args})
	if err != nil {
		return nil, err
	}

	values := []string{}
	// deduplicate label values
	for _, val := range strings.Split(string(output), ",") {
		val = strings.TrimSpace(val)
		if val != "" {
			// check if this is the first time we see this value
			found := false
			for _, existing := range values {
				if val == existing {
					found = true
				}
			}
			if !found {
				values = append(values, val)
			}
		}
	}

	return values, nil
}
