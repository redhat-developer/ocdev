package project

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/odo/tests/helper"
)

var _ = Describe("odo project command tests", func() {
	var project string
	var context string

	// This is run after every Spec (It)
	var _ = BeforeEach(func() {
		SetDefaultEventuallyTimeout(10 * time.Minute)
		SetDefaultConsistentlyDuration(30 * time.Second)
		context = helper.CreateNewContext()
		os.Setenv("GLOBALODOCONFIG", filepath.Join(context, "config.yaml"))
		project = helper.CreateRandProject()
	})

	// Clean up after the test
	// This is run after every Spec (It)
	var _ = AfterEach(func() {
		helper.DeleteProject(project)
		helper.DeleteDir(context)
		os.Unsetenv("GLOBALODOCONFIG")
	})

	Context("when running help for project command", func() {
		It("should display the help", func() {
			appHelp := helper.CmdShouldPass("odo", "project", "-h")
			Expect(appHelp).To(ContainSubstring("Perform project operations"))
		})
	})

	Context("when running project command app parameter in directory that doesn't contain .odo config directory", func() {
		It("should successfully execute list along with machine readable output", func() {
			listOutput := helper.CmdShouldPass("odo", "project", "list")
			Expect(listOutput).To(ContainSubstring(project))

			// project deletion doesn't happen immediately, so we test subset of the string
			listOutputJson := helper.CmdShouldPass("odo", "project", "list", "-o", "json")
			Expect(listOutputJson).To(ContainSubstring(`{"kind":"Project","apiVersion":"odo.openshift.io/v1alpha1","metadata":{"name":"` + project + `","namespace":"` + project + `","creationTimestamp":null},"spec":{},"status":{"active":true}}`))
		})
	})
})
