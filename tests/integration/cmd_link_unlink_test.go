package integration

import (
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openshift/odo/tests/helper"
)

var _ = Describe("odo link and unlink command tests", func() {

	var commonVar helper.CommonVar

	var _ = BeforeEach(func() {
		commonVar = helper.CommonBeforeEach()
	})

	var _ = AfterEach(func() {
		helper.CommonAfterEach(commonVar)
	})

	Context("Running the help command", func() {
		It("should display the help", func() {
			By("for link command", func() {
				appHelp := helper.Cmd("odo", "link", "-h").ShouldPass().Out()
				helper.MatchAllInOutput(appHelp, []string{"Link component to a service ", "backed by an Operator or Service Catalog", "or component", "works only with s2i components"})
			})
			By("for unlink command", func() {
				appHelp := helper.Cmd("odo", "unlink", "-h").ShouldPass().Out()
				Expect(appHelp).To(ContainSubstring("Unlink component or service from a component"))
			})
		})
	})

	When("Two components are deployed", func() {
		var frontendContext, backendContext, frontendURL string
		var oc helper.OcRunner

		JustBeforeEach(func() {
			oc = helper.NewOcRunner("oc")
			frontendContext = helper.CreateNewContext()
			backendContext = helper.CreateNewContext()
			helper.CopyExample(filepath.Join("source", "nodejs"), frontendContext)
			helper.Cmd("odo", "create", "nodejs", "frontend", "--context", frontendContext, "--project", commonVar.Project).ShouldPass()
			helper.Cmd("odo", "url", "create", "--port", "8080", "--context", frontendContext).ShouldPass()
			helper.Cmd("odo", "push", "--context", frontendContext).ShouldPass()
			frontendURL = helper.DetermineRouteURL(frontendContext)
			helper.CopyExample(filepath.Join("source", "python"), backendContext)
			helper.Cmd("odo", "create", "python", "backend", "--project", commonVar.Project, "--context", backendContext).ShouldPass()
			helper.Cmd("odo", "url", "create", "--port", "8080", "--context", backendContext).ShouldPass()
			helper.Cmd("odo", "push", "--context", backendContext).ShouldPass()
		})

		JustAfterEach(func() {
			helper.DeleteDir(frontendContext)
			helper.DeleteDir(backendContext)
		})

		When("a link is created between the two components", func() {
			JustBeforeEach(func() {
				// we link
				helper.Chdir(frontendContext)
				helper.Cmd("odo", "link", "backend").ShouldPass()
			})

			JustAfterEach(func() {
				helper.Chdir(commonVar.Context)
			})

			It("should find the link in odo describe", func() {
				stdOut := helper.Cmd("odo", "describe").ShouldPass().Out()
				Expect(stdOut).To(ContainSubstring("Linked Services"))
				Expect(stdOut).To(ContainSubstring("backend"))
			})

			When("the link is pushed", func() {
				JustBeforeEach(func() {
					helper.Cmd("odo", "push").ShouldPass()
				})

				JustAfterEach(func() {})

				It("should successfully link", func() {
					By("ensuring that the proper envFrom entry was created", func() {
						envFromOutput := oc.GetEnvFromEntry("frontend", "app", commonVar.Project, "deployment")
						Expect(envFromOutput).To(ContainSubstring("backend"))
						helper.HttpWaitFor(frontendURL, "Hello world from node.js!", 20, 1)
					})
					By("finding the link in odo describe", func() {
						stdOut := helper.Cmd("odo", "describe").ShouldPass().Out()
						Expect(stdOut).To(ContainSubstring("Linked Services"))
						Expect(stdOut).To(ContainSubstring("backend"))
						Expect(stdOut).To(ContainSubstring("SERVICE_BACKEND_IP"))
						Expect(stdOut).To(ContainSubstring("SERVICE_BACKEND_PORT"))
					})
					By("finding the linked environment variable", func() {
						stdOut := helper.Cmd("odo", "exec", "--", "sh", "-c", "echo $SERVICE_BACKEND_IP").ShouldPass().Out()
						Expect(stdOut).To(Not(BeEmpty()))
					})
					By("not allowing re-linking", func() {
						outputErr := helper.Cmd("odo", "link", "backend").ShouldFail().Err()
						Expect(outputErr).To(ContainSubstring("already linked"))
					})
				})

				It("should successfully delete component after linked component is deleted", func() {
					// Testing: https://github.com/openshift/odo/issues/2355
					helper.Cmd("odo", "delete", "-f", "--context", backendContext).ShouldPass()
					helper.Cmd("odo", "delete", "-f").ShouldPass()
				})

				When("unlinking the two components", func() {
					JustBeforeEach(func() {
						helper.Cmd("odo", "unlink", "backend", "--context", frontendContext).ShouldPass()
					})
					It("should find the link in odo describe", func() {
						stdOut := helper.Cmd("odo", "describe").ShouldPass().Out()
						Expect(stdOut).To(ContainSubstring("Linked Services"))
						Expect(stdOut).To(ContainSubstring("backend"))
						Expect(stdOut).To(ContainSubstring("SERVICE_BACKEND_IP"))
						Expect(stdOut).To(ContainSubstring("SERVICE_BACKEND_PORT"))
					})
					It("should not allow unlinking again", func() {
						stdOut := helper.Cmd("odo", "unlink", "backend", "--context", frontendContext).ShouldFail().Err()
						Expect(stdOut).To(ContainSubstring("failed to unlink the component \"backend\" since no link was found in the configuration referring this component"))
					})

					When("odo push is executed", func() {
						JustBeforeEach(func() {
							helper.Cmd("odo", "push", "--context", frontendContext).ShouldPass()
						})
						JustAfterEach(func() {})
						It("should no longer find the link in odo describe", func() {
							stdOut := helper.Cmd("odo", "describe").ShouldPass().Out()
							Expect(stdOut).ToNot(ContainSubstring("Linked Services"))
							Expect(stdOut).ToNot(ContainSubstring("backend"))
						})
					})
				})
			})
		})
		When("a link is created between the two components with --bind-as-files", func() {
			JustBeforeEach(func() {
				helper.Chdir(frontendContext)
				helper.Cmd("odo", "link", "backend", "--bind-as-files").ShouldPass()
			})
			JustAfterEach(func() {
				helper.Chdir(commonVar.Context)
			})
			When("the component is pushed", func() {
				JustBeforeEach(func() {
					helper.Cmd("odo", "push").ShouldPass()
				})

				JustAfterEach(func() {})

				It("should successfully link", func() {
					By("ensuring that the proper envFrom entry was created", func() {
						envFromOutput := oc.GetEnvFromEntry("frontend", "app", commonVar.Project, "deployment")
						Expect(envFromOutput).To(ContainSubstring("backend"))
						helper.HttpWaitFor(frontendURL, "Hello world from node.js!", 20, 1)
					})

					By("finding the link in odo describe", func() {
						stdOut := helper.Cmd("odo", "describe").ShouldPass().Out()
						Expect(stdOut).To(ContainSubstring("Linked Services"))
						Expect(stdOut).To(ContainSubstring("backend"))
						Expect(stdOut).To(ContainSubstring("/bindings"))
					})

					By("finding the linked environment variable", func() {
						stdOut := helper.Cmd("odo", "exec", "--", "ls", "/bindings").ShouldPass().Out()
						Expect(stdOut).To(Not(ContainSubstring("backend")))
					})

					By("not allowing re-linking", func() {
						outputErr := helper.Cmd("odo", "link", "backend").ShouldFail().Err()
						Expect(outputErr).To(ContainSubstring("already linked"))
					})
				})
				When("unlinking the two components", func() {
					JustBeforeEach(func() {
						helper.Cmd("odo", "unlink", "backend", "--context", frontendContext).ShouldPass()
					})
					It("should find the link in odo describe", func() {
						stdOut := helper.Cmd("odo", "describe").ShouldPass().Out()
						Expect(stdOut).To(ContainSubstring("Linked Services"))
						Expect(stdOut).To(ContainSubstring("backend"))
						Expect(stdOut).To(ContainSubstring("/bindings"))
					})

					It("should not allow unlinking again", func() {
						stdOut := helper.Cmd("odo", "unlink", "backend", "--context", frontendContext).ShouldFail().Err()
						Expect(stdOut).To(ContainSubstring("failed to unlink the component \"backend\" since no link was found in the configuration referring this component"))
					})

					When("odo push is executed", func() {
						JustBeforeEach(func() {
							helper.Cmd("odo", "push", "--context", frontendContext).ShouldPass()
						})
						JustAfterEach(func() {})
						It("should no longer find the link in odo describe", func() {
							stdOut := helper.Cmd("odo", "describe").ShouldPass().Out()
							Expect(stdOut).ToNot(ContainSubstring("Linked Services"))
							Expect(stdOut).ToNot(ContainSubstring("backend"))
						})
					})
				})

			})

		})
	})
})
