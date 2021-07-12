package helper

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	applabels "github.com/openshift/odo/pkg/application/labels"
	"github.com/openshift/odo/pkg/component/labels"
)

const (
	ResourceTypeDeploymentConfig = "dc"
	ResourceTypeRoute            = "route"
)

type OcRunner struct {
	// path to oc binary
	path string
}

// NewOcRunner initializes new OcRunner
func NewOcRunner(ocPath string) OcRunner {
	return OcRunner{
		path: ocPath,
	}
}

// Run oc with given arguments
func (oc OcRunner) Run(args ...string) *gexec.Session {
	session := CmdRunner(oc.path, args...)
	Eventually(session).Should(gexec.Exit(0))
	return session
}

// GetCurrentProject get currently active project in oc
// returns empty string if there no active project, or no access to the project
func (oc OcRunner) GetCurrentProject() string {
	session := CmdRunner(oc.path, "project", "-q")
	session.Wait()
	if session.ExitCode() == 0 {
		return strings.TrimSpace(string(session.Out.Contents()))
	}
	return ""
}

// GetCurrentServerURL retrieves the URL of the server we're currently connected to
// returns empty if not connected or an error occurred
func (oc OcRunner) GetCurrentServerURL() string {
	session := CmdRunner(oc.path, "project")
	session.Wait()
	if session.ExitCode() == 0 {
		output := strings.TrimSpace(string(session.Out.Contents()))
		// format is: Using project "<namespace>" on server "<url>".
		a := strings.Split(output, "\"")
		return a[len(a)-2] // last entry is ".", we need the one before that
	}
	return ""
}

// GetFirstURL returns the url of the first Route that it can find for given component
func (oc OcRunner) GetFirstURL(component string, app string, project string) string {
	session := CmdRunner(oc.path, "get", "route",
		"-n", project,
		"-l", "app.kubernetes.io/instance="+component,
		"-l", "app.kubernetes.io/part-of="+app,
		"-o", "jsonpath={.items[0].spec.host}")

	session.Wait()
	if session.ExitCode() == 0 {
		return string(session.Out.Contents())
	}
	return ""
}

//StatFileInPodContainer returns stat result of filepath in a container of a pod of given component, in a given app, in a given project.
//It also strips access time information as it vaires accross file systems/kernel configs, and we are not interested
//in it anyway
func StatFileInPodContainer(runner CliRunner, cmpName, containerName, appName, project, filepath string) string {
	podName := runner.GetRunningPodNameByComponent(cmpName, project)
	var result string
	runner.CheckCmdOpInRemoteDevfilePod(
		podName,
		containerName,
		project,
		[]string{"stat", filepath},
		func(cmdOp string, err error) bool {
			//strip out access info as
			// 1. Touching a file (such as running it in a script) modifies access times. This gives wrong value on mounts without noatime
			// 2. We are not interested in Access info anyway.
			re := regexp.MustCompile("(?m)[\r\n]+^.*Access.*$")
			result = re.ReplaceAllString(cmdOp, "")
			return true
		},
	)
	return result
}

// GetComponentRoutes run command to get the Routes in yaml format for given component
func (oc OcRunner) GetComponentRoutes(component string, app string, project string) string {
	session := CmdRunner(oc.path, "get", "route",
		"-n", project,
		"-l", "app.kubernetes.io/instance="+component,
		"-l", "app.kubernetes.io/part-of="+app,
		"-o", "yaml")

	session.Wait()
	if session.ExitCode() == 0 {
		return string(session.Out.Contents())
	}
	return ""
}

// GetComponentDC run command to get the DeploymentConfig in yaml format for given component
func (oc OcRunner) GetComponentDC(component string, app string, project string) string {
	session := CmdRunner(oc.path, "get", "dc",
		"-n", project,
		"-l", "app.kubernetes.io/instance="+component,
		"-l", "app.kubernetes.io/part-of="+app,
		"-o", "yaml")

	session.Wait()
	if session.ExitCode() == 0 {
		return string(session.Out.Contents())
	}
	return ""
}

