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
	var context, currentWorkingDirectory, cmpName string
	dockerClient := helper.NewDockerRunner("docker")

	// This is run after every Spec (It)
	var _ = BeforeEach(func() {
		SetDefaultEventuallyTimeout(10 * time.Minute)
		context = helper.CreateNewContext()
		currentWorkingDirectory = helper.Getwd()
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
			helper.CmdShouldPass("odo", "create", "nodejs", cmpName)

			helper.CopyExample(filepath.Join("source", "devfiles", "nodejs", "project"), context)
			helper.CopyExampleDevFile(filepath.Join("source", "devfiles", "nodejs", "devfile.yaml"), filepath.Join(context, "devfile.yaml"))

			stdout = helper.CmdShouldPass("odo", "url", "create")
			helper.MatchAllInOutput(stdout, []string{cmpName + "-3000", "created for component"})
			stdout = helper.CmdShouldPass("odo", "push", "--devfile", "devfile.yaml")
			Expect(stdout).To(ContainSubstring("Changes successfully pushed to component"))
		})

		It("create with now flag should pass", func() {
			var stdout string
			url1 := helper.RandString(5)

			helper.CmdShouldPass("odo", "create", "nodejs", cmpName)

			helper.CopyExample(filepath.Join("source", "devfiles", "nodejs", "project"), context)
			helper.CopyExampleDevFile(filepath.Join("source", "devfiles", "nodejs", "devfile.yaml"), filepath.Join(context, "devfile.yaml"))

			stdout = helper.CmdShouldPass("odo", "url", "create", url1, "--now")
			helper.MatchAllInOutput(stdout, []string{url1, "created for component", "Changes successfully pushed to component"})
		})

		It("create with same url name should fail", func() {
			var stdout string
			url1 := helper.RandString(5)

			helper.CmdShouldPass("odo", "create", "nodejs", cmpName)

			helper.CopyExample(filepath.Join("source", "devfiles", "nodejs", "project"), context)
			helper.CopyExampleDevFile(filepath.Join("source", "devfiles", "nodejs", "devfile.yaml"), filepath.Join(context, "devfile.yaml"))

			helper.CmdShouldPass("odo", "url", "create", url1)

			stdout = helper.CmdShouldFail("odo", "url", "create", url1)
			Expect(stdout).To(ContainSubstring("the url " + url1 + " already exists"))

		})

		It("should be able to do a GET on the URL after a successful push", func() {
			helper.CmdShouldPass("odo", "create", "nodejs", cmpName)

			helper.CopyExample(filepath.Join("source", "devfiles", "nodejs", "project"), context)
			helper.CopyExampleDevFile(filepath.Join("source", "devfiles", "nodejs", "devfile.yaml"), filepath.Join(context, "devfile.yaml"))

			helper.CmdShouldPass("odo", "url", "create", cmpName)

			output := helper.CmdShouldPass("odo", "push", "--devfile", "devfile.yaml")
			helper.MatchAllInOutput(output, []string{"Executing devbuild command", "Executing devrun command"})

			url := strings.TrimSpace(helper.ExtractSubString(output, "127.0.0.1", "created"))

			helper.HttpWaitFor("http://"+url, "Hello World!", 30, 1)
		})
	})

	Context("Switching pushtarget", func() {
		It("switch from docker to kube, odo push should display warning", func() {
			var stdout string

			helper.CmdShouldPass("odo", "create", "nodejs", cmpName)

			helper.CopyExample(filepath.Join("source", "devfiles", "nodejs", "project"), context)
			helper.CopyExampleDevFile(filepath.Join("source", "devfiles", "nodejs", "devfile.yaml"), filepath.Join(context, "devfile.yaml"))

			helper.CmdShouldPass("odo", "url", "create")

			helper.CmdShouldPass("odo", "preference", "set", "pushtarget", "kube", "-f")
			session := helper.CmdRunner("odo", "push", "--devfile", "devfile.yaml")
			stdout = string(session.Wait().Out.Contents())
			stderr := string(session.Wait().Err.Contents())
			Expect(stderr).To(ContainSubstring("Found a URL defined for Docker, but no valid URLs for Kubernetes."))
			Expect(stdout).To(ContainSubstring("Changes successfully pushed to component"))
		})

		It("switch from kube to docker, odo push should display warning", func() {
			var stdout string
			helper.CmdShouldPass("odo", "preference", "set", "pushtarget", "kube", "-f")
			helper.CmdShouldPass("odo", "create", "nodejs", cmpName)

			helper.CopyExample(filepath.Join("source", "devfiles", "nodejs", "project"), context)
			helper.CopyExampleDevFile(filepath.Join("source", "devfiles", "nodejs", "devfile.yaml"), filepath.Join(context, "devfile.yaml"))

			helper.CmdShouldPass("odo", "url", "create", "--host", "1.2.3.4.com", "--ingress")

			helper.CmdShouldPass("odo", "preference", "set", "pushtarget", "docker", "-f")
			session := helper.CmdRunner("odo", "push", "--devfile", "devfile.yaml")
			stdout = string(session.Wait().Out.Contents())
			stderr := string(session.Wait().Err.Contents())
			Expect(stderr).To(ContainSubstring("Found a URL defined for Kubernetes, but no valid URLs for Docker."))
			Expect(stdout).To(ContainSubstring("Changes successfully pushed to component"))

		})
	})

})
