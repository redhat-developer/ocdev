package integration

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/odo/tests/helper"
)

func componentTests(args ...string) {
	const initContainerName = "copy-files-to-volume"
	const wildflyURI1 = "https://github.com/marekjelen/katacoda-odo-backend"
	const wildflyURI2 = "https://github.com/mik-dass/katacoda-odo-backend"
	const appRootVolumeName = "-testing-s2idata"
	var oc helper.OcRunner
	var project string
	var context string
	var originalDir string
	oc = helper.NewOcRunner("oc")

	BeforeEach(func() {
		SetDefaultEventuallyTimeout(10 * time.Minute)
		oc = helper.NewOcRunner("oc")
	})

	Context("odo component creation without application", func() {
		JustBeforeEach(func() {
			project = helper.CreateRandProject()
		})
		JustAfterEach(func() {
			helper.DeleteProject(project)
			os.RemoveAll(".odo")
		})
		It("creating a component without an application should create one", func() {
			componentName := helper.RandString(6)
			helper.CmdShouldPass("odo", append(args, "create", "nodejs", "--project", project, componentName, "--ref", "master", "--git", "https://github.com/openshift/nodejs-ex")...)
			helper.CmdShouldPass("odo", "push")
			appName := helper.CmdShouldPass("odo", "app", "list")
			Expect(appName).ToNot(BeEmpty())

			// checking if application name is set to "app"
			applicationName := helper.GetConfigValue("Application")
			Expect(applicationName).To(Equal("app"))

			// clean up
			helper.CmdShouldPass("odo", "app", "delete", "app", "-f")
			helper.CmdShouldFail("odo", "app", "delete", "app", "-f")
			helper.CmdShouldFail("odo", "component", "delete", componentName, "-f")

		})
	})

	Context("odo component creation", func() {

		JustBeforeEach(func() {
			project = helper.CreateRandProject()
		})
		JustAfterEach(func() {
			helper.DeleteProject(project)
			os.RemoveAll(".odo")
		})

		It("should show an error when ref flag is provided with sources except git", func() {
			outputErr := helper.CmdShouldFail("odo", append(args, "create", "nodejs", "--project", project, "cmp-git", "--ref", "test")...)
			Expect(outputErr).To(ContainSubstring("The --ref flag is only valid for --git flag"))
		})

		It("should create the component from the branch ref when provided", func() {
			helper.CmdShouldPass("odo", append(args, "create", "ruby", "ref-test", "--project", project, "--git", "https://github.com/girishramnani/ruby-ex.git", "--ref", "develop")...)
			helper.CmdShouldPass("odo", "push")
		})
	})

	Context("odo component creation", func() {
		JustBeforeEach(func() {
			project = helper.CreateRandProject()
			context = helper.CreateNewContext()
		})

		JustAfterEach(func() {
			helper.DeleteProject(project)
			os.RemoveAll(context)
		})

		It("should list the component", func() {
			helper.CmdShouldPass("odo", append(args, "create", "nodejs", "cmp-git", "--project", project, "--git", "https://github.com/openshift/nodejs-ex", "--min-memory", "100Mi", "--max-memory", "300Mi", "--min-cpu", "0.1", "--max-cpu", "2", "--context", context, "--app", "testing")...)
			helper.CmdShouldPass("odo", "push", "--context", context)
			originalDir := helper.Getwd()
			helper.Chdir(context)
			cmpList := helper.CmdShouldPass("odo", append(args, "list")...)
			Expect(cmpList).To(ContainSubstring("cmp-git"))
			helper.CmdShouldPass("odo", append(args, "delete", "cmp-git", "-f")...)
			helper.Chdir(originalDir)
		})
	})

	Context("Test odo push with --source and --config flags", func() {
		var originalDir string
		BeforeEach(func() {
			context = helper.CreateNewContext()
		})

		AfterEach(func() {
			helper.DeleteProject(project)
			helper.DeleteDir(context)
		})

		Context("when using project flag(--project) and current directory", func() {
			JustBeforeEach(func() {
				project = helper.CreateRandProject()
				originalDir = helper.Getwd()
				helper.Chdir(context)
			})

			JustAfterEach(func() {
				helper.Chdir(originalDir)
			})

			It("create local nodejs component and push source and code separately", func() {
				appName := "nodejs-push-test"
				cmpName := "nodejs"
				helper.CopyExample(filepath.Join("source", "nodejs"), context)

				helper.CmdShouldPass("odo", append(args, "create", "nodejs", cmpName, "--app", appName, "--project", project)...)

				// component doesn't exist yet so attempt to only push source should fail
				helper.CmdShouldFail("odo", "push", "--source")

				// Push only config and see that the component is created but wothout any source copied
				helper.CmdShouldPass("odo", "push", "--config")
				oc.VerifyCmpExists(cmpName, appName, project)

				// Push only source and see that the component is updated with source code
				helper.CmdShouldPass("odo", "push", "--source")
				oc.VerifyCmpExists(cmpName, appName, project)
				remoteCmdExecPass := oc.CheckCmdOpInRemoteCmpPod(
					cmpName,
					appName,
					project,
					[]string{"ls", "-la", "/tmp/src/package.json"},
					func(cmdOp string, err error) bool {
						if err != nil {
							return false
						}
						return true
					},
				)
				Expect(remoteCmdExecPass).To(Equal(true))
			})

			It("create local nodejs component and push source and code at once", func() {
				appName := "nodejs-push-test"
				cmpName := "nodejs-push-atonce"
				helper.CopyExample(filepath.Join("source", "nodejs"), context)

				helper.CmdShouldPass("odo", append(args, "create", "nodejs", cmpName, "--app", appName, "--project", project)...)

				// Push only config and see that the component is created but wothout any source copied
				helper.CmdShouldPass("odo", "push")
				oc.VerifyCmpExists(cmpName, appName, project)
				remoteCmdExecPass := oc.CheckCmdOpInRemoteCmpPod(
					cmpName,
					appName,
					project,
					[]string{"ls", "-la", "/tmp/src/package.json"},
					func(cmdOp string, err error) bool {
						if err != nil {
							return false
						}
						return true
					},
				)
				Expect(remoteCmdExecPass).To(Equal(true))
			})
		})

		Context("when --context is used", func() {
			// don't need to switch to any dir here, as this test should use --context flag
			JustBeforeEach(func() {
				project = helper.CreateRandProject()
			})

			JustAfterEach(func() {
				os.RemoveAll(".odo")
			})

			It("create local nodejs component and push source and code separately", func() {
				appName := "nodejs-push-context-test"
				cmpName := "nodejs"
				helper.CopyExample(filepath.Join("source", "nodejs"), context)

				helper.CmdShouldPass("odo", append(args, "create", "nodejs", cmpName, "--context", context, "--app", appName, "--project", project)...)
				//TODO: verify that config was properly created

				// component doesn't exist yet so attempt to only push source should fail
				helper.CmdShouldFail("odo", "push", "--source", "--context", context)

				// Push only config and see that the component is created but wothout any source copied
				helper.CmdShouldPass("odo", "push", "--config", "--context", context)
				oc.VerifyCmpExists(cmpName, appName, project)

				// Push only source and see that the component is updated with source code
				helper.CmdShouldPass("odo", "push", "--source", "--context", context)
				oc.VerifyCmpExists(cmpName, appName, project)
				remoteCmdExecPass := oc.CheckCmdOpInRemoteCmpPod(
					cmpName,
					appName,
					project,
					[]string{"ls", "-la", "/tmp/src/package.json"},
					func(cmdOp string, err error) bool {
						if err != nil {
							return false
						}
						return true
					},
				)
				Expect(remoteCmdExecPass).To(Equal(true))
			})

			It("create local nodejs component and push source and code at once", func() {
				appName := "nodejs-push-context-test"
				cmpName := "nodejs-push-atonce"
				helper.CopyExample(filepath.Join("source", "nodejs"), context)

				helper.CmdShouldPass("odo", append(args, "create", "nodejs", cmpName, "--app", appName, "--context", context, "--project", project)...)

				// Push both config and source
				helper.CmdShouldPass("odo", "push", "--context", context)
				oc.VerifyCmpExists(cmpName, appName, project)
				remoteCmdExecPass := oc.CheckCmdOpInRemoteCmpPod(
					cmpName,
					appName,
					project,
					[]string{"ls", "-la", "/tmp/src/package.json"},
					func(cmdOp string, err error) bool {
						if err != nil {
							return false
						}
						return true
					},
				)
				Expect(remoteCmdExecPass).To(Equal(true))
			})
		})
	})

	Context("Creating Component even in new project", func() {
		var project string
		JustBeforeEach(func() {
			context = helper.CreateNewContext()
			project = helper.RandString(10)
		})

		JustAfterEach(func() {
			os.RemoveAll(context)
			helper.DeleteProject(project)
		})
		It("should create component", func() {
			helper.CmdShouldPass("odo", append(args, "create", "nodejs", "cmp-git", "--git", "https://github.com/openshift/nodejs-ex", "--project", project, "--context", context, "--app", "testing")...)
			helper.CmdShouldPass("odo", "push", "--context", context, "-v4")
			oc.SwitchProject(project)
			projectList := helper.CmdShouldPass("odo", "project", "list")
			Expect(projectList).To(ContainSubstring(project))
		})
	})

	Context("when component is in the current directory and --project flag is used", func() {

		JustBeforeEach(func() {
			context = helper.CreateNewContext()
			originalDir = helper.Getwd()
			helper.Chdir(context)
		})

		JustAfterEach(func() {
			helper.Chdir(originalDir)
			os.RemoveAll(context)
		})

		It("create local nodejs component twice and fail", func() {
			helper.CopyExample(filepath.Join("source", "nodejs"), context)
			helper.CmdShouldPass("odo", append(args, "create", "nodejs", "--project", project, "--env", "key=value,key1=value1")...)
			output := helper.CmdShouldFail("odo", append(args, "create", "nodejs", "--project", project, "--env", "key=value,key1=value1")...)
			Expect(output).To(ContainSubstring("this directory already contains a component"))
		})

		It("creates and pushes local nodejs component and then deletes --all", func() {
			helper.CopyExample(filepath.Join("source", "nodejs"), context)
			helper.CmdShouldPass("odo", append(args, "create", "nodejs", "--project", project, "--env", "key=value,key1=value1")...)
			helper.CmdShouldPass("odo", append(args, "push")...)
			helper.CmdShouldPass("odo", append(args, "delete", "--all", "-f")...)

		})
	})

	/*
			Enable once #1782 and #1778 are fixed

				Context("odo component updating", func() {
					JustBeforeEach(func() {
						project = helper.CreateRandProject()
						context = helper.CreateNewContext()
					})

					JustAfterEach(func() {
						helper.DeleteProject(project)
						os.RemoveAll(context)
					})

					It("should be able to create a git component and update it from local to git", func() {
						helper.CopyExample(filepath.Join("source", "nodejs"), context)
						helper.CmdShouldPass("odo", append(args, "create", "nodejs", "cmp-git", "--project", project, "--min-cpu", "0.1", "--max-cpu", "2", "--context", context, "--app", "testing")...)
						helper.CmdShouldPass("odo", "push", "--context", context, "-v", "4")
						getCPULimit := oc.MaxCPU("cmp-git", "testing", project)
						Expect(getCPULimit).To(ContainSubstring("2"))
						getCPURequest := oc.MinCPU("cmp-git", "testing", project)
						Expect(getCPURequest).To(ContainSubstring("100m"))

						// update the component config according to the git component
						helper.CmdShouldPass("odo", "config", "set", "sourcelocation", "https://github.com/openshift/nodejs-ex", "--context", context, "-f")
						helper.CmdShouldPass("odo", "config", "set", "sourcetype", "git", "--context", context, "-f")

						// check if the earlier resource requests are still valid
						helper.CmdShouldPass("odo", "push", "--context", context, "-v", "4")
						getCPULimit = oc.MaxCPU("cmp-git", "testing", project)
						Expect(getCPULimit).To(ContainSubstring("2"))
						getCPURequest = oc.MinCPU("cmp-git", "testing", project)
						Expect(getCPURequest).To(ContainSubstring("100m"))

						// check the source location and type in the deployment config
						getSourceLocation := oc.SourceLocationDC("cmp-git", "testing", project)
						Expect(getSourceLocation).To(ContainSubstring("https://github.com/openshift/nodejs-ex"))
						getSourceType := oc.SourceTypeDC("cmp-git", "testing", project)
						Expect(getSourceType).To(ContainSubstring("git"))

						// since the current component type is git
						// check the source location and type in the build config
						getSourceLocation = oc.SourceLocationBC("cmp-git", "testing", project)
						Expect(getSourceLocation).To(ContainSubstring("https://github.com/openshift/nodejs-ex"))
						getSourceType = oc.SourceTypeBC("cmp-git", "testing", project)
						Expect(getSourceType).To(ContainSubstring("Git"))
					})

					It("should be able to update a component from git to local", func() {
						helper.CmdShouldPass("odo", append(args, "create", "nodejs", "cmp-git", "--project", project, "--git", "https://github.com/openshift/nodejs-ex", "--min-memory", "100Mi", "--max-memory", "300Mi", "--context", context, "--app", "testing")...)
						helper.CmdShouldPass("odo", "push", "--context", context, "-v", "4")
						getMemoryLimit := oc.MaxMemory("cmp-git", "testing", project)
						Expect(getMemoryLimit).To(ContainSubstring("300Mi"))
						getMemoryRequest := oc.MinMemory("cmp-git", "testing", project)
						Expect(getMemoryRequest).To(ContainSubstring("100Mi"))

						// update the component config according to the git component
						helper.CopyExample(filepath.Join("source", "nodejs"), context)
						helper.CmdShouldPass("odo", "config", "set", "sourcelocation", "./", "--context", context, "-f")
						helper.CmdShouldPass("odo", "config", "set", "sourcetype", "local", "--context", context, "-f")

						// check if the earlier resource requests are still valid
						helper.CmdShouldPass("odo", "push", "--context", context, "-v", "4")
						getMemoryLimit = oc.MaxMemory("cmp-git", "testing", project)
						Expect(getMemoryLimit).To(ContainSubstring("300Mi"))
						getMemoryRequest = oc.MinMemory("cmp-git", "testing", project)
						Expect(getMemoryRequest).To(ContainSubstring("100Mi"))

						// check the source location and type in the deployment config
						getSourceLocation := oc.SourceLocationDC("cmp-git", "testing", project)
						Expect(getSourceLocation).To(ContainSubstring("file://./"))
						getSourceType := oc.SourceTypeDC("cmp-git", "testing", project)
						Expect(getSourceType).To(ContainSubstring("local"))
					})
				})
		})
	*/

	Context("odo component delete, list and describe", func() {

		appName := "app"
		cmpName := "nodejs"

		JustBeforeEach(func() {
			SetDefaultEventuallyTimeout(10 * time.Minute)
			project = helper.CreateRandProject()
			context = helper.CreateNewContext()
			originalDir = helper.Getwd()
		})

		JustAfterEach(func() {
			helper.DeleteProject(project)
			helper.DeleteDir(context)
			os.RemoveAll(context)
		})

		It("should pass inside a odo directory without component name as parameter", func() {
			helper.CopyExample(filepath.Join("source", "nodejs"), context)
			helper.CmdShouldPass("odo", "component", "create", "nodejs", cmpName, "--app", appName, "--project", project, "--context", context)
			helper.CmdShouldPass("odo", "push", "--context", context)

			// changing directory to the context directory
			helper.Chdir(context)
			cmpListOutput := helper.CmdShouldPass("odo", "list")
			Expect(cmpListOutput).To(ContainSubstring(cmpName))
			helper.CmdShouldPass("odo", "describe")
			helper.CmdShouldPass("odo", "delete", "-f")
			helper.Chdir(originalDir)
		})

		It("should fail outside a odo directory without component name as parameter", func() {
			helper.CopyExample(filepath.Join("source", "nodejs"), context)
			helper.CmdShouldPass("odo", "component", "create", "nodejs", cmpName, "--app", appName, "--project", project, "--context", context)
			helper.CmdShouldPass("odo", "push", "--context", context)

			// list command should fail as no app flag is given
			helper.CmdShouldFail("odo", "list")
			// commands should fail as the component name is missing
			helper.CmdShouldFail("odo", "describe", "--app", appName)
			helper.CmdShouldFail("odo", "delete", "-f", "--app", appName)
		})

		It("should pass outside a odo directory with component name as parameter", func() {
			helper.CopyExample(filepath.Join("source", "nodejs"), context)
			helper.CmdShouldPass("odo", "component", "create", "nodejs", cmpName, "--app", appName, "--project", project, "--context", context)
			helper.CmdShouldPass("odo", "push", "--context", context)

			cmpListOutput := helper.CmdShouldPass("odo", "list", "--app", appName)
			Expect(cmpListOutput).To(ContainSubstring(cmpName))
			helper.CmdShouldPass("odo", "describe", cmpName, "--app", appName)
			helper.CmdShouldPass("odo", "delete", cmpName, "--app", appName, "-f")
		})

	})

}