// SourceTest checks the component-source-type and the source url in the annotation of the bc and dc
// appTestName is the name of the app
// sourceType is the type of the source of the component i.e git/binary/local
// source is the source of the component i.e gitURL or path to the directory or binary file
func (oc OcRunner) SourceTest(appTestName string, sourceType string, source string) {
	// checking for source-type in dc
	sourceTypeInDc := Cmd(oc.path, "get", "dc", "wildfly-"+appTestName,
		"-o", "go-template='{{index .metadata.annotations \"app.kubernetes.io/component-source-type\"}}'").ShouldPass().Out()
	Expect(sourceTypeInDc).To(ContainSubstring(sourceType))

	// checking for source in dc
	sourceInDc := Cmd(oc.path, "get", "dc", "wildfly-"+appTestName,
		"-o", "go-template='{{index .metadata.annotations \"app.openshift.io/vcs-uri\"}}'").ShouldPass().Out()
	Expect(sourceInDc).To(ContainSubstring(source))
}

// ExecListDir returns dir list in specified location of pod
func (oc OcRunner) ExecListDir(podName string, projectName string, dir string) string {
	stdOut := Cmd(oc.path, "exec", podName, "--namespace", projectName,
		"--", "ls", "-lai", dir).ShouldPass().Out()
	return stdOut
}

// Exec allows generic execution of commands, returning the contents of stdout
func (oc OcRunner) Exec(podName string, projectName string, args ...string) string {

	cmd := []string{"exec", podName, "--namespace", projectName}

	cmd = append(cmd, args...)

	stdOut := Cmd(oc.path, cmd...).ShouldPass().Out()
	return stdOut
}

// CheckCmdOpInRemoteCmpPod runs the provided command on remote component pod and returns the return value of command output handler function passed to it
func (oc OcRunner) CheckCmdOpInRemoteCmpPod(cmpName string, appName string, prjName string, cmd []string, checkOp func(cmdOp string, err error) bool) bool {
	cmpDCName := fmt.Sprintf("%s-%s", cmpName, appName)
	outPodName := Cmd(oc.path, "get", "pods", "--namespace", prjName,
		"--selector=deploymentconfig="+cmpDCName,
		"-o", "jsonpath='{.items[0].metadata.name}'").ShouldPass().Out()
	podName := strings.Replace(outPodName, "'", "", -1)
	session := CmdRunner(oc.path, append([]string{"exec", podName, "--namespace", prjName,
		"-c", cmpDCName, "--"}, cmd...)...)
	stdOut := string(session.Wait().Out.Contents())
	stdErr := string(session.Wait().Err.Contents())
	if stdErr != "" && session.ExitCode() != 0 {
		return checkOp(stdOut, fmt.Errorf("cmd %s failed with error %s on pod %s", cmd, stdErr, podName))
	}
	return checkOp(stdOut, nil)
}

// CheckCmdOpInRemoteDevfilePod runs the provided command on remote component pod and returns the return value of command output handler function passed to it
func (oc OcRunner) CheckCmdOpInRemoteDevfilePod(podName string, containerName string, prjName string, cmd []string, checkOp func(cmdOp string, err error) bool) bool {
	var execOptions []string
	execOptions = []string{"exec", podName, "--namespace", prjName, "--"}
	if containerName != "" {
		execOptions = []string{"exec", podName, "-c", containerName, "--namespace", prjName, "--"}
	}
	args := append(execOptions, cmd...)
	session := CmdRunner(oc.path, args...)
	stdOut := string(session.Wait().Out.Contents())
	stdErr := string(session.Wait().Err.Contents())
	if stdErr != "" && session.ExitCode() != 0 {
		return checkOp(stdOut, fmt.Errorf("cmd %s failed with error %s on pod %s", cmd, stdErr, podName))
	}
	return checkOp(stdOut, nil)
}

// VerifyCmpExists verifies if component was created successfully
func (oc OcRunner) VerifyCmpExists(cmpName string, appName string, prjName string) {
	cmpDCName := fmt.Sprintf("%s-%s", cmpName, appName)
	Cmd(oc.path, "get", "dc", cmpDCName, "--namespace", prjName).ShouldPass()
}

// VerifyLabelExistsOfComponent verifies app name of component
func (oc OcRunner) VerifyLabelExistsOfComponent(cmpName string, namespace string, labelName string) {
	dcName := oc.GetDcName(cmpName, namespace)
	session := Cmd(oc.path, "get", "dc", dcName, "--namespace", namespace,
		"--template={{.metadata.labels}}").ShouldPass().Out()
	Expect(session).To(ContainSubstring(labelName))
}

