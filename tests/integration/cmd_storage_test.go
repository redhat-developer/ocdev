package integration

import (
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/odo/tests/helper"
)

var _ = Describe("odo storage command tests", func() {

	var oc helper.OcRunner
	var globals helper.Globals

	// This is run after every Spec (It)
	var _ = BeforeEach(func() {
		globals = helper.CommonBeforeEach()

		oc = helper.NewOcRunner("oc")
	})

	// Clean up after the test
	// This is run after every Spec (It)
	var _ = AfterEach(func() {
		helper.CommonAfterEeach(globals)

	})

	Context("when running help for storage command", func() {
		It("should display the help", func() {
			appHelp := helper.CmdShouldPass("odo", "storage", "-h")
			Expect(appHelp).To(ContainSubstring("Perform storage operations"))
		})
	})

	Context("when running storage command without required flag(s)", func() {
		It("should fail", func() {
			helper.CopyExample(filepath.Join("source", "nodejs"), globals.Context)
			helper.CmdShouldPass("odo", "component", "create", "nodejs", "nodejs", "--app", "nodeapp", "--project", globals.Project, "--context", globals.Context)
			stdErr := helper.CmdShouldFail("odo", "storage", "create", "pv1")
			Expect(stdErr).To(ContainSubstring("required flag"))
			//helper.CmdShouldFail("odo", "storage", "create", "pv1", "-o", "json")
		})
	})

	Context("when using storage command with default flag values", func() {
		It("should add a storage, list and delete it", func() {
			helper.CopyExample(filepath.Join("source", "nodejs"), globals.Context)

			helper.CmdShouldPass("odo", "component", "create", "nodejs", "nodejs", "--app", "nodeapp", "--project", globals.Project, "--context", globals.Context)
			// Default flag value
			// --app string         Application, defaults to active application
			// --component string   Component, defaults to active component.
			// --project string     Project, defaults to active project
			storAdd := helper.CmdShouldPass("odo", "storage", "create", "pv1", "--path", "/mnt/pv1", "--size", "1Gi", "--context", globals.Context)
			Expect(storAdd).To(ContainSubstring("nodejs"))
			helper.CmdShouldPass("odo", "push", "--context", globals.Context)

			dcName := oc.GetDcName("nodejs", globals.Project)

			// Check against the volume name against dc
			getDcVolumeMountName := oc.GetVolumeMountName(dcName, globals.Project)
			Expect(getDcVolumeMountName).To(ContainSubstring("pv1"))

			// Check if the storage is added on the path provided
			getMntPath := oc.GetVolumeMountPath(dcName, globals.Project)
			Expect(getMntPath).To(ContainSubstring("/mnt/pv1"))

			storeList := helper.CmdShouldPass("odo", "storage", "list", "--context", globals.Context)
			Expect(storeList).To(ContainSubstring("pv1"))

			// delete the storage
			helper.CmdShouldPass("odo", "storage", "delete", "pv1", "--context", globals.Context, "-f")
			helper.CmdShouldPass("odo", "push", "--context", globals.Context)

			storeList = helper.CmdShouldPass("odo", "storage", "list", "--context", globals.Context)
			Expect(storeList).NotTo(ContainSubstring("pv1"))

			helper.CmdShouldPass("odo", "push", "--context", globals.Context)
			getDcVolumeMountName = oc.GetVolumeMountName(dcName, globals.Project)
			Expect(getDcVolumeMountName).NotTo(ContainSubstring("pv1"))
		})
	})

	Context("when using storage command with specified flag values", func() {
		It("should add a storage, list and delete it", func() {
			helper.CopyExample(filepath.Join("source", "python"), globals.Context)
			helper.CmdShouldPass("odo", "component", "create", "python", "python", "--app", "pyapp", "--project", globals.Project, "--context", globals.Context)
			helper.CmdShouldPass("odo", "push", "--context", globals.Context)
			storAdd := helper.CmdShouldPass("odo", "storage", "create", "pv1", "--path", "/mnt/pv1", "--size", "1Gi", "--context", globals.Context)
			Expect(storAdd).To(ContainSubstring("python"))
			helper.CmdShouldPass("odo", "push", "--context", globals.Context)

			dcName := oc.GetDcName("python", globals.Project)

			// Check against the volume name against dc
			getDcVolumeMountName := oc.GetVolumeMountName(dcName, globals.Project)
			Expect(getDcVolumeMountName).To(ContainSubstring("pv1"))

			// Check if the storage is added on the path provided
			getMntPath := oc.GetVolumeMountPath(dcName, globals.Project)
			Expect(getMntPath).To(ContainSubstring("/mnt/pv1"))

			storeList := helper.CmdShouldPass("odo", "storage", "list", "--context", globals.Context)
			Expect(storeList).To(ContainSubstring("pv1"))

			// delete the storage
			helper.CmdShouldPass("odo", "storage", "delete", "pv1", "--context", globals.Context, "-f")
			helper.CmdShouldPass("odo", "push", "--context", globals.Context)

			storeList = helper.CmdShouldPass("odo", "storage", "list", "--context", globals.Context)

			Expect(storeList).NotTo(ContainSubstring("pv1"))

			helper.CmdShouldPass("odo", "push", "--context", globals.Context)
			getDcVolumeMountName = oc.GetVolumeMountName(dcName, globals.Project)
			Expect(getDcVolumeMountName).NotTo(ContainSubstring("pv1"))
		})
	})

	Context("when using storage command with -o json", func() {
		It("should create and list output in json format", func() {
			helper.CopyExample(filepath.Join("source", "wildfly"), globals.Context)
			helper.CmdShouldPass("odo", "component", "create", "wildfly", "wildfly", "--app", "wildflyapp", "--project", globals.Project, "--context", globals.Context)
			actualJSONStorage := helper.CmdShouldPass("odo", "storage", "create", "mystorage", "--path=/opt/app-root/src/storage/", "--size=1Gi", "--context", globals.Context, "-o", "json")
			desiredJSONStorage := `{"kind":"storage","apiVersion":"odo.dev/v1alpha1","metadata":{"name":"mystorage","creationTimestamp":null},"spec":{"size":"1Gi","path":"/opt/app-root/src/storage/"}}`
			Expect(desiredJSONStorage).Should(MatchJSON(actualJSONStorage))

			actualStorageList := helper.CmdShouldPass("odo", "storage", "list", "--context", globals.Context, "-o", "json")
			desiredStorageList := `{"kind":"List","apiVersion":"odo.dev/v1alpha1","metadata":{},"items":[{"kind":"storage","apiVersion":"odo.dev/v1alpha1","metadata":{"name":"mystorage","creationTimestamp":null},"spec":{"size":"1Gi","path":"/opt/app-root/src/storage/"},"status":"Not Pushed"}]}`
			Expect(desiredStorageList).Should(MatchJSON(actualStorageList))

			helper.CmdShouldPass("odo", "storage", "delete", "mystorage", "--context", globals.Context, "-f")
		})
	})

	Context("when running storage list command to check state", func() {
		It("should list storage with correct state", func() {

			helper.CopyExample(filepath.Join("source", "nodejs"), globals.Context)
			helper.CmdShouldPass("odo", "component", "create", "nodejs", "nodejs", "--app", "nodeapp", "--project", globals.Project, "--context", globals.Context)
			helper.CmdShouldPass("odo", "push", "--context", globals.Context)

			// create storage, list storage should have state "Not Pushed"
			helper.CmdShouldPass("odo", "storage", "create", "pv1", "--path=/tmp1", "--size=1Gi", "--context", globals.Context)
			StorageList := helper.CmdShouldPass("odo", "storage", "list", "--context", globals.Context)
			Expect(StorageList).To(ContainSubstring("Not Pushed"))

			// Push storage, list storage should have state "Pushed"
			helper.CmdShouldPass("odo", "push", "--context", globals.Context)
			StorageList = helper.CmdShouldPass("odo", "storage", "list", "--context", globals.Context)
			Expect(StorageList).To(ContainSubstring("Pushed"))

			// Delete storage, list storage should have state "Locally Deleted"
			helper.CmdShouldPass("odo", "storage", "delete", "pv1", "-f", "--context", globals.Context)
			StorageList = helper.CmdShouldPass("odo", "storage", "list", "--context", globals.Context)
			Expect(StorageList).To(ContainSubstring("Locally Deleted"))

		})
	})

})
