package helper

import "github.com/onsi/gomega/gexec"

// CliRunner requires functions which are common for oc and kubectl
// By abstracting these functions into an interface, it handles the cli runner and calls the functions specified to particular cluster only
type CliRunner interface {
	Run(args ...string) *gexec.Session
	ExecListDir(podName string, projectName string, dir string) string
	Exec(podName string, projectName string, args ...string) string
	CheckCmdOpInRemoteDevfilePod(podName string, containerName string, prjName string, cmd []string, checkOp func(cmdOp string, err error) bool) bool
	GetRunningPodNameByComponent(compName string, namespace string) string
	GetVolumeMountNamesandPathsFromContainer(deployName string, containerName, namespace string) string
	WaitAndCheckForExistence(resourceType, namespace string, timeoutMinutes int) bool
	GetServices(namespace string) string
	CreateRandNamespaceProject() string
	CreateRandNamespaceProjectOfLength(i int) string
	SetProject(namespace string) string
	DeleteNamespaceProject(projectName string)
	DeletePod(podName string, projectName string)
	GetEnvsDevFileDeployment(componentName, appName, projectName string) map[string]string
	GetEnvRefNames(componentName, appName, projectName string) []string
	GetPVCSize(compName, storageName, namespace string) string
	GetAllPVCNames(namespace string) []string
	GetPodInitContainers(compName, namespace string) []string
	GetContainerEnv(podName, containerName, namespace string) string
	WaitAndCheckForTerminatingState(resourceType, namespace string, timeoutMinutes int) bool
	VerifyResourceDeleted(ri ResourceInfo)
	VerifyResourceToBeDeleted(ri ResourceInfo)
	GetAnnotationsDeployment(cmpName, appName, projectName string) map[string]string
	GetAllPodsInNs(namespace string) string
	WaitForRunnerCmdOut(args []string, timeout int, errOnFail bool, check func(output string) bool, includeStdErr ...bool) bool
	PodsShouldBeRunning(project string, regex string)
	CreateSecret(secretName, secretPass, project string)
	GetSecrets(project string) string
}