// VerifyAppNameOfComponent verifies app name of component
func (oc OcRunner) VerifyAppNameOfComponent(cmpName string, appName string, namespace string) {
	session := Cmd(oc.path, "get", "dc", cmpName+"-"+appName, "--namespace", namespace,
		"--template={{.metadata.labels.app}}").ShouldPass().Out()
	Expect(session).To(ContainSubstring(appName))
}

// VerifyCmpName verifies the component name
func (oc OcRunner) VerifyCmpName(cmpName string, namespace string) {
	dcName := oc.GetDcName(cmpName, namespace)
	session := Cmd(oc.path, "get", "dc", dcName,
		"--namespace", namespace,
		"-L", "app.kubernetes.io/instance").ShouldPass().Out()
	Expect(session).To(ContainSubstring(cmpName))
}

// GetDcName execute oc command and returns dc name of a deployed
// component by passing component name as a argument
func (oc OcRunner) GetDcName(compName string, namespace string) string {
	session := Cmd(oc.path, "get", "dc", "--namespace", namespace).ShouldPass().Out()
	re := regexp.MustCompile(compName + `-\S+ `)
	dcName := re.FindString(session)
	return strings.TrimSpace(dcName)
}

// DescribeDc execute oc command and returns dc describe as a string
// by passing dcname and namespace as arguments
func (oc OcRunner) DescribeDc(dcName string, namespace string) string {
	describeInfo := Cmd(oc.path, "describe", "dc/"+dcName, "-n", namespace).ShouldPass().Out()
	return strings.TrimSpace(describeInfo)
}

// GetDcPorts returns the ports of the component
func (oc OcRunner) GetDcPorts(componentName string, appName string, project string) string {
	ports := Cmd(oc.path, "get", "dc", componentName+"-"+appName, "--namespace", project,
		"-o", "go-template='{{range.spec.template.spec.containers}}{{.ports}}{{end}}'").ShouldPass().Out()
	return ports
}

// MaxMemory returns maximum memory
func (oc OcRunner) MaxMemory(componentName string, appName string, project string) string {
	maxMemory := Cmd(oc.path, "get", "dc", componentName+"-"+appName, "--namespace", project,
		"-o", "go-template='{{range.spec.template.spec.containers}}{{.resources.limits.memory}}{{end}}'").ShouldPass().Out()
	return maxMemory
}

// MinMemory returns minimum memory
func (oc OcRunner) MinMemory(componentName string, appName string, project string) string {
	minMemory := Cmd(oc.path, "get", "dc", componentName+"-"+appName, "--namespace", project,
		"-o", "go-template='{{range.spec.template.spec.containers}}{{.resources.requests.memory}}{{end}}'").ShouldPass().Out()
	return minMemory
}

// MaxCPU returns maximum cpu
func (oc OcRunner) MaxCPU(componentName string, appName string, project string) string {
	maxCPU := Cmd(oc.path, "get", "dc", componentName+"-"+appName, "--namespace", project,
		"-o", "go-template='{{range.spec.template.spec.containers}}{{.resources.limits.cpu}}{{end}}'").ShouldPass().Out()
	return maxCPU
}

// MinCPU returns minimum cpu
func (oc OcRunner) MinCPU(componentName string, appName string, project string) string {
	minCPU := Cmd(oc.path, "get", "dc", componentName+"-"+appName, "--namespace", project,
		"-o", "go-template='{{range.spec.template.spec.containers}}{{.resources.requests.cpu}}{{end}}'").ShouldPass().Out()
	return minCPU
}

// SourceTypeDC returns the source type from the deployment config
func (oc OcRunner) SourceTypeDC(componentName string, appName string, project string) string {
	sourceType := Cmd(oc.path, "get", "dc", componentName+"-"+appName, "--namespace", project,
		"-o", "go-template='{{index .metadata.annotations \"app.kubernetes.io/component-source-type\"}}'").ShouldPass().Out()
	return sourceType
}

// SourceTypeBC returns the source type from the build config
func (oc OcRunner) SourceTypeBC(componentName string, appName string, project string) string {
	sourceType := Cmd(oc.path, "get", "bc", componentName+"-"+appName, "--namespace", project,
		"-o", "go-template='{{.spec.source.type}}'").ShouldPass().Out()
	return sourceType
}

// SourceLocationDC returns the source location from the deployment config
func (oc OcRunner) SourceLocationDC(componentName string, appName string, project string) string {
	sourceLocation := Cmd(oc.path, "get", "dc", componentName+"-"+appName, "--namespace", project,
		"-o", "go-template='{{index .metadata.annotations \"app.openshift.io/vcs-uri\"}}'").ShouldPass().Out()
	return sourceLocation
}

