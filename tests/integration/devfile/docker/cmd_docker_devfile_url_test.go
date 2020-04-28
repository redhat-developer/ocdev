package docker

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/openshift/odo/tests/helper"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("odo docker devfile url command tests", func() {
	var projectDirPath, context, currentWorkingDirectory, cmpName string
	var projectDir = "/projectDir"
	dockerClient := helper.NewDockerRunner("docker")

	// This is run after every Spec (It)
	var _ = BeforeEach(func() {
		SetDefaultEventuallyTimeout(10 * time.Minute)
		context = helper.CreateNewContext()
		currentWorkingDirectory = helper.Getwd()
		projectDirPath = context + projectDir
		cmpName = helper.RandString(6)
		helper.Chdir(context)
		os.Setenv("GLOBALODOCONFIG", filepath.Join(context, "config.yaml"))
		helper.CmdShouldPass("odo", "preference", "set", "Experimental", "true")
		helper.CmdShouldPass("odo", "preference", "set", "pushtarget", "docker")
	})

	// Clean up after the test
	// This is run after every Spec (It)
	var _ = AfterEach(func() {
		// Stop all containers labeled with the component name
		label := "component=" + cmpName
		dockerClient.StopContainers(label)

		helper.Chdir(currentWorkingDirectory)
		helper.DeleteDir(context)
		os.Unsetenv("GLOBALODOCONFIG")
	})

	Context("Creating urls", func() {
		It("create should pass", func() {
			var stdout string
			helper.CmdShouldPass("git", "clone", "https://github.com/che-samples/web-nodejs-sample.git", projectDirPath)
			helper.Chdir(projectDirPath)

			helper.CmdShouldPass("odo", "create", "nodejs", cmpName)
			stdout = helper.CmdShouldPass("odo", "url", "create")
			helper.MatchAllInOutput(stdout, []string{"local-" + cmpName + "-3000", "created for component"})
			stdout = helper.CmdShouldPass("odo", "push", "--devfile", "devfile.yaml")
			Expect(stdout).To(ContainSubstring("Changes successfully pushed to component"))
		})

		It("create with now flag should pass", func() {
			var stdout string
			url1 := helper.RandString(5)
			helper.CmdShouldPass("git", "clone", "https://github.com/che-samples/web-nodejs-sample.git", projectDirPath)
			helper.Chdir(projectDirPath)

			helper.CmdShouldPass("odo", "create", "nodejs", cmpName)
			stdout = helper.CmdShouldPass("odo", "url", "create", url1, "--now")
			helper.MatchAllInOutput(stdout, []string{url1, "created for component", "Changes successfully pushed to component"})
		})

		It("create with same url name under a different push target should fail", func() {
			var stdout string
			url1 := helper.RandString(5)
			url2 := helper.RandString(5)
			helper.CmdShouldPass("git", "clone", "https://github.com/che-samples/web-nodejs-sample.git", projectDirPath)
			helper.Chdir(projectDirPath)

			helper.CmdShouldPass("odo", "create", "nodejs", cmpName)
			helper.CmdShouldPass("odo", "url", "create", url1)

			// url1 exists with push target set to docker, create url with same name should fail if push target is set to kube
			helper.CmdShouldPass("odo", "preference", "set", "pushtarget", "kube", "-f")
			stdout = helper.CmdShouldFail("odo", "url", "create", url1, "--host", "1.2.3.4.com")
			Expect(stdout).To(ContainSubstring("already exists for a different push target"))

			// url2 exists with push target set to kube, create url with same name should fail if push target is set to docker
			helper.CmdShouldPass("odo", "url", "create", url2, "--host", "1.2.3.4.com")
			helper.CmdShouldPass("odo", "preference", "set", "pushtarget", "docker", "-f")
			stdout = helper.CmdShouldFail("odo", "url", "create", url2)
			Expect(stdout).To(ContainSubstring("already exists for a different push target"))

		})

		It("should be able to do a GET on the URL after a successful push", func() {
			helper.CmdShouldPass("git", "clone", "https://github.com/che-samples/web-nodejs-sample.git", projectDirPath)
			helper.Chdir(projectDirPath)

			helper.CmdShouldPass("odo", "create", "nodejs", cmpName)

			helper.CmdShouldPass("odo", "url", "create", cmpName)

			output := helper.CmdShouldPass("odo", "push", "--devfile", "devfile.yaml")
			helper.MatchAllInOutput(output, []string{"Executing devbuild command", "Executing devrun command"})

			url := strings.TrimSpace(helper.ExtractSubString(output, "127.0.0.1", "created"))

			helper.HttpWaitFor("http://"+url, "Hello World!", 30, 1)
		})
	})

})
