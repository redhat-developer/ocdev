package devfile

import (
	. "github.com/onsi/ginkgo"
	"github.com/openshift/odo/tests/helper"
)

var _ = Describe("odo devfile config command tests", func() {

	var commonVar helper.CommonVar

	// This is run before every Spec (It)
	var _ = BeforeEach(func() {
		commonVar = helper.CommonBeforeEach()
		helper.Chdir(commonVar.Context)
	})

	// This is run after every Spec (It)
	var _ = AfterEach(func() {
		helper.CommonAfterEach(commonVar)
	})

	It("should fail to set and unset an invalid parameter", func() {
		const fakeParameter = "fakeParameter"
		helper.Cmd("odo", "config", "set", fakeParameter, fakeParameter, "-f").ShouldFail()
		helper.Cmd("odo", "config", "unset", fakeParameter, "-f").ShouldFail()
	})

	When("a component is created", func() {
		BeforeEach(func() {
			helper.Cmd("odo", "create", "nodejs").ShouldPass()
		})

		When("executing config view", func() {
			var output string

			BeforeEach(func() {
				output = helper.Cmd("odo", "config", "view").ShouldPass().Out()
			})

			It("should view all default parameters", func() {
				wantOutput := []string{
					"nodejs",
					"Ports",
					"Memory",
				}
				helper.MatchAllInOutput(output, wantOutput)
			})
		})

		When("executing config set", func() {

			const (
				testName      = "testname"
				testMemory    = "500Mi"
				testDebugPort = "8888"
			)

			BeforeEach(func() {
				helper.Cmd("odo", "config", "set", "Name", testName, "-f").ShouldPass()
				helper.Cmd("odo", "config", "set", "Ports", testDebugPort, "-f").ShouldPass()
				helper.Cmd("odo", "config", "set", "Memory", testMemory, "-f").ShouldPass()
			})

			It("should successfully set the parameters", func() {
				output := helper.Cmd("odo", "config", "view").ShouldPass().Out()
				wantOutput := []string{
					testName,
					testMemory,
					testDebugPort,
				}
				helper.MatchAllInOutput(output, wantOutput)
			})

			When("unsettting a parameter", func() {

				BeforeEach(func() {
					helper.Cmd("odo", "config", "unset", "Ports", "-f").ShouldPass()
				})

				It("should successfully unset the parameter", func() {
					output := helper.Cmd("odo", "config", "view").ShouldPass().Out()
					dontWantOutput := []string{
						testDebugPort,
					}
					helper.DontMatchAllInOutput(output, dontWantOutput)
					helper.Cmd("odo", "push", "--project", commonVar.Project).ShouldPass()
				})
			})
		})
	})
})