// SourceLocationBC returns the source location from the build config
func (oc OcRunner) SourceLocationBC(componentName string, appName string, project string) string {
	sourceLocation := Cmd(oc.path, "get", "bc", componentName+"-"+appName, "--namespace", project,
		"-o", "go-template='{{index .spec.source.git \"uri\"}}'").ShouldPass().Out()
	return sourceLocation
}

// checkForImageStream checks if there is a ImageStream with name and tag in openshift namespace
func (oc OcRunner) checkForImageStream(name string, tag string) bool {
	// first check if there is ImageStream with given name
	names := strings.Trim(Cmd(oc.path, "get", "is", "-n", "openshift",
		"-o", "jsonpath='{range .items[*]}{.metadata.name}{\"\\n\"}{end}'").ShouldPass().Out(), "'")
	scanner := bufio.NewScanner(strings.NewReader(names))
	namePresent := false
	for scanner.Scan() {
		if scanner.Text() == name {
			namePresent = true
		}
	}
	tagPresent := false
	// if there is a ImageStream check if there is a given tag
	if namePresent {
		tags := strings.Trim(Cmd(oc.path, "get", "is", name, "-n", "openshift",
			"-o", "jsonpath='{range .spec.tags[*]}{.name}{\"\\n\"}{end}'").ShouldPass().Out(), "'")
		scanner := bufio.NewScanner(strings.NewReader(tags))
		for scanner.Scan() {
			if scanner.Text() == tag {
				tagPresent = true
			}
		}
	}

	if tagPresent {
		return true
	}
	return false
}

// ImportImageFromRegistry import the required image of the respective component type from the specified registry
func (oc OcRunner) ImportImageFromRegistry(registry, image, cmpType, project string) {
	Cmd(oc.path, "--request-timeout", "5m", "import-image", cmpType, "--namespace="+project, fmt.Sprintf("--from=%s/%s", registry, image), "--confirm").ShouldPass()
	Cmd(oc.path, "annotate", fmt.Sprintf("istag/%s", cmpType), "--namespace="+project, "tags=builder", "--overwrite").ShouldPass()

}

// ImportJavaIS import the openjdk image which is used for jars
func (oc OcRunner) ImportJavaIS(project string) {
	// if ImageStream already exists, no need to do anything
	if oc.checkForImageStream("java", "8") {
		return
	}

	// we need to import the openjdk image which is used for jars because it's not available by default
	Cmd(oc.path, "--request-timeout", "5m", "import-image", "java:8",
		"--namespace="+project, "--from=registry.access.redhat.com/redhat-openjdk-18/openjdk18-openshift:1.8",
		"--confirm").ShouldPass()
	Cmd(oc.path, "annotate", "istag/java:8", "--namespace="+project,
		"tags=builder", "--overwrite").ShouldPass()
}

// ImportDotnet20IS import the dotnet image
func (oc OcRunner) ImportDotnet20IS(project string) {
	// if ImageStream already exists, no need to do anything
	if oc.checkForImageStream("dotnet", "2.0") {
		return
	}

	// we need to import the openjdk image which is used for jars because it's not available by default
	Cmd(oc.path, "--request-timeout", "5m", "import-image", "dotnet:2.0",
		"--namespace="+project, "--from=registry.centos.org/dotnet/dotnet-20-centos7",
		"--confirm").ShouldPass()
	Cmd(oc.path, "annotate", "istag/dotnet:2.0", "--namespace="+project,
		"tags=builder", "--overwrite").ShouldPass()
}

// EnvVarTest checks the component container env vars in the build config for git and deployment config for git/binary/local
// appTestName is the app of the app
// sourceType is the type of the source of the component i.e git/binary/local
func (oc OcRunner) EnvVarTest(resourceName string, sourceType string, envString string) {

	if sourceType == "git" {
		// checking the values of the env vars pairs in bc
		envVars := Cmd(oc.path, "get", "bc", resourceName,
			"-o", "go-template='{{range .spec.strategy.sourceStrategy.env}}{{.name}}{{.value}}{{end}}'").ShouldPass().Out()
		Expect(envVars).To(Equal(envString))
	}

	// checking the values of the env vars pairs in dc
	envVars := Cmd(oc.path, "get", "dc", resourceName,
		"-o", "go-template='{{range .spec.template.spec.containers}}{{range .env}}{{.name}}{{.value}}{{end}}{{end}}'").ShouldPass().Out()
	Expect(envVars).To(Equal(envString))
}

