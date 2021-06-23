package integration

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/odo/pkg/util"
	"github.com/openshift/odo/tests/helper"
	"github.com/tidwall/gjson"
)

var _ = Describe("odo service command tests for OperatorHub", func() {

	var commonVar helper.CommonVar
	var oc helper.OcRunner

	BeforeEach(func() {
		commonVar = helper.CommonBeforeEach()
		helper.Chdir(commonVar.Context)
		oc = helper.NewOcRunner("oc")
	})

	AfterEach(func() {
		helper.CommonAfterEach(commonVar)
	})

	Context("Operators are installed in the cluster", func() {

		JustBeforeEach(func() {
			// wait till odo can see that all operators installed by setup script in the namespace
			odoArgs := []string{"catalog", "list", "services"}
			operators := []string{"etcdoperator", "service-binding-operator"}
			for _, operator := range operators {
				helper.WaitForCmdOut("odo", odoArgs, 5, true, func(output string) bool {
					return strings.Contains(output, operator)
				})
			}
		})

		JustAfterEach(func() {
			helper.DeleteProject(commonVar.Project)
		})

		It("should not allow creating service without valid context", func() {
			stdOut := helper.Cmd("odo", "service", "create").ShouldFail().Err()
			Expect(stdOut).To(ContainSubstring("service can be created/deleted from a valid component directory only"))
		})

		Context("a namespace specific operator is installed", func() {

			var postgresOperator string
			var postgresDatabase string
			var projectName string

			JustBeforeEach(func() {
				projectName = util.GetEnvWithDefault("REDHAT_POSTGRES_OPERATOR_PROJECT", "odo-operator-test")
				helper.GetCliRunner().SetProject(projectName)
				operators := helper.Cmd("odo", "catalog", "list", "services").ShouldPass().Out()
				postgresOperator = regexp.MustCompile(`postgresql-operator\.*[a-z][0-9]\.[0-9]\.[0-9]`).FindString(operators)
				postgresDatabase = fmt.Sprintf("%s/Database", postgresOperator)
			})

			When("a nodejs component is created", func() {

				JustBeforeEach(func() {
					helper.Cmd("odo", "create", "nodejs").ShouldPass().Out()
				})

				JustAfterEach(func() {
					// we do this because for these specific tests we dont delete the project
					helper.Cmd("odo", "delete", "--all", "-f").ShouldPass().Out()
				})

				When("creating a postgres operand with params", func() {
					var operandName string

					JustBeforeEach(func() {
						operandName = helper.RandString(10)
						helper.Cmd("odo", "service", "create", postgresDatabase, operandName, "-p",
							"databaseName=odo", "-p", "size=1", "-p", "databaseUser=odo", "-p",
							"databaseStorageRequest=1Gi", "-p", "databasePassword=odopasswd").ShouldPass().Out()

					})

					JustAfterEach(func() {
						helper.Cmd("odo", "service", "delete", fmt.Sprintf("Database/%s", operandName), "-f").ShouldPass().Out()
						helper.Cmd("odo", "push").ShouldPass().Out()
					})

					When("odo push is executed", func() {
						JustBeforeEach(func() {
							helper.Cmd("odo", "push").ShouldPass().Out()
						})

						It("should create pods in running state", func() {
							oc.PodsShouldBeRunning(projectName, fmt.Sprintf(`%s-.[\-a-z0-9]*`, operandName))
						})

						It("should list the service", func() {
							// now test listing of the service using odo
							stdOut := helper.Cmd("odo", "service", "list").ShouldPass().Out()
							Expect(stdOut).To(ContainSubstring(fmt.Sprintf("Database/%s", operandName)))
						})
					})

				})

			})
		})

		Context("a specific operator is installed", func() {
			var etcdOperator string
			var etcdCluster string

			JustBeforeEach(func() {
				operators := helper.Cmd("odo", "catalog", "list", "services").ShouldPass().Out()
				etcdOperator = regexp.MustCompile(`etcdoperator\.*[a-z][0-9]\.[0-9]\.[0-9]-clusterwide`).FindString(operators)
				etcdCluster = fmt.Sprintf("%s/EtcdCluster", etcdOperator)
			})

			It("should describe the operator with human-readable output", func() {
				output := helper.Cmd("odo", "catalog", "describe", "service", etcdCluster).ShouldPass().Out()
				Expect(output).To(ContainSubstring("Kind: EtcdCluster"))
			})

			It("should describe the example of the operator", func() {
				output := helper.Cmd("odo", "catalog", "describe", "service", etcdCluster, "--example").ShouldPass().Out()
				Expect(output).To(ContainSubstring("kind: EtcdCluster"))
				helper.MatchAllInOutput(output, []string{"apiVersion", "kind"})
			})

			It("should describe the example of the operator as json", func() {
				outputJSON := helper.Cmd("odo", "catalog", "describe", "service", etcdCluster, "--example", "-o", "json").ShouldPass().Out()
				value := gjson.Get(outputJSON, "spec.kind")
				Expect(value.String()).To(Equal("EtcdCluster"))
			})

			It("should describe the operator with json output", func() {
				outputJSON := helper.Cmd("odo", "catalog", "describe", "service", etcdCluster, "-o", "json").ShouldPass().Out()
				values := gjson.GetMany(outputJSON, "spec.kind", "spec.displayName")
				expected := []string{"EtcdCluster", "etcd Cluster"}
				Expect(helper.GjsonMatcher(values, expected)).To(Equal(true))
			})

			It("should find the services by keyword", func() {
				stdOut := helper.Cmd("odo", "catalog", "search", "service", "etcd").ShouldPass().Out()
				helper.MatchAllInOutput(stdOut, []string{"etcdoperator", "EtcdCluster"})

				stdOut = helper.Cmd("odo", "catalog", "search", "service", "EtcdCluster").ShouldPass().Out()
				helper.MatchAllInOutput(stdOut, []string{"etcdoperator", "EtcdCluster"})

				stdOut = helper.Cmd("odo", "catalog", "search", "service", "dummy").ShouldFail().Err()
				Expect(stdOut).To(ContainSubstring("no service matched the query: dummy"))
			})

			It("should list the operator in JSON output", func() {
				jsonOut := helper.Cmd("odo", "catalog", "list", "services", "-o", "json").ShouldPass().Out()
				helper.MatchAllInOutput(jsonOut, []string{"etcdoperator"})
			})

			When("a nodejs component is created", func() {

				JustBeforeEach(func() {
					helper.Cmd("odo", "create", "nodejs").ShouldPass()
				})

				It("should fail for interactive mode", func() {
					stdOut := helper.Cmd("odo", "service", "create").ShouldFail().Err()
					Expect(stdOut).To(ContainSubstring("odo doesn't support interactive mode for creating Operator backed service"))
				})

				When("odo push is executed", func() {
					JustBeforeEach(func() {
						helper.Cmd("odo", "push").ShouldPass()
					})

					It("should fail if the provided service doesn't exist in the namespace", func() {
						if os.Getenv("KUBERNETES") == "true" {
							Skip("This is a OpenShift specific scenario, skipping")
						}
						stdOut := helper.Cmd("odo", "link", "EtcdCluster/example").ShouldFail().Err()
						Expect(stdOut).To(ContainSubstring("couldn't find service named %q", "EtcdCluster/example"))
					})
				})

				When("an EtcdCluster instance is created in dryRun mode", func() {

					var stdOut string

					JustBeforeEach(func() {
						stdOut = helper.Cmd("odo", "service", "create", fmt.Sprintf("%s/EtcdCluster", etcdOperator), "--dry-run", "--project", commonVar.Project).ShouldPass().Out()
					})

					It("should only output the definition of the CR that will be used to start service", func() {
						helper.MatchAllInOutput(stdOut, []string{"apiVersion", "kind"})
					})

					When("the output of the command is stored in a file", func() {

						var fileName string

						JustBeforeEach(func() {
							randomFileName := helper.RandString(6) + ".yaml"
							fileName = filepath.Join(os.TempDir(), randomFileName)
							if err := ioutil.WriteFile(fileName, []byte(stdOut), 0644); err != nil {
								fmt.Printf("Could not write yaml spec to file %s because of the error %v", fileName, err.Error())
							}
						})

						JustAfterEach(func() {
							os.Remove(fileName)
						})

						When("a service is created from the output of the dryRun command with no name", func() {
							JustBeforeEach(func() {
								helper.Cmd("odo", "service", "create", "--from-file", fileName, "--project", commonVar.Project).ShouldPass()
							})

							JustAfterEach(func() {
								helper.Cmd("odo", "service", "delete", "EtcdCluster/example", "-f").ShouldPass()
							})

							When("odo push is executed", func() {
								JustBeforeEach(func() {
									helper.Cmd("odo", "push").ShouldPass()
								})

								It("should create pods in running state", func() {
									oc.PodsShouldBeRunning(commonVar.Project, `example-.[a-z0-9]*`)
								})
							})
						})

						When("a service is created from the output of the dryRun command with a specific name", func() {

							var name string
							var svcFullName string
							JustBeforeEach(func() {
								name = helper.RandString(6)
								svcFullName = strings.Join([]string{"EtcdCluster", name}, "/")
								helper.Cmd("odo", "service", "create", "--from-file", fileName, name, "--project", commonVar.Project).ShouldPass()
							})

							JustAfterEach(func() {
								helper.Cmd("odo", "service", "delete", svcFullName, "-f").ShouldPass()
							})

							When("odo push is executed", func() {

								JustBeforeEach(func() {
									helper.Cmd("odo", "push").ShouldPass()
								})

								It("should fail to create a service again with the same name", func() {
									stdOut = helper.Cmd("odo", "service", "create", "--from-file", fileName, name, "--project", commonVar.Project).ShouldFail().Err()
									Expect(stdOut).To(ContainSubstring("please provide a different name or delete the existing service first"))
								})

								It("should create pods in running state", func() {
									oc.PodsShouldBeRunning(commonVar.Project, name+`-.[a-z0-9]*`)
								})
							})
						})
					})
				})

				When("an EtcdCluster instance is created with no name", func() {
					var stdOut string
					JustBeforeEach(func() {
						stdOut = helper.Cmd("odo", "service", "create", fmt.Sprintf("%s/EtcdCluster", etcdOperator), "--project", commonVar.Project).ShouldPass().Out()
						Expect(stdOut).To(ContainSubstring("Successfully added service to the configuration"))
					})

					It("should insert service definition in devfile.yaml", func() {
						devfilePath := filepath.Join(commonVar.Context, "devfile.yaml")
						content, err := ioutil.ReadFile(devfilePath)
						Expect(err).To(BeNil())
						matchInOutput := []string{"kubernetes", "inlined", "EtcdCluster", "etcdcluster"}
						helper.MatchAllInOutput(string(content), matchInOutput)
					})

					When("odo push is executed", func() {

						JustBeforeEach(func() {
							helper.Cmd("odo", "push").ShouldPass()
						})

						It("should create pods in running state", func() {
							oc.PodsShouldBeRunning(commonVar.Project, `etcdcluster-.[a-z0-9]*`)
						})

						It("should list the service", func() {
							// now test listing of the service using odo
							stdOut := helper.Cmd("odo", "service", "list").ShouldPass().Out()
							Expect(stdOut).To(ContainSubstring("EtcdCluster/etcdcluster"))
						})

						It("should list the service in JSON format", func() {
							jsonOut := helper.Cmd("odo", "service", "list", "-o", "json").ShouldPass().Out()
							helper.MatchAllInOutput(jsonOut, []string{"\"apiVersion\": \"etcd.database.coreos.com/v1beta2\"", "\"kind\": \"EtcdCluster\"", "\"name\": \"etcdcluster\""})
						})

						When("a link is created with the service", func() {
							var stdOut string
							JustBeforeEach(func() {
								stdOut = helper.Cmd("odo", "link", "EtcdCluster/etcdcluster").ShouldPass().Out()
							})

							It("should display a successful message", func() {
								if os.Getenv("KUBERNETES") == "true" {
									Skip("This is a OpenShift specific scenario, skipping")
								}
								Expect(stdOut).To(ContainSubstring("Successfully created link between component"))
							})

							It("Should fail to link it again", func() {
								if os.Getenv("KUBERNETES") == "true" {
									Skip("This is a OpenShift specific scenario, skipping")
								}
								stdOut = helper.Cmd("odo", "link", "EtcdCluster/etcdcluster").ShouldFail().Err()
								Expect(stdOut).To(ContainSubstring("already linked with the service"))
							})

							When("the link is deleted", func() {
								JustBeforeEach(func() {
									stdOut = helper.Cmd("odo", "unlink", "EtcdCluster/etcdcluster").ShouldPass().Out()
								})

								It("should display a successful message", func() {
									if os.Getenv("KUBERNETES") == "true" {
										Skip("This is a OpenShift specific scenario, skipping")
									}
									Expect(stdOut).To(ContainSubstring("Successfully unlinked component"))
								})

								It("should fail to delete it again", func() {
									if os.Getenv("KUBERNETES") == "true" {
										Skip("This is a OpenShift specific scenario, skipping")
									}
									stdOut = helper.Cmd("odo", "unlink", "EtcdCluster/etcdcluster").ShouldFail().Err()
									Expect(stdOut).To(ContainSubstring("failed to unlink the service"))
								})
							})
						})

						When("the service is deleted", func() {
							JustBeforeEach(func() {
								helper.Cmd("odo", "service", "delete", "EtcdCluster/etcdcluster", "-f").ShouldPass()
							})

							It("should delete service definition from devfile.yaml", func() {
								// read the devfile.yaml to check if service definition was deleted
								devfilePath := filepath.Join(commonVar.Context, "devfile.yaml")
								content, err := ioutil.ReadFile(devfilePath)
								Expect(err).To(BeNil())
								matchInOutput := []string{"kubernetes", "inlined", "EtcdCluster", "etcdcluster"}
								helper.DontMatchAllInOutput(string(content), matchInOutput)
							})

							It("should fail to delete the service again", func() {
								stdOut = helper.Cmd("odo", "service", "delete", "EtcdCluster/etcdcluster", "-f").ShouldFail().Err()
								Expect(stdOut).To(ContainSubstring("couldn't find service named"))
							})

							When("odo push is executed", func() {
								JustBeforeEach(func() {
									helper.Cmd("odo", "push").ShouldPass()
								})

								It("Should fail listing the services", func() {
									out := helper.Cmd("odo", "service", "list").ShouldFail().Err()
									msg := fmt.Sprintf("no operator backed services found in namespace: %s", commonVar.Project)
									Expect(out).To(ContainSubstring(msg))
								})

								It("Should fail listing the services in JSON format", func() {
									jsonOut := helper.Cmd("odo", "service", "list", "-o", "json").ShouldFail().Err()
									msg := fmt.Sprintf("no operator backed services found in namespace: %s", commonVar.Project)
									msgWithQuote := fmt.Sprintf("\"message\": \"no operator backed services found in namespace: %s\"", commonVar.Project)
									helper.MatchAllInOutput(jsonOut, []string{msg, msgWithQuote})
								})
							})
						})

						When("a second service is created", func() {
							JustBeforeEach(func() {
								stdOut = helper.Cmd("odo", "service", "create", fmt.Sprintf("%s/EtcdCluster", etcdOperator), "myetcd2", "--project", commonVar.Project).ShouldPass().Out()
								Expect(stdOut).To(ContainSubstring("Successfully added service to the configuration"))
							})

							When("odo push is executed", func() {
								JustBeforeEach(func() {
									helper.Cmd("odo", "push").ShouldPass()
								})

								It("should list both services", func() {
									stdOut = helper.Cmd("odo", "service", "list").ShouldPass().Out()
									// first service still here
									Expect(stdOut).To(ContainSubstring("EtcdCluster/etcdcluster"))
									// second service created
									Expect(stdOut).To(ContainSubstring("EtcdCluster/myetcd2"))
								})
							})
						})
					})
				})

				When("an EtcdCluster instance is created with a specific name", func() {

					var name string
					var svcFullName string

					JustBeforeEach(func() {
						name = helper.RandString(6)
						svcFullName = strings.Join([]string{"EtcdCluster", name}, "/")
						helper.Cmd("odo", "service", "create", fmt.Sprintf("%s/EtcdCluster", etcdOperator), name, "--project", commonVar.Project).ShouldPass()
					})

					JustAfterEach(func() {
						helper.Cmd("odo", "service", "delete", svcFullName, "-f").ShouldRun()
					})

					It("should be listed as Not pushed", func() {
						stdOut := helper.Cmd("odo", "service", "list").ShouldPass().Out()
						helper.MatchAllInOutput(stdOut, []string{svcFullName, "Not pushed"})
					})

					When("odo push is executed", func() {

						JustBeforeEach(func() {
							helper.Cmd("odo", "push").ShouldPass()
						})

						It("should create pods in running state", func() {
							oc.PodsShouldBeRunning(commonVar.Project, name+`-.[a-z0-9]*`)
						})

						It("should fail to create a service again with the same name", func() {
							stdOut := helper.Cmd("odo", "service", "create", fmt.Sprintf("%s/EtcdCluster", etcdOperator), name, "--project", commonVar.Project).ShouldFail().Err()
							Expect(stdOut).To(ContainSubstring(fmt.Sprintf("service %q already exists", svcFullName)))
						})

						It("should be listed as Pushed", func() {
							stdOut := helper.Cmd("odo", "service", "list").ShouldPass().Out()
							helper.MatchAllInOutput(stdOut, []string{svcFullName, "Pushed"})
						})

						When("the etcdCluster instance is deleted", func() {
							JustBeforeEach(func() {
								helper.Cmd("odo", "service", "delete", svcFullName, "-f").ShouldPass()
							})

							It("should be listed as Deleted locally", func() {
								stdOut := helper.Cmd("odo", "service", "list").ShouldPass().Out()
								helper.MatchAllInOutput(stdOut, []string{svcFullName, "Deleted locally"})
							})

							When("odo push is executed", func() {
								JustBeforeEach(func() {
									helper.Cmd("odo", "push").ShouldPass()
								})

								It("should not be listed anymore", func() {
									stdOut := helper.Cmd("odo", "service", "list").ShouldRun().Out()
									Expect(strings.Contains(stdOut, svcFullName)).To(BeFalse())
								})
							})
						})

						When("a link is created with a specific name", func() {

							var linkName string

							JustBeforeEach(func() {
								linkName = "link-" + helper.RandString(6)
								helper.Cmd("odo", "link", "EtcdCluster/"+name, "--name", linkName).ShouldPass()
								// for the moment, odo push is not necessary to deploy the link
							})

							JustAfterEach(func() {
								// delete the link
								helper.Cmd("odo", "unlink", "EtcdCluster/"+name).ShouldPass()
							})

							It("should create the link with the specified name", func() {
								ocArgs := []string{"get", "servicebinding", linkName, "-n", commonVar.Project}
								helper.WaitForCmdOut("oc", ocArgs, 1, true, func(output string) bool {
									return strings.Contains(output, linkName)
								})
							})
						})

						When("a link is created with a specific name and bind-as-files flag", func() {

							var linkName string

							JustBeforeEach(func() {
								linkName = "link-" + helper.RandString(6)
								helper.Cmd("odo", "link", "EtcdCluster/"+name, "--name", linkName, "--bind-as-files").ShouldPass()
								// for the moment, odo push is not necessary to deploy the link
							})

							JustAfterEach(func() {
								// delete the link
								helper.Cmd("odo", "unlink", "EtcdCluster/"+name).ShouldPass()
							})

							It("should create a servicebinding resource with bindAsFiles set to true", func() {
								ocArgs := []string{"get", "servicebinding", linkName, "-o", "jsonpath='{.spec.bindAsFiles}'", "-n", commonVar.Project}
								helper.WaitForCmdOut("oc", ocArgs, 1, true, func(output string) bool {
									return strings.Contains(output, "true")
								})
							})
						})
					})
				})

				Context("Invalid service templates exist", func() {

					var tmpContext string
					var noMetaFileName string
					var invalidFileName string

					JustBeforeEach(func() {
						tmpContext = helper.CreateNewContext()

						// TODO write helpers to create such files
						noMetadata := `
apiVersion: etcd.database.coreos.com/v1beta2
kind: EtcdCluster
spec:
  size: 3
  version: 3.2.13`
						noMetaFile := helper.RandString(6) + ".yaml"
						noMetaFileName = filepath.Join(tmpContext, noMetaFile)
						if err := ioutil.WriteFile(noMetaFileName, []byte(noMetadata), 0644); err != nil {
							fmt.Printf("Could not write yaml spec to file %s because of the error %v", noMetaFileName, err.Error())
						}

						invalidMetadata := `
apiVersion: etcd.database.coreos.com/v1beta2
kind: EtcdCluster
metadata:
  noname: noname
spec:
  size: 3
  version: 3.2.13`
						invalidMetaFile := helper.RandString(6) + ".yaml"
						invalidFileName = filepath.Join(tmpContext, invalidMetaFile)
						if err := ioutil.WriteFile(invalidFileName, []byte(invalidMetadata), 0644); err != nil {
							fmt.Printf("Could not write yaml spec to file %s because of the error %v", invalidFileName, err.Error())
						}

					})

					JustAfterEach(func() {
						helper.DeleteDir(tmpContext)
					})

					It("should fail to create a service based on a template without metadata", func() {
						stdOut := helper.Cmd("odo", "service", "create", "--from-file", noMetaFileName, "--project", commonVar.Project).ShouldFail().Err()
						Expect(stdOut).To(ContainSubstring("couldn't find \"metadata\" in the yaml"))
					})

					It("should fail to create a service based on a template with invalid metadata", func() {
						stdOut := helper.Cmd("odo", "service", "create", "--from-file", invalidFileName, "--project", commonVar.Project).ShouldFail().Err()
						Expect(stdOut).To(ContainSubstring("couldn't find metadata.name in the yaml"))
					})
				})
			})
		})

		When("one component is deployed", func() {
			var context0 string
			var cmp0 string

			JustBeforeEach(func() {
				if os.Getenv("KUBERNETES") == "true" {
					Skip("This is a OpenShift specific scenario, skipping")
				}

				context0 = helper.CreateNewContext()
				cmp0 = helper.RandString(5)

				helper.Cmd("odo", "create", "nodejs", cmp0, "--context", context0).ShouldPass()
				helper.Cmd("odo", "push", "--context", context0).ShouldPass()
			})

			JustAfterEach(func() {
				helper.Cmd("odo", "delete", "-f", "--context", context0).ShouldPass()
				helper.DeleteDir(context0)
			})

			It("should fail when linking to itself", func() {
				stdOut := helper.Cmd("odo", "link", cmp0, "--context", context0).ShouldFail().Err()
				helper.MatchAllInOutput(stdOut, []string{cmp0, "cannot be linked with itself"})
			})

			It("should fail if the component doesn't exist and the service name doesn't adhere to the <service-type>/<service-name> format", func() {
				if os.Getenv("KUBERNETES") == "true" {
					Skip("This is a OpenShift specific scenario, skipping")
				}
				helper.Cmd("odo", "link", "EtcdCluster").ShouldFail()
				helper.Cmd("odo", "link", "EtcdCluster/").ShouldFail()
				helper.Cmd("odo", "link", "/example").ShouldFail()
			})

			When("another component is deployed", func() {
				var context1 string
				var cmp1 string

				JustBeforeEach(func() {
					context1 = helper.CreateNewContext()
					cmp1 = helper.RandString(5)

					helper.Cmd("odo", "create", "nodejs", cmp1, "--context", context1).ShouldPass()
					helper.Cmd("odo", "push", "--context", context1).ShouldPass()
				})

				JustAfterEach(func() {
					helper.Cmd("odo", "delete", "-f", "--context", context1).ShouldPass()
					helper.DeleteDir(context1)
				})

				It("should link the two components successfully", func() {

					helper.Cmd("odo", "link", cmp1, "--context", context0).ShouldPass()

					// check the link exists with the specific name
					ocArgs := []string{"get", "servicebinding", strings.Join([]string{cmp0, cmp1}, "-"), "-o", "jsonpath='{.status.secret}'", "-n", commonVar.Project}
					helper.WaitForCmdOut("oc", ocArgs, 1, true, func(output string) bool {
						return strings.Contains(output, strings.Join([]string{cmp0, cmp1}, "-"))
					})

					// delete the link
					helper.Cmd("odo", "unlink", cmp1, "--context", context0).ShouldPass()

					commonVar.CliRunner.WaitAndCheckForTerminatingState("servicebinding", commonVar.Project, 1)
				})
			})
		})
	})
})
