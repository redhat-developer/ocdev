package integration

import (
	"fmt"
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
		helper.Chdir(commonVar.Context)
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
			operators := []string{"redis-operator", "service-binding-operator"}
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
				helper.Cmd("odo", "create", "nodejs", componentName).ShouldPass()
				helper.Cmd("odo", "config", "set", "Memory", "300M", "-f").ShouldPass()

				serviceName := "service-" + helper.RandString(6)
				svcFullName = strings.Join([]string{"Redis", serviceName}, "/")
				helper.Cmd("odo", "service", "create", redisCluster, serviceName, "--project", commonVar.Project).ShouldPass()

				helper.Cmd("odo", "push").ShouldPass()
				name := commonVar.CliRunner.GetRunningPodNameByComponent(componentName, commonVar.Project)
				Expect(name).To(Not(BeEmpty()))
			})

			It("should find files in component container", func() {
				helper.Cmd("odo", "exec", "--", "ls", "/project/server.js").ShouldPass()
			})

			When("a link between the component and the service is created", func() {

				BeforeEach(func() {
					helper.Cmd("odo", "link", svcFullName).ShouldPass()
				})

				It("should find the link in odo describe", func() {
					stdOut := helper.Cmd("odo", "describe").ShouldPass().Out()
					Expect(stdOut).To(ContainSubstring(svcFullName))
				})

				When("odo push is executed", func() {
					BeforeEach(func() {
						helper.Cmd("odo", "push").ShouldPass()
						name := commonVar.CliRunner.GetRunningPodNameByComponent(componentName, commonVar.Project)
						Expect(name).To(Not(BeEmpty()))
					})

					It("should find files in component container", func() {
						helper.Cmd("odo", "exec", "--", "ls", "/project/server.js").ShouldPass()
					})

					It("should find the link environment variable", func() {
						stdOut := helper.Cmd("odo", "exec", "--", "sh", "-c", "echo $REDIS_CLUSTERIP").ShouldPass().Out()
						Expect(stdOut).To(Not(BeEmpty()))
					})

					It("should find the link in odo describe", func() {
						stdOut := helper.Cmd("odo", "describe").ShouldPass().Out()
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
					helper.Cmd("odo", "link", svcFullName, "--bind-as-files", "--name", bindingName).ShouldPass()
				})

				It("should display the link in odo describe", func() {
					stdOut := helper.Cmd("odo", "describe").ShouldPass().Out()
					Expect(stdOut).To(ContainSubstring(svcFullName))
				})

				When("odo push is executed", func() {
					BeforeEach(func() {
						helper.Cmd("odo", "push").ShouldPass()
						name := commonVar.CliRunner.GetRunningPodNameByComponent(componentName, commonVar.Project)
						Expect(name).To(Not(BeEmpty()))
					})

					It("should find files in component container", func() {
						helper.Cmd("odo", "exec", "--", "ls", "/project/server.js").ShouldPass()
					})

					It("should find bindings for service", func() {
						helper.Cmd("odo", "exec", "--", "ls", "/bindings/"+bindingName+"/clusterIP").ShouldPass()
					})

					It("should display the link in odo describe", func() {
						stdOut := helper.Cmd("odo", "describe").ShouldPass().Out()
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
				helper.Cmd("odo", "create", componentName).ShouldPass()

				helper.Cmd("odo", "push").ShouldPass()
				name := commonVar.CliRunner.GetRunningPodNameByComponent(componentName, commonVar.Project)
				Expect(name).To(Not(BeEmpty()))
			})

			It("should find files in component container", func() {
				helper.Cmd("odo", "exec", "--", "ls", "/project/server.js").ShouldPass()
			})

			It("should find bindings for service", func() {
				helper.Cmd("odo", "exec", "--", "ls", "/bindings/redis-link/clusterIP").ShouldPass()
			})

			It("should find owner references on link and service", func() {
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
})