// GetRunningPodNameOfComp executes oc command and returns the running pod name of a deployed
// component by passing component name as a argument
func (oc OcRunner) GetRunningPodNameOfComp(compName string, namespace string) string {
	stdOut := Cmd(oc.path, "get", "pods", "--namespace", namespace, "--show-labels").ShouldPass().Out()
	re := regexp.MustCompile(`(` + compName + `-\S+)\s+\S+\s+Running.*deploymentconfig=` + compName)
	podName := re.FindStringSubmatch(stdOut)[1]
	return strings.TrimSpace(podName)
}

// GetRunningPodNameByComponent executes oc command and returns the running pod name of a deployed
// devfile component by passing component name as a argument
func (oc OcRunner) GetRunningPodNameByComponent(compName string, namespace string) string {
	selector := fmt.Sprintf("--selector=component=%s", compName)
	stdOut := Cmd(oc.path, "get", "pods", "--namespace", namespace, selector, "-o", "jsonpath={.items[*].metadata.name}").ShouldPass().Out()
	return strings.TrimSpace(stdOut)
}

// GetPVCSize executes oc command and returns the bound storage size
func (oc OcRunner) GetPVCSize(compName, storageName, namespace string) string {
	selector := fmt.Sprintf("--selector=app.kubernetes.io/storage-name=%s,app.kubernetes.io/instance=%s", storageName, compName)
	stdOut := Cmd(oc.path, "get", "pvc", "--namespace", namespace, selector, "-o", "jsonpath={.items[*].spec.resources.requests.storage}").ShouldPass().Out()
	return strings.TrimSpace(stdOut)
}

// GetPodInitContainers executes oc command and returns the init containers of the pod
func (oc OcRunner) GetPodInitContainers(compName string, namespace string) []string {
	selector := fmt.Sprintf("--selector=component=%s", compName)
	stdOut := Cmd(oc.path, "get", "pods", "--namespace", namespace, selector, "-o", "jsonpath={.items[*].spec.initContainers[*].name}").ShouldPass().Out()
	return strings.Split(stdOut, " ")
}

// GetRoute returns route URL
func (oc OcRunner) GetRoute(urlName string, appName string) string {
	session := CmdRunner(oc.path, "get", "routes", urlName+"-"+appName,
		"-o jsonpath={.spec.host}")
	Eventually(session).Should(gexec.Exit(0))
	return strings.TrimSpace(string(session.Wait().Out.Contents()))
}

// GetToken returns current user token
func (oc OcRunner) GetToken() string {
	session := CmdRunner(oc.path, "whoami", "-t")
	Eventually(session).Should(gexec.Exit(0))
	return strings.TrimSpace(string(session.Wait().Out.Contents()))
}

// LoginUsingToken returns output after successful login
func (oc OcRunner) LoginUsingToken(token string) string {
	session := CmdRunner(oc.path, "login", "--token", token)
	Eventually(session).Should(gexec.Exit(0))
	return strings.TrimSpace(string(session.Wait().Out.Contents()))
}

// GetLoginUser returns current user name
func (oc OcRunner) GetLoginUser() string {
	user := Cmd(oc.path, "whoami").ShouldPass().Out()
	return strings.TrimSpace(user)
}

// ServiceInstanceStatus returns service instance
func (oc OcRunner) ServiceInstanceStatus(serviceInstanceName string) string {
	serviceinstance := Cmd(oc.path, "get", "serviceinstance", serviceInstanceName,
		"-o", "go-template='{{ (index .status.conditions 0).reason}}'").ShouldPass().Out()
	return strings.TrimSpace(serviceinstance)
}

// GetVolumeMountNamesandPathsFromContainer returns the volume name and mount path in the format name:path\n
func (oc OcRunner) GetVolumeMountNamesandPathsFromContainer(deployName string, containerName, namespace string) string {
	volumeName := Cmd(oc.path, "get", "deploy", deployName, "--namespace", namespace,
		"-o", "go-template="+
			"{{range .spec.template.spec.containers}}{{if eq .name \""+containerName+
			"\"}}{{range .volumeMounts}}{{.name}}{{\":\"}}{{.mountPath}}{{\"\\n\"}}{{end}}{{end}}{{end}}").ShouldPass().Out()

	return strings.TrimSpace(volumeName)
}

