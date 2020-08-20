package template

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openshift/odo/tests/helper"
)

// following command will tests in Describe section below in parallel (in 2 nodes)
// ginkgo -nodes=2 -focus="Example of a clean test" slowSpecThreshold=120 -randomizeAllSpecs  tests/e2e/
var _ = Describe("Example of a clean test", func() {
	// new clean project and context for each test
	var project string
	var context string

	// current directory and component name (before any test runs) so that it can be restored  after all testing is done
	var originalDir string
	var cmpName string

	BeforeEach(func() {
		// Set default timeout for Eventually assertions
		// commands like odo push, might take a long time
		SetDefaultEventuallyTimeout(10 * time.Minute)
		context = helper.CreateNewContext()
		os.Setenv("GLOBALODOCONFIG", filepath.Join(context, "config.yaml"))
		project = helper.CreateRandProject()

		// we will be testing components that are created from the current directory
		// switch to the clean context dir before each test
		originalDir = helper.Getwd()
		helper.Chdir(context)

	})

	AfterEach(func() {
		helper.DeleteProject(project)
		// go back to original directory after each test
		helper.Chdir(originalDir)
		helper.DeleteDir(context)
		os.Unsetenv("GLOBALODOCONFIG")
	})

	Context("when --project flag is used", func() {
		JustBeforeEach(func() {
			cmpName = "nodejs"
		})

		It("create local nodejs component and push code", func() {
			helper.CopyExample(filepath.Join("source", "nodejs"), context)
			helper.CmdShouldPass("odo", "component", "create", "nodejs", cmpName, "--project", project)
			// verify that config was properly created
			helper.ValidateLocalCmpExist(context, "Type,nodejs", "Project,"+project, "Application,app")
			output := helper.CmdShouldPass("odo", "push")
			Expect(output).To(ContainSubstring("Changes successfully pushed to component"))
		})

		It("create, push and list local nodejs component", func() {
			appName := "testing"
			helper.CopyExample(filepath.Join("source", "nodejs"), context)
			helper.CmdShouldPass("odo", "component", "create", "nodejs", cmpName, "--app", appName, "--project", project, "--context", context)

			// verify that config was properly created
			helper.ValidateLocalCmpExist(context, "Type,nodejs", "Project,"+project, "Application,"+appName)
			helper.CmdShouldPass("odo", "push")

			// list the component name
			cmpListOutput := helper.CmdShouldPass("odo", "list", "--app", appName, "--project", project)
			Expect(cmpListOutput).To(ContainSubstring(cmpName))
		})

	})
})
