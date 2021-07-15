package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/odo/tests/helper"
)

var _ = Describe("odo link command tests for OperatorHub", func() {

	var commonVar helper.CommonVar

	BeforeEach(func() {
		commonVar = helper.CommonBeforeEach()
	})

	AfterEach(func() {
		helper.CommonAfterEach(commonVar)
	})

	Context("Operators are installed in the cluster", func() {

		var redisOperator string
		var redisCluster string

		BeforeEach(func() {
			// wait till odo can see that all operators installed by setup script in the namespace
			odoArgs := []string{"catalog", "list", "services"}
			operators := []string{"redis-operator"}
			for _, operator := range operators {
				helper.WaitForCmdOut("odo", odoArgs, 5, true, func(output string) bool {
					return strings.Contains(output, operator)
				})
			}

			commonVar.CliRunner.CreateSecret("redis-secret", "password", commonVar.Project)
			list := helper.Cmd("odo", "catalog", "list", "services").ShouldPass().Out()
			redisOperator = regexp.MustCompile(`redis-operator\.*[a-z][0-9]\.[0-9]\.[0-9]`).FindString(list)
			redisCluster = fmt.Sprintf("%s/Redis", redisOperator)
		})

		When("a component and a service are deployed", func() {

			var componentName string
			var svcFullName string

			BeforeEach(func() {
				helper.CopyExample(filepath.Join("source", "nodejs"), commonVar.Context)
				componentName = "cmp-" + helper.RandString(6)
				helper.Cmd("odo", "create", "nodejs", componentName, "--context", commonVar.Context, "--project", commonVar.Project).ShouldPass()
				helper.Cmd("odo", "config", "set", "Memory", "300M", "-f", "--context", commonVar.Context).ShouldPass()

				serviceName := "service-" + helper.RandString(6)
				svcFullName = strings.Join([]string{"Redis", serviceName}, "/")
				helper.Cmd("odo", "service", "create", redisCluster, serviceName, "--context", commonVar.Context).ShouldPass()

				helper.Cmd("odo", "push", "--context", commonVar.Context).ShouldPass()
				name := commonVar.CliRunner.GetRunningPodNameByComponent(componentName, commonVar.Project)
				Expect(name).To(Not(BeEmpty()))
			})

			It("should find files in component container", func() {
				helper.Cmd("odo", "exec", "--context", commonVar.Context, "--", "ls", "/project/server.js").ShouldPass()
			})

			When("a link between the component and the service is created", func() {

				BeforeEach(func() {
					helper.Cmd("odo", "link", svcFullName, "--context", commonVar.Context).ShouldPass()
				})

				It("should find the link in odo describe", func() {
					stdOut := helper.Cmd("odo", "describe", "--context", commonVar.Context).ShouldPass().Out()
					Expect(stdOut).To(ContainSubstring(svcFullName))
				})

				When("odo push is executed", func() {
					BeforeEach(func() {
						helper.Cmd("odo", "push", "--context", commonVar.Context).ShouldPass()
						name := commonVar.CliRunner.GetRunningPodNameByComponent(componentName, commonVar.Project)
						Expect(name).To(Not(BeEmpty()))
					})

					It("should find files in component container", func() {
						helper.Cmd("odo", "exec", "--context", commonVar.Context, "--", "ls", "/project/server.js").ShouldPass()
					})

					It("should find the link environment variable", func() {
						stdOut := helper.Cmd("odo", "exec", "--context", commonVar.Context, "--", "sh", "-c", "echo $REDIS_CLUSTERIP").ShouldPass().Out()
						Expect(stdOut).To(Not(BeEmpty()))
					})

					It("should find the link in odo describe", func() {
						stdOut := helper.Cmd("odo", "describe", "--context", commonVar.Context).ShouldPass().Out()
						Expect(stdOut).To(ContainSubstring(svcFullName))
						Expect(stdOut).To(ContainSubstring("Environment Variables"))
						Expect(stdOut).To(ContainSubstring("REDIS_CLUSTERIP"))
					})
				})
			})

			When("a link with between the component and the service is created with --bind-as-files", func() {

				var bindingName string
				BeforeEach(func() {
					bindingName = "sbr-" + helper.RandString(6)
					helper.Cmd("odo", "link", svcFullName, "--bind-as-files", "--name", bindingName, "--context", commonVar.Context).ShouldPass()
				})

				It("should dislay the link in odo describe", func() {
					stdOut := helper.Cmd("odo", "describe", "--context", commonVar.Context).ShouldPass().Out()
					Expect(stdOut).To(ContainSubstring(svcFullName))
				})

				When("odo push is executed", func() {
					BeforeEach(func() {
						helper.Cmd("odo", "push", "--context", commonVar.Context).ShouldPass()
						name := commonVar.CliRunner.GetRunningPodNameByComponent(componentName, commonVar.Project)
						Expect(name).To(Not(BeEmpty()))
					})

					It("should find files in component container", func() {
						helper.Cmd("odo", "exec", "--context", commonVar.Context, "--", "ls", "/project/server.js").ShouldPass()
					})

					It("should find bindings for service", func() {
						helper.Cmd("odo", "exec", "--context", commonVar.Context, "--", "ls", "/bindings/"+bindingName+"/clusterIP").ShouldPass()
					})

					It("should display the link in odo describe", func() {
						stdOut := helper.Cmd("odo", "describe", "--context", commonVar.Context).ShouldPass().Out()
						Expect(stdOut).To(ContainSubstring(svcFullName))
						Expect(stdOut).To(ContainSubstring("Files"))
						Expect(stdOut).To(ContainSubstring("/bindings/" + bindingName + "/clusterIP"))
					})
				})
			})
		})

		When("getting sources, a devfile defining a component, a service and a link, and executing odo push", func() {

			BeforeEach(func() {
				componentName := "api" // this is the name of the component in the devfile
				helper.CopyExample(filepath.Join("source", "nodejs"), commonVar.Context)
				helper.CopyExampleDevFile(filepath.Join("source", "devfiles", "nodejs", "devfile-with-link.yaml"), filepath.Join(commonVar.Context, "devfile.yaml"))
				helper.Cmd("odo", "create", componentName, "--project", commonVar.Project, "--context", commonVar.Context).ShouldPass()

				helper.Cmd("odo", "push", "--context", commonVar.Context).ShouldPass()
				name := commonVar.CliRunner.GetRunningPodNameByComponent(componentName, commonVar.Project)
				Expect(name).To(Not(BeEmpty()))
			})

			It("should find files in component container", func() {
				helper.Cmd("odo", "exec", "--context", commonVar.Context, "--", "ls", "/project/server.js").ShouldPass()
			})

			It("should find bindings for service", func() {
				helper.Cmd("odo", "exec", "--context", commonVar.Context, "--", "ls", "/bindings/redis-link/clusterIP").ShouldPass()
			})

			It("should find owner references on link and service", func() {
				if os.Getenv("KUBERNETES") == "true" {
					Skip("This is a OpenShift specific scenario, skipping")
				}
				args := []string{"get", "servicebinding", "redis-link", "-o", "jsonpath='{.metadata.ownerReferences.*.name}'", "-n", commonVar.Project}
				commonVar.CliRunner.WaitForRunnerCmdOut(args, 1, true, func(output string) bool {
					return strings.Contains(output, "api-app")
				})

				args = []string{"get", "redis.redis.redis.opstreelabs.in", "myredis", "-o", "jsonpath='{.metadata.ownerReferences.*.name}'", "-n", commonVar.Project}
				commonVar.CliRunner.WaitForRunnerCmdOut(args, 1, true, func(output string) bool {
					return strings.Contains(output, "api-app")
				})
			})
		})
	})

	When("one component is deployed", func() {
		var context0 string
		var cmp0 string

		BeforeEach(func() {
			context0 = helper.CreateNewContext()
			cmp0 = helper.RandString(5)

			helper.Cmd("odo", "create", "nodejs", cmp0, "--context", context0).ShouldPass()

			helper.CopyExample(filepath.Join("source", "devfiles", "nodejs", "project"), context0)
			helper.CopyExampleDevFile(filepath.Join("source", "devfiles", "nodejs", "devfile.yaml"), filepath.Join(context0, "devfile.yaml"))

			helper.Cmd("odo", "push", "--context", context0).ShouldPass()
		})

		AfterEach(func() {
			helper.Cmd("odo", "delete", "-f", "--context", context0).ShouldPass()
			helper.DeleteDir(context0)
		})

		It("should fail when linking to itself", func() {
			stdOut := helper.Cmd("odo", "link", cmp0, "--context", context0).ShouldFail().Err()
			helper.MatchAllInOutput(stdOut, []string{cmp0, "cannot be linked with itself"})
		})

		It("should fail if the component doesn't exist and the service name doesn't adhere to the <service-type>/<service-name> format", func() {
			helper.Cmd("odo", "link", "Redis").ShouldFail()
			helper.Cmd("odo", "link", "Redis/").ShouldFail()
			helper.Cmd("odo", "link", "/redis-standalone").ShouldFail()
		})

		When("another component is deployed", func() {
			var context1 string
			var cmp1 string

			BeforeEach(func() {
				context1 = helper.CreateNewContext()
				cmp1 = helper.RandString(5)

				helper.Cmd("odo", "create", "nodejs", cmp1, "--context", context1).ShouldPass()

				helper.CopyExample(filepath.Join("source", "devfiles", "nodejs", "project"), context1)
				helper.CopyExampleDevFile(filepath.Join("source", "devfiles", "nodejs", "devfileNestedCompCommands.yaml"), filepath.Join(context1, "devfile.yaml"))

				helper.Cmd("odo", "push", "--context", context1).ShouldPass()
			})

			AfterEach(func() {
				helper.Cmd("odo", "delete", "-f", "--context", context1).ShouldPass()
				helper.DeleteDir(context1)
			})

			It("should link the two components successfully with service binding operator", func() {

				if os.Getenv("KUBERNETES") == "true" {
					// service binding operator is not installed on kubernetes
					Skip("This is a OpenShift specific scenario, skipping")
				}

				helper.Cmd("odo", "link", cmp1, "--context", context0).ShouldPass()
				helper.Cmd("odo", "push", "--context", context0).ShouldPass()

				// check the link exists with the specific name
				ocArgs := []string{"get", "servicebinding", strings.Join([]string{cmp0, cmp1}, "-"), "-o", "jsonpath='{.status.secret}'", "-n", commonVar.Project}
				helper.WaitForCmdOut("oc", ocArgs, 1, true, func(output string) bool {
					return strings.Contains(output, strings.Join([]string{cmp0, cmp1}, "-"))
				})

				// delete the link and undeploy it
				helper.Cmd("odo", "unlink", cmp1, "--context", context0).ShouldPass()
				helper.Cmd("odo", "push", "--context", context0).ShouldPass()
				commonVar.CliRunner.WaitAndCheckForTerminatingState("servicebinding", commonVar.Project, 1)
			})

			It("should link the two components successfully without service binding operator", func() {

				if os.Getenv("KUBERNETES") != "true" {
					// service binding operator is not installed on kubernetes
					Skip("This is a Kubernetes specific scenario, skipping")
				}

				helper.Cmd("odo", "link", cmp1, "--context", context0).ShouldPass()
				helper.Cmd("odo", "push", "--context", context0).ShouldPass()

				// check the secrets exists with the specific name
				secrets := commonVar.CliRunner.GetSecrets(commonVar.Project)
				Expect(secrets).To(ContainSubstring(fmt.Sprintf("%v-%v", cmp0, cmp1)))

				envFromValues := commonVar.CliRunner.GetEnvRefNames(cmp0, "app", commonVar.Project)
				envFound := false
				for i := range envFromValues {
					if strings.Contains(envFromValues[i], fmt.Sprintf("%v-%v", cmp0, cmp1)) {
						envFound = true
					}
				}
				Expect(envFound).To(BeTrue())

				// delete the link and undeploy it
				helper.Cmd("odo", "unlink", cmp1, "--context", context0).ShouldPass()
				helper.Cmd("odo", "push", "--context", context0).ShouldPass()

				// check the secrets exists with the specific name
				secrets = commonVar.CliRunner.GetSecrets(commonVar.Project)
				Expect(secrets).NotTo(ContainSubstring(fmt.Sprintf("%v-%v", cmp0, cmp1)))
				envFromValues = commonVar.CliRunner.GetEnvRefNames(cmp0, "app", commonVar.Project)
				envFound = false
				for i := range envFromValues {
					if strings.Contains(envFromValues[i], fmt.Sprintf("%v-%v", cmp0, cmp1)) {
						envFound = true
					}
				}
				Expect(envFound).To(BeFalse())
			})
		})
	})
})