// GetContainerEnv returns the container env in the format name:value\n
func (oc OcRunner) GetContainerEnv(podName, containerName, namespace string) string {
	containerEnv := Cmd(oc.path, "get", "po", podName, "--namespace", namespace,
		"-o", "go-template="+
			"{{range .spec.containers}}{{if eq .name \""+containerName+
			"\"}}{{range .env}}{{.name}}{{\":\"}}{{.value}}{{\"\\n\"}}{{end}}{{end}}{{end}}").ShouldPass().Out()

	return strings.TrimSpace(containerEnv)
}

// GetVolumeMountName returns the name of the volume
func (oc OcRunner) GetVolumeMountName(dcName string, namespace string) string {
	volumeName := Cmd(oc.path, "get", "dc", dcName, "--namespace", namespace,
		"-o", "go-template='"+
			"{{range .spec.template.spec.containers}}"+
			"{{range .volumeMounts}}{{.name}}{{end}}{{end}}'").ShouldPass().Out()

	return strings.TrimSpace(volumeName)
}

// GetVolumeMountPath returns the path of the volume mount
func (oc OcRunner) GetVolumeMountPath(dcName string, namespace string) string {
	volumePaths := Cmd(oc.path, "get", "dc", dcName, "--namespace", namespace,
		"-o", "go-template='"+
			"{{range .spec.template.spec.containers}}"+
			"{{range .volumeMounts}}{{.mountPath}} {{end}}{{end}}'").ShouldPass().Out()

	return strings.TrimSpace(volumePaths)
}

// GetEnvFromEntry returns envFrom entry
func (oc OcRunner) GetEnvFromEntry(componentName string, appName string, projectName string) string {
	envFromOut := Cmd(oc.path, "get", "dc", componentName+"-"+appName, "--namespace", projectName,
		"-o", "jsonpath='{.spec.template.spec.containers[0].envFrom}'").ShouldPass().Out()
	return strings.TrimSpace(envFromOut)
}

// GetEnvs returns all env variables in deployment config
func (oc OcRunner) GetEnvs(componentName string, appName string, projectName string) map[string]string {
	var mapOutput = make(map[string]string)

	output := Cmd(oc.path, "get", "dc", componentName+"-"+appName, "--namespace", projectName,
		"-o", "jsonpath='{range .spec.template.spec.containers[0].env[*]}{.name}:{.value}{\"\\n\"}{end}'").ShouldPass().Out()

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimPrefix(line, "'")
		splits := strings.Split(line, ":")
		name := splits[0]
		value := strings.Join(splits[1:], ":")
		mapOutput[name] = value
	}
	return mapOutput
}

func (oc OcRunner) GetEnvsDevFileDeployment(componentName, appName, projectName string) map[string]string {
	var mapOutput = make(map[string]string)

	selector := fmt.Sprintf("--selector=%s=%s,%s=%s", labels.ComponentLabel, componentName, applabels.ApplicationLabel, appName)
	output := Cmd(oc.path, "get", "deployment", selector, "--namespace", projectName,
		"-o", "jsonpath='{range .items[0].spec.template.spec.containers[0].env[*]}{.name}:{.value}{\"\\n\"}{end}'").ShouldPass().Out()

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimPrefix(line, "'")
		splits := strings.Split(line, ":")
		name := splits[0]
		value := strings.Join(splits[1:], ":")
		mapOutput[name] = value
	}
	return mapOutput
}

// GetEnvRefNames gets the ref values from the envFroms of the deployment belonging to the given data
func (oc OcRunner) GetEnvRefNames(componentName, appName, projectName string) []string {
	return GetEnvRefNames(oc.path, componentName, appName, projectName)
}

// WaitForDCRollout wait for DeploymentConfig to finish active rollout
// timeout is a maximum wait time in seconds
func (oc OcRunner) WaitForDCRollout(dcName string, project string, timeout time.Duration) {
	session := CmdRunner(oc.path, "rollout", "status",
		"-w",
		"-n", project,
		"dc", dcName)

	Eventually(session).Should(gexec.Exit(0), runningCmd(session.Command))
	session.Wait(timeout)
}

