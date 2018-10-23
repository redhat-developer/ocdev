// +build !race

package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"fmt"
	"io/ioutil"
	"strconv"
	"time"
)

// SourceTest checks the component-source-type and the source url in the annotation of the bc and dc
// appTestName is the name of the app
// sourceType is the type of the source of the component i.e git/binary/local
// source is the source of the component i.e gitURL or path to the directory or binary file
func SourceTest(appTestName string, sourceType string, source string) {
	// checking for source-type in dc
	getDc := runCmd("oc get dc wildfly-" + appTestName + " -o go-template='{{index .metadata.annotations \"app.kubernetes.io/component-source-type\"}}'")
	Expect(getDc).To(ContainSubstring(sourceType))

	// checking for source in dc
	getDc = runCmd("oc get dc wildfly-" + appTestName + " -o go-template='{{index .metadata.annotations \"app.kubernetes.io/url\"}}'")
	Expect(getDc).To(ContainSubstring(source))
}

var _ = Describe("odoCmpE2e", func() {
	const bootStrapSupervisorURI = "https://github.com/kadel/bootstrap-supervisored-s2i"
	const initContainerName = "copy-files-to-volume"
	const wildflyUri1 = "https://github.com/marekjelen/katacoda-odo-backend"
	const wildflyUri2 = "https://github.com/mik-dass/katacoda-odo-backend"
	const appRootVolumeName = "-testing-s2idata"

	var t = strconv.FormatInt(time.Now().Unix(), 10)
	var projName = fmt.Sprintf("odocmp-%s", t)
	const appTestName = "testing"

	tmpDir, err := ioutil.TempDir("", "odoCmp")
	if err != nil {
		Fail(err.Error())
	}

	Context("odo component creation", func() {

		It("should create the project and application", func() {
			runCmd("odo project create " + projName)
			runCmd("odo app create " + appTestName)
		})

		It("should be able to create a component with git source", func() {
			runCmd("odo create nodejs cmp-git --git https://github.com/openshift/nodejs-ex")
		})

		It("should list the component", func() {
			cmpList := runCmd("odo list")
			Expect(cmpList).To(ContainSubstring("cmp-git"))
		})

		It("should be in component description", func() {
			cmpDesc := runCmd("odo describe cmp-git")
			Expect(cmpDesc).To(ContainSubstring("source in https://github.com/openshift/nodejs-ex"))
		})

		It("should be in application description", func() {
			appDesc := runCmd("odo describe")
			Expect(appDesc).To(ContainSubstring("source in https://github.com/openshift/nodejs-ex"))
		})

		It("should list the components in the catalog", func() {
			getProj := runCmd("odo catalog list components")
			Expect(getProj).To(ContainSubstring("wildfly"))
			Expect(getProj).To(ContainSubstring("ruby"))
		})
	})

	Context("updating the component", func() {
		It("should be able to create binary component", func() {
			runCmd("wget -O " + tmpDir + "/sample-binary-testing-1.war " +
				"https://gist.github.com/mik-dass/f95bd818ddba508ff76a386f8d984909/raw/e5bc575ac8b14ba2b23d66b5cb4873657e1a1489/sample.war")
			runCmd("odo create wildfly wildfly --binary " + tmpDir + "/sample-binary-testing-1.war")
			runCmd("find " + tmpDir)

			// Run push
			runCmd("odo push")

			cmpList := runCmd("odo list")
			Expect(cmpList).To(ContainSubstring("wildfly"))

			runCmd("oc get dc")
		})

		It("should update component from binary to binary", func() {
			runCmd("wget -O " + tmpDir + "/sample-binary-testing-2.war " +
				"'https://gist.github.com/mik-dass/f95bd818ddba508ff76a386f8d984909/raw/85354d9ee8583a9c1e64a331425eede235b07a9e/sample%2520(1).war'")
			runCmd("odo update wildfly --binary " + tmpDir + "/sample-binary-testing-2.war")

			// checking for init containers
			getDc := runCmd("oc get dc wildfly-" + appTestName + " -o go-template='" +
				"{{range .spec.template.spec.initContainers}}" +
				"{{.name}}{{end}}'")
			Expect(getDc).To(ContainSubstring(initContainerName))

			// checking for volumes
			getDc = runCmd("oc get dc wildfly-" + appTestName + " -o go-template='" +
				"{{range .spec.template.spec.volumes}}" +
				"{{.name}}{{end}}'")
			Expect(getDc).To(ContainSubstring("wildfly" + appRootVolumeName))

			// checking for volumes mounts
			getDc = runCmd("oc get dc wildfly-" + appTestName + " -o go-template='" +
				"{{range .spec.template.spec.containers}}{{range .volumeMounts}}{{.name}}" +
				"{{.name}}{{end}}{{end}}'")
			Expect(getDc).To(ContainSubstring("wildfly" + appRootVolumeName))

			SourceTest(appTestName, "binary", "file://"+tmpDir+"/sample-binary-testing-2.war")
		})

		It("should update component from binary to local", func() {
			runCmd("git clone " + wildflyUri1 + " " +
				tmpDir + "/katacoda-odo-backend-1")

			runCmd("odo update wildfly --local " + tmpDir + "/katacoda-odo-backend-1")

			// checking for init containers
			getDc := runCmd("oc get dc wildfly-" + appTestName + " -o go-template='" +
				"{{range .spec.template.spec.initContainers}}" +
				"{{.name}}{{end}}'")
			Expect(getDc).To(ContainSubstring(initContainerName))

			// checking for volumes
			getDc = runCmd("oc get dc wildfly-" + appTestName + " -o go-template='" +
				"{{range .spec.template.spec.volumes}}" +
				"{{.name}}{{end}}'")
			Expect(getDc).To(ContainSubstring("wildfly" + appRootVolumeName))

			// checking for volumes mounts
			getDc = runCmd("oc get dc wildfly-" + appTestName + " -o go-template='" +
				"{{range .spec.template.spec.containers}}{{range .volumeMounts}}{{.name}}" +
				"{{.name}}{{end}}{{end}}'")
			Expect(getDc).To(ContainSubstring("wildfly" + appRootVolumeName))

			SourceTest(appTestName, "local", "file://"+tmpDir+"/katacoda-odo-backend-1")
		})

		It("should update component from local to local", func() {
			runCmd("git clone " + wildflyUri2 + " " +
				tmpDir + "/katacoda-odo-backend-2")

			runCmd("odo update wildfly --local " + tmpDir + "/katacoda-odo-backend-2")

			// checking for init containers
			getDc := runCmd("oc get dc wildfly-" + appTestName + " -o go-template='" +
				"{{range .spec.template.spec.initContainers}}" +
				"{{.name}}{{end}}'")
			Expect(getDc).To(ContainSubstring(initContainerName))

			// checking for volumes
			getDc = runCmd("oc get dc wildfly-" + appTestName + " -o go-template='" +
				"{{range .spec.template.spec.volumes}}" +
				"{{.name}}{{end}}'")
			Expect(getDc).To(ContainSubstring("wildfly" + appRootVolumeName))

			// checking for volumes mounts
			getDc = runCmd("oc get dc wildfly-" + appTestName + " -o go-template='" +
				"{{range .spec.template.spec.containers}}{{range .volumeMounts}}{{.name}}" +
				"{{.name}}{{end}}{{end}}'")
			Expect(getDc).To(ContainSubstring("wildfly" + appRootVolumeName))

			SourceTest(appTestName, "local", "file://"+tmpDir+"/katacoda-odo-backend-2")
		})

		It("should update component from local to git", func() {
			runCmd("odo update wildfly --git " + wildflyUri1)

			// checking bc for updates
			getBc := runCmd("oc get bc wildfly-" + appTestName + " -o go-template={{.spec.source.git.uri}}")
			Expect(getBc).To(Equal(wildflyUri1))

			// checking for init containers
			getDc := runCmd("oc get dc wildfly-" + appTestName + " -o go-template='" +
				"{{range .spec.template.spec.initContainers}}" +
				"{{.name}}{{end}}'")
			Expect(getDc).NotTo(ContainSubstring(initContainerName))

			// checking for volumes
			getDc = runCmd("oc get dc wildfly-" + appTestName + " -o go-template='" +
				"{{range .spec.template.spec.volumes}}" +
				"{{.name}}{{end}}'")
			Expect(getDc).NotTo(ContainSubstring("wildfly" + appRootVolumeName))

			// checking for volumes mounts
			getDc = runCmd("oc get dc wildfly-" + appTestName + " -o go-template='" +
				"{{range .spec.template.spec.containers}}{{range .volumeMounts}}{{.name}}" +
				"{{.name}}{{end}}{{end}}'")
			Expect(getDc).NotTo(ContainSubstring("wildfly" + appRootVolumeName))

			SourceTest(appTestName, "git", wildflyUri1)
		})

		It("should update component from git to git", func() {
			runCmd("odo update wildfly --git " + wildflyUri2)

			// checking bc for updates
			getBc := runCmd("oc get bc wildfly-" + appTestName + " -o go-template={{.spec.source.git.uri}}")
			Expect(getBc).To(Equal(wildflyUri2))

			// checking for init containers
			getDc := runCmd("oc get dc wildfly-" + appTestName + " -o go-template='" +
				"{{range .spec.template.spec.initContainers}}" +
				"{{.name}}{{end}}'")
			Expect(getDc).NotTo(ContainSubstring(initContainerName))

			// checking for volumes
			getDc = runCmd("oc get dc wildfly-" + appTestName + " -o go-template='" +
				"{{range .spec.template.spec.volumes}}" +
				"{{.name}}{{end}}'")
			Expect(getDc).NotTo(ContainSubstring("wildfly" + appRootVolumeName))

			// checking for volumes mounts
			getDc = runCmd("oc get dc wildfly-" + appTestName + " -o go-template='" +
				"{{range .spec.template.spec.containers}}{{range .volumeMounts}}{{.name}}" +
				"{{.name}}{{end}}{{end}}'")
			Expect(getDc).NotTo(ContainSubstring("wildfly" + appRootVolumeName))

			SourceTest(appTestName, "git", wildflyUri2)
		})

		It("should update component from git to binary", func() {
			runCmd("odo update wildfly --binary " + tmpDir + "/sample-binary-testing-1.war")

			// checking for init containers
			getDc := runCmd("oc get dc wildfly-" + appTestName + " -o go-template='" +
				"{{range .spec.template.spec.initContainers}}" +
				"{{.name}}{{end}}'")
			Expect(getDc).To(ContainSubstring(initContainerName))

			// checking for volumes
			getDc = runCmd("oc get dc wildfly-" + appTestName + " -o go-template='" +
				"{{range .spec.template.spec.volumes}}" +
				"{{.name}}{{end}}'")
			Expect(getDc).To(ContainSubstring("wildfly" + appRootVolumeName))

			// checking for volumes mounts
			getDc = runCmd("oc get dc wildfly-" + appTestName + " -o go-template='" +
				"{{range .spec.template.spec.containers}}{{range .volumeMounts}}{{.name}}" +
				"{{.name}}{{end}}{{end}}'")
			Expect(getDc).To(ContainSubstring("wildfly" + appRootVolumeName))

			SourceTest(appTestName, "binary", "file://"+tmpDir+"/sample-binary-testing-1.war")
		})

		It("should update component from binary to git", func() {
			runCmd("odo update wildfly --git " + wildflyUri1)

			// checking bc for updates
			getBc := runCmd("oc get bc wildfly-" + appTestName + " -o go-template={{.spec.source.git.uri}}")
			Expect(getBc).To(Equal(wildflyUri1))

			// checking for init containers
			getDc := runCmd("oc get dc wildfly-" + appTestName + " -o go-template='" +
				"{{range .spec.template.spec.initContainers}}" +
				"{{.name}}{{end}}'")
			Expect(getDc).NotTo(ContainSubstring(initContainerName))

			// checking for volumes
			getDc = runCmd("oc get dc wildfly-" + appTestName + " -o go-template='" +
				"{{range .spec.template.spec.volumes}}" +
				"{{.name}}{{end}}'")
			Expect(getDc).NotTo(ContainSubstring("wildfly" + appRootVolumeName))

			// checking for volumes mounts
			getDc = runCmd("oc get dc wildfly-" + appTestName + " -o go-template='" +
				"{{range .spec.template.spec.containers}}{{range .volumeMounts}}{{.name}}" +
				"{{.name}}{{end}}{{end}}'")
			Expect(getDc).NotTo(ContainSubstring("wildfly" + appRootVolumeName))

			SourceTest(appTestName, "git", wildflyUri1)
		})

		It("should update component from git to local", func() {
			runCmd("odo update wildfly --local " + tmpDir + "/katacoda-odo-backend-1")

			// checking for init containers
			getDc := runCmd("oc get dc wildfly-" + appTestName + " -o go-template='" +
				"{{range .spec.template.spec.initContainers}}" +
				"{{.name}}{{end}}'")
			Expect(getDc).To(ContainSubstring(initContainerName))

			// checking for volumes
			getDc = runCmd("oc get dc wildfly-" + appTestName + " -o go-template='" +
				"{{range .spec.template.spec.volumes}}" +
				"{{.name}}{{end}}'")
			Expect(getDc).To(ContainSubstring("wildfly" + appRootVolumeName))

			// checking for volumes mounts
			getDc = runCmd("oc get dc wildfly-" + appTestName + " -o go-template='" +
				"{{range .spec.template.spec.containers}}{{range .volumeMounts}}{{.name}}" +
				"{{.name}}{{end}}{{end}}'")
			Expect(getDc).To(ContainSubstring("wildfly" + appRootVolumeName))

			SourceTest(appTestName, "local", "file://"+tmpDir+"/katacoda-odo-backend-1")
		})

		It("should update component from local to binary", func() {
			runCmd("odo update wildfly --binary " + tmpDir + "/sample-binary-testing-1.war")

			// checking for init containers
			getDc := runCmd("oc get dc wildfly-" + appTestName + " -o go-template='" +
				"{{range .spec.template.spec.initContainers}}" +
				"{{.name}}{{end}}'")
			Expect(getDc).To(ContainSubstring(initContainerName))

			// checking for volumes
			getDc = runCmd("oc get dc wildfly-" + appTestName + " -o go-template='" +
				"{{range .spec.template.spec.volumes}}" +
				"{{.name}}{{end}}'")
			Expect(getDc).To(ContainSubstring("wildfly" + appRootVolumeName))

			// checking for volumes mounts
			getDc = runCmd("oc get dc wildfly-" + appTestName + " -o go-template='" +
				"{{range .spec.template.spec.containers}}{{range .volumeMounts}}{{.name}}" +
				"{{.name}}{{end}}{{end}}'")
			Expect(getDc).To(ContainSubstring("wildfly" + appRootVolumeName))

			SourceTest(appTestName, "binary", "file://"+tmpDir+"/sample-binary-testing-1.war")
		})
	})

	Context("cleaning up", func() {
		It("should delete the application", func() {
			runCmd("odo app delete " + appTestName + " -f")

			runCmd("odo project delete " + projName + " -f")
			waitForDeleteCmd("odo project list", projName)
		})
	})
})
