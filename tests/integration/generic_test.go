package integration

import (
	"os"
	"path/filepath"
	"regexp"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/odo/tests/helper"
)

var _ = Describe("odo generic", func() {
	// TODO: A neater way to provide odo path. Currently we assume \
	// odo and oc in $PATH already.
	var project string
	var context string
	var originalDir string
	var oc helper.OcRunner
	var testPHPGitURL = "https://github.com/appuio/example-php-sti-helloworld"
	var testLongURLName = "long-url-name-long-url-name-long-url-name-long-url-name-long-url-name"

	BeforeEach(func() {
		oc = helper.NewOcRunner("oc")
		SetDefaultEventuallyTimeout(10 * time.Minute)
		SetDefaultConsistentlyDuration(30 * time.Second)
	})

	Context("When executing catalog list without component directory", func() {
		It("should list all component catalogs", func() {
			stdOut := helper.CmdShouldPass("odo", "catalog", "list", "components")
			Expect(stdOut).To(ContainSubstring("dotnet"))
			Expect(stdOut).To(ContainSubstring("nginx"))
			Expect(stdOut).To(ContainSubstring("php"))
			Expect(stdOut).To(ContainSubstring("ruby"))
			Expect(stdOut).To(ContainSubstring("wildfly"))
		})
	})

	Context("creating component with an application and url", func() {
		JustBeforeEach(func() {
			project = helper.CreateRandProject()
			context = helper.CreateNewContext()
			os.Setenv("GLOBALODOCONFIG", filepath.Join(context, "config.yaml"))
			originalDir = helper.Getwd()
			helper.Chdir(context)
		})

		JustAfterEach(func() {
			helper.DeleteProject(project)
			helper.DeleteDir(context)
			os.Unsetenv("GLOBALODOCONFIG")
			helper.Chdir(originalDir)
		})
		It("should create the component in default application", func() {
			helper.CmdShouldPass("odo", "create", "php", "testcmp", "--app", "e2e-xyzk", "--project", project, "--git", testPHPGitURL)
			helper.CmdShouldPass("odo", "config", "set", "Ports", "8080/TCP")
			helper.CmdShouldPass("odo", "push")
			oc.VerifyCmpName("testcmp", project)
			oc.VerifyAppNameOfComponent("testcmp", "e2e-xyzk", project)
			helper.CmdShouldPass("odo", "app", "delete", "e2e-xyzk", "-f")
		})
	})

	Context("should list applications in other project", func() {
		JustBeforeEach(func() {
			project = helper.CreateRandProject()
			context = helper.CreateNewContext()
			os.Setenv("GLOBALODOCONFIG", filepath.Join(context, "config.yaml"))
		})

		JustAfterEach(func() {
			helper.DeleteProject(project)
			helper.DeleteDir(context)
			os.Unsetenv("GLOBALODOCONFIG")
		})
		It("should be able to create a php component with application created", func() {
			helper.CmdShouldPass("odo", "create", "php", "testcmp", "--app", "testing", "--project", project, "--ref", "master", "--git", testPHPGitURL, "--context", context)
			helper.CmdShouldPass("odo", "push", "--context", context)
			currentProject := helper.CreateRandProject()
			currentAppNames := helper.CmdShouldPass("odo", "app", "list", "--project", currentProject)
			Expect(currentAppNames).To(ContainSubstring("There are no applications deployed in the project '" + currentProject + "'"))
			appNames := helper.CmdShouldPass("odo", "app", "list", "--project", project)
			Expect(appNames).To(ContainSubstring("testing"))
			helper.DeleteProject(currentProject)
		})
	})

	Context("when running odo push with flag --show-log", func() {
		JustBeforeEach(func() {
			project = helper.CreateRandProject()
			context = helper.CreateNewContext()
			os.Setenv("GLOBALODOCONFIG", filepath.Join(context, "config.yaml"))
		})

		JustAfterEach(func() {
			helper.DeleteProject(project)
			os.RemoveAll(context)
			os.Unsetenv("GLOBALODOCONFIG")
		})
		It("should be able to push changes", func() {
			helper.CopyExample(filepath.Join("source", "nodejs"), context)
			helper.CmdShouldPass("odo", "create", "nodejs", "nodejs", "--project", project, "--context", context)

			// Push the changes with --show-log
			getLogging := helper.CmdShouldPass("odo", "push", "--show-log", "--context", context)
			Expect(getLogging).To(ContainSubstring("Building component"))
		})
	})

	Context("deploying a component with a specific image name", func() {
		JustBeforeEach(func() {
			project = helper.CreateRandProject()
			context = helper.CreateNewContext()
			os.Setenv("GLOBALODOCONFIG", filepath.Join(context, "config.yaml"))
		})

		JustAfterEach(func() {
			helper.DeleteProject(project)
			os.RemoveAll(context)
			os.Unsetenv("GLOBALODOCONFIG")
		})
		It("should deploy the component", func() {
			helper.CmdShouldPass("git", "clone", "https://github.com/openshift/nodejs-ex", context+"/nodejs-ex")
			helper.CmdShouldPass("odo", "create", "nodejs:latest", "testversioncmp", "--project", project, "--context", context+"/nodejs-ex")
			helper.CmdShouldPass("odo", "push", "--context", context+"/nodejs-ex")
			helper.CmdShouldPass("odo", "delete", "-f", "--context", context+"/nodejs-ex")
		})
	})

	Context("When deleting two project one after the other", func() {
		It("should be able to delete sequentially", func() {
			project1 := helper.CreateRandProject()
			project2 := helper.CreateRandProject()

			helper.DeleteProject(project2)
			helper.DeleteProject(project1)
		})
	})

	Context("When deleting three project one after the other in opposite order", func() {
		It("should be able to delete", func() {
			project1 := helper.CreateRandProject()
			project2 := helper.CreateRandProject()
			project3 := helper.CreateRandProject()

			helper.DeleteProject(project1)
			helper.DeleteProject(project2)
			helper.DeleteProject(project3)

		})
	})

	Context("when executing odo version command", func() {
		It("should show the version of odo major components including server login URL", func() {
			odoVersion := helper.CmdShouldPass("odo", "version")
			reOdoVersion := regexp.MustCompile(`^odo\s*v[0-9]+.[0-9]+.[0-9]+(?:-\w+)?\s*\(\w+\)`)
			odoVersionStringMatch := reOdoVersion.MatchString(odoVersion)
			rekubernetesVersion := regexp.MustCompile(`Kubernetes:\s*v[0-9]+.[0-9]+.[0-9]+\+\w+`)
			kubernetesVersionStringMatch := rekubernetesVersion.MatchString(odoVersion)
			reServerURL := regexp.MustCompile(`Server:\s*https:\/\/(.+\.com|([0-9]+.){3}[0-9]+):[0-9]{4}`)
			serverURLStringMatch := reServerURL.MatchString(odoVersion)
			Expect(odoVersionStringMatch).Should(BeTrue())
			Expect(kubernetesVersionStringMatch).Should(BeTrue())
			Expect(serverURLStringMatch).Should(BeTrue())
		})
	})

	Context("prevent the user from creating a URL with name that has more than 63 characters", func() {
		var originalDir string

		JustBeforeEach(func() {
			project = helper.CreateRandProject()
			context = helper.CreateNewContext()
			originalDir = helper.Getwd()
			helper.Chdir(context)
		})

		JustAfterEach(func() {
			helper.DeleteProject(project)
			helper.Chdir(originalDir)
			helper.DeleteDir(context)
		})
		It("should not allow creating a URL with long name", func() {
			helper.CmdShouldPass("odo", "create", "nodejs", "--project", project)
			stdOut := helper.CmdShouldFail("odo", "url", "create", testLongURLName, "--port", "8080")
			Expect(stdOut).To(ContainSubstring("url name must be shorter than 63 characters"))
		})
	})
})