// WaitAndCheckForExistence wait for the given and checks if the given resource type gets deleted on the cluster
func (oc OcRunner) WaitAndCheckForExistence(resourceType, namespace string, timeoutMinutes int) bool {
	pingTimeout := time.After(time.Duration(timeoutMinutes) * time.Minute)
	// this is a test package so time.Tick() is acceptable
	// nolint
	tick := time.Tick(time.Second)
	for {
		select {
		case <-pingTimeout:
			Fail(fmt.Sprintf("Timeout after %d minutes", timeoutMinutes))

		case <-tick:
			session := CmdRunner(oc.path, "get", resourceType, "--namespace", namespace)
			Eventually(session).Should(gexec.Exit(0))
			// https://github.com/kubernetes/kubectl/issues/847
			output := string(session.Wait().Err.Contents())

			if strings.Contains(strings.ToLower(output), "no resources found") {
				return true
			}
		}
	}
}

// GetServices gets services on the cluster
func (oc OcRunner) GetServices(namespace string) string {
	session := CmdRunner(oc.path, "get", "services", "--namespace", namespace)
	Eventually(session).Should(gexec.Exit(0))
	output := string(session.Wait().Out.Contents())
	return output
}

// VerifyResourceDeleted verifies if the given resource is deleted from cluster
func (oc OcRunner) VerifyResourceDeleted(ri ResourceInfo) {
	session := CmdRunner(oc.path, "get", ri.ResourceType, "--namespace", ri.Namespace)
	Eventually(session).Should(gexec.Exit(0))
	output := string(session.Wait().Out.Contents())
	Expect(output).NotTo(ContainSubstring(ri.ResourceName))
}

func (oc OcRunner) VerifyResourceToBeDeleted(ri ResourceInfo) {
	deletedOrMarkedToDelete := func() bool {
		session := CmdRunner(oc.path, "get", ri.ResourceType, ri.ResourceName, "--namespace", ri.Namespace, "-o", "jsonpath='{.metadata.deletionTimestamp}'")
		exit := session.Wait().ExitCode()
		if exit == 1 {
			// resources does not exist
			return true
		}
		content := session.Wait().Out.Contents()
		// resource is marked for deletion
		return len(content) > 0
	}
	Expect(deletedOrMarkedToDelete()).To(BeTrue())
}

// CreateRandNamespaceProject create new project
func (oc OcRunner) CreateRandNamespaceProject() string {
	projectName := SetProjectName()
	oc.createRandNamespaceProject(projectName)
	return projectName
}

// CreateRandNamespaceProjectOfLength creates a new project with name of length i
func (oc OcRunner) CreateRandNamespaceProjectOfLength(i int) string {
	projectName := RandString(i)
	oc.createRandNamespaceProject(projectName)
	return projectName
}

func (oc OcRunner) createRandNamespaceProject(projectName string) string {
	fmt.Fprintf(GinkgoWriter, "Creating a new project: %s\n", projectName)
	session := Cmd("odo", "project", "create", projectName, "-w", "-v4").ShouldPass().Out()
	Expect(session).To(ContainSubstring("New project created"))
	Expect(session).To(ContainSubstring(projectName))
	return projectName
}

func (oc OcRunner) SetProject(namespace string) string {
	fmt.Fprintf(GinkgoWriter, "Setting project: %s\n", namespace)
	Cmd("odo", "project", "set", namespace).ShouldPass()
	return namespace
}

// DeleteNamespaceProject deletes a specified project in oc cluster
func (oc OcRunner) DeleteNamespaceProject(projectName string) {
	fmt.Fprintf(GinkgoWriter, "Deleting project: %s\n", projectName)
	session := Cmd("odo", "project", "delete", projectName, "-f").ShouldPass().Out()
	Expect(session).To(ContainSubstring("Deleted project : " + projectName))
}

func (oc OcRunner) GetAllPVCNames(namespace string) []string {
	session := CmdRunner(oc.path, "get", "pvc", "--namespace", namespace, "-o", "jsonpath={.items[*].metadata.name}")
	Eventually(session).Should(gexec.Exit(0))
	output := string(session.Wait().Out.Contents())
	if output == "" {
		return []string{}
	}
	return strings.Split(output, " ")
}

// DeletePod deletes a specified pod in the namespace
func (oc OcRunner) DeletePod(podName string, namespace string) {
	Cmd(oc.path, "delete", "pod", "--namespace", namespace, podName).ShouldPass()
}

