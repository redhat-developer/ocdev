package helper

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	applabels "github.com/openshift/odo/pkg/application/labels"
	"github.com/openshift/odo/pkg/component/labels"
)

func runningCmd(cmd *exec.Cmd) string {
	prog := filepath.Base(cmd.Path)
	return fmt.Sprintf("Running %s with args %v", prog, cmd.Args)
}

func CmdRunner(program string, args ...string) *gexec.Session {
	//prefix ginkgo verbose output with program name
	prefix := fmt.Sprintf("[%s] ", filepath.Base(program))
	prefixWriter := gexec.NewPrefixedWriter(prefix, GinkgoWriter)
	command := exec.Command(program, args...)
	fmt.Fprintln(GinkgoWriter, runningCmd(command))
	session, err := gexec.Start(command, prefixWriter, prefixWriter)
	Expect(err).NotTo(HaveOccurred())
	return session
}

// WaitForOutputToContain waits for for the session stdout output to contain a particular substring
func WaitForOutputToContain(substring string, timeoutInSeconds int, intervalInSeconds int, session *gexec.Session) {

	Eventually(func() string {
		contents := string(session.Out.Contents())
		return contents
	}, timeoutInSeconds, intervalInSeconds).Should(ContainSubstring(substring))

}

// WaitAndCheckForTerminatingState waits for the given interval
// and checks if the given resource type has been deleted on the cluster or is in the terminating state
// path is the path to the program's binary
func WaitAndCheckForTerminatingState(path, resourceType, namespace string, timeoutMinutes int) bool {
	pingTimeout := time.After(time.Duration(timeoutMinutes) * time.Minute)
	// this is a test package so time.Tick() is acceptable
	// nolint
	tick := time.Tick(time.Second)
	for {
		select {
		case <-pingTimeout:
			Fail(fmt.Sprintf("Timeout after %d minutes", timeoutMinutes))

		case <-tick:
			session := CmdRunner(path, "get", resourceType, "--namespace", namespace)
			Eventually(session).Should(gexec.Exit(0))
			// https://github.com/kubernetes/kubectl/issues/847
			outputStdErr := string(session.Wait().Err.Contents())
			outputStdOut := string(session.Wait().Out.Contents())

			// if the resource gets deleted before the check, we won't get the `terminating` state output
			// thus we also check and exit when the resource has been deleted on the cluster.
			if strings.Contains(strings.ToLower(outputStdErr), "no resources found") || strings.Contains(strings.ToLower(outputStdOut), "terminating") {
				return true
			}
		}
	}
}

// GetAnnotationsDeployment gets the annotations from the deployment
// belonging to the given component, app and project
func GetAnnotationsDeployment(path, componentName, appName, projectName string) map[string]string {
	var mapOutput = make(map[string]string)

	selector := fmt.Sprintf("--selector=%s=%s,%s=%s", labels.ComponentLabel, componentName, applabels.ApplicationLabel, appName)
	output := Cmd(path, "get", "deployment", selector, "--namespace", projectName,
		"-o", "go-template='{{ range $k, $v := (index .items 0).metadata.annotations}}{{$k}}:{{$v}}{{\"\\n\"}}{{end}}'").ShouldPass().Out()

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimPrefix(line, "'")
		splits := strings.Split(line, ":")
		if len(splits) < 2 {
			continue
		}
		name := splits[0]
		value := strings.Join(splits[1:], ":")
		mapOutput[name] = value
	}
	return mapOutput
}

// GetSecrets gets all the secrets belonging to the project
func GetSecrets(path, project string) string {
	session := CmdRunner(path, "get", "secrets", "--namespace", project)
	Eventually(session).Should(gexec.Exit(0))
	output := string(session.Wait().Out.Contents())
	return output
}

// GetEnvRefNames gets the ref values from the envFroms of the deployment belonging to the given data
func GetEnvRefNames(path, componentName, appName, projectName string) []string {
	selector := fmt.Sprintf("--selector=%s=%s,%s=%s", labels.ComponentLabel, componentName, applabels.ApplicationLabel, appName)
	output := Cmd(path, "get", "deployment", selector, "--namespace", projectName,
		"-o", "jsonpath='{range .items[0].spec.template.spec.containers[0].envFrom[*]}{.secretRef.name}{\"\\n\"}{end}'").ShouldPass().Out()

	var result []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimPrefix(line, "'")
		result = append(result, strings.TrimSpace(line))
	}
	return result
}