//GetAllPodsInNs gets the list of pods in given namespace. It waits for reasonable amount of time for pods to come up
func (oc OcRunner) GetAllPodsInNs(namespace string) string {
	args := []string{"get", "pods", "-n", namespace}
	noResourcesMsg := fmt.Sprintf("No resources found in %s namespace", namespace)
	oc.WaitForRunnerCmdOut(args, 1, true, func(output string) bool {
		return !strings.Contains(output, noResourcesMsg)
	}, true)
	return Cmd(oc.path, args...).ShouldPass().Out()
}

//StatFileInPod returns stat result of filepath in pod of given component, in a given app, in a given project.
//It also strips access time information as it vaires accross file systems/kernel configs, and we are not interested
//in it anyway
func (oc OcRunner) StatFileInPod(cmpName, appName, project, filepath string) string {
	var result string
	oc.CheckCmdOpInRemoteCmpPod(
		cmpName,
		appName,
		project,
		[]string{"stat", filepath},
		func(cmdOp string, err error) bool {
			//strip out access info as
			// 1. Touching a file (such as running it in a script) modifies access times. This gives wrong value on mounts without noatime
			// 2. We are not interested in Access info anyway.
			re := regexp.MustCompile("(?m)[\r\n]+^.*Access.*$")
			result = re.ReplaceAllString(cmdOp, "")
			return true
		},
	)
	return result
}

// WaitAndCheckForTerminatingState waits for the given interval
// and checks if the given resource type has been deleted on the cluster or is in the terminating state
func (oc OcRunner) WaitAndCheckForTerminatingState(resourceType, namespace string, timeoutMinutes int) bool {
	return WaitAndCheckForTerminatingState(oc.path, resourceType, namespace, timeoutMinutes)
}

// GetAnnotationsDeployment gets the annotations from the deployment
// belonging to the given component, app and project
func (oc OcRunner) GetAnnotationsDeployment(componentName, appName, projectName string) map[string]string {
	return GetAnnotationsDeployment(oc.path, componentName, appName, projectName)
}

func (oc OcRunner) PodsShouldBeRunning(project string, regex string) {
	// now verify if the pods for the operator have started
	pods := oc.GetAllPodsInNs(project)
	// Look for pods with specified regex
	pod := regexp.MustCompile(regex).FindString(pods)
	args := []string{"get", "pods", pod, "-o", "template=\"{{.status.phase}}\"", "-n", project}
	oc.WaitForRunnerCmdOut(args, 1, true, func(output string) bool {
		return strings.Contains(output, "Running")
	})
}

// WaitForRunnerCmdOut runs "oc" command until it gets
// the expected output.
// It accepts 4 arguments
// args (arguments to the program)
// timeout (the time to wait for the output)
// errOnFail (flag to set if test should fail if command fails)
// check (function with output check logic)
// It times out if the command doesn't fetch the
// expected output  within the timeout period.
func (oc OcRunner) WaitForRunnerCmdOut(args []string, timeout int, errOnFail bool, check func(output string) bool, includeStdErr ...bool) bool {
	pingTimeout := time.After(time.Duration(timeout) * time.Minute)
	// this is a test package so time.Tick() is acceptable
	// nolint
	tick := time.Tick(time.Second)
	for {
		select {
		case <-pingTimeout:
			Fail(fmt.Sprintf("Timeout after %v minutes", timeout))

		case <-tick:
			session := CmdRunner(oc.path, args...)
			if errOnFail {
				Eventually(session).Should(gexec.Exit(0), runningCmd(session.Command))
			} else {
				Eventually(session).Should(gexec.Exit(), runningCmd(session.Command))
			}
			session.Wait()
			output := string(session.Out.Contents())

			if len(includeStdErr) > 0 && includeStdErr[0] {
				output += "\n"
				output += string(session.Err.Contents())
			}
			if check(strings.TrimSpace(string(output))) {
				return true
			}
		}
	}
}

// CreateSecret takes secret name, password and the namespace where we want to create the specific secret into the cluster
func (oc OcRunner) CreateSecret(secretName, secretPass, project string) {
	Cmd(oc.path, "create", "secret", "generic", secretName, "--from-literal=password="+secretPass, "-n", project).ShouldPass()
}

// GetSecrets gets all the secrets belonging to the project
func (oc OcRunner) GetSecrets(project string) string {
	return GetSecrets(oc.path, project)
}
