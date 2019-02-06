package e2e

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("odoWatchE2e", func() {

	const appTestName = "testing"
	const wildflyURI = "https://github.com/marekjelen/katacoda-odo-backend"
	const pythonURI = "https://github.com/OpenShiftDemos/os-sample-python"
	const nodejsURI = "https://github.com/openshift/nodejs-ex"
	const openjdkURI = "https://github.com/geoand/javalin-helloworld"

	projName := generateTimeBasedName("odowatch")

	tmpDir, err := ioutil.TempDir("", "odoCmp")
	if err != nil {
		Fail(err.Error())
	}

	Context("watch component created from local source or binary", func() {
		It("should create the project and application", func() {
			runCmd("odo project create " + projName)
			runCmd("odo app create " + appTestName)
			importOpenJDKImage()
		})

		It("watch nodejs component created from local source", func() {
			runCmd("git clone " + nodejsURI + " " + tmpDir + "/nodejs-ex")
			runCmd("odo create nodejs nodejs-watch --local " + tmpDir + "/nodejs-ex --min-memory 400Mi --max-memory 700Mi")
			runCmd("odo push -v 4")
			// Test multiple push so as to avoid regressions like: https://github.com/redhat-developer/odo/issues/1054
			runCmd("odo push -v 4")
			runCmd("odo url create --port 8080")

			startSimulationCh := make(chan bool)
			go func() {
				startMsg := <-startSimulationCh
				if startMsg {
					fileModificationPath := filepath.Join(
						tmpDir,
						"nodejs-ex",
						"server.js",
					)

					fmt.Printf("Triggering filemodification @; %s\n", fileModificationPath)
					fileModificationCmd := fmt.Sprintf(
						"sed -i \"s/res.send('{ pageCount: -1 }')/res.send('{ pageCount: -1, message: Hello odo }')/g\" %s",
						fileModificationPath,
					)
					fmt.Println("Received signal, starting file modification simulation")
					runCmd("mkdir -p " + tmpDir + "/nodejs-ex" + "/'.a b'")
					runCmd("mkdir -p " + tmpDir + "/nodejs-ex" + "/'a b'")
					runCmd("touch " + tmpDir + "/nodejs-ex" + "/'a b.txt'")

					runCmd(fmt.Sprintf("mkdir -p %s/nodejs-ex/tests/sample-tests", tmpDir))
					runCmd(fmt.Sprintf("touch %s/nodejs-ex/tests/sample-tests/test_1.js", tmpDir))

					// Delete during watch
					runCmd(fmt.Sprintf("rm -fr %s/nodejs-ex/tests/sample-tests", tmpDir))
					runCmd("rm -fr " + tmpDir + "/nodejs-ex/'a b.txt'")

					runCmd(fileModificationCmd)
				}
			}()
			success, err := pollNonRetCmdStdOutForString(
				"odo watch nodejs-watch -v 4",
				time.Duration(20)*time.Minute,
				func(output string) bool {
					url := runCmd("odo url list | grep `odo component get -q` | grep 8080 | awk '{print $2}' | tr -d '\n'")
					url = fmt.Sprintf("%s/pagecount", url)
					curlOp := runCmd(fmt.Sprintf("curl %s", url))
					if strings.Contains(curlOp, "Hello odo") {
						// Verify delete from component pod
						podName := runCmd("oc get pods | grep nodejs-watch | awk '{print $1}' | tr -d '\n'")
						runFailCmd("oc exec "+podName+" -c nodejs-watch-testing -- ls -lai /tmp/src/tests/sample-tests/test_1.js /opt/app-root/src-backup/src/tests/sample-tests;exit", 2)
						runCmd("oc exec " + podName + " -c nodejs-watch-testing -- ls -lai /tmp/src/ | grep 'a b';exit")
						runFailCmd("oc exec "+podName+" -c nodejs-watch-testing -- ls -lai /tmp/src/ | grep 'a b.txt';exit", 1)
					}
					return strings.Contains(curlOp, "Hello odo")
				},
				startSimulationCh,
				func(output string) bool {
					return strings.Contains(output, "Waiting for something to change")
				})
			Expect(success).To(Equal(true))
			Expect(err).To(BeNil())

			// Verify memory limits to be same as configured
			getMemoryLimit := runCmd("oc get dc nodejs-watch-" +
				appTestName +
				" -o go-template='{{range .spec.template.spec.containers}}{{.resources.limits.memory}}{{end}}'",
			)
			Expect(getMemoryLimit).To(ContainSubstring("700Mi"))
			getMemoryRequest := runCmd("oc get dc nodejs-watch-" +
				appTestName +
				" -o go-template='{{range .spec.template.spec.containers}}{{.resources.requests.memory}}{{end}}'",
			)
			Expect(getMemoryRequest).To(ContainSubstring("400Mi"))
		})

		It("should watch python component local sources for any changes", func() {
			runCmd("git clone " + pythonURI + " " + tmpDir + "/os-sample-python")
			runCmd("odo create python python-watch --local " + tmpDir + "/os-sample-python --memory 400Mi")
			runCmd("odo push -v 4")
			// Test multiple push so as to avoid regressions like: https://github.com/redhat-developer/odo/issues/1054
			runCmd("odo push -v 4")
			runCmd("odo url create")

			startSimulationCh := make(chan bool)
			go func() {
				startMsg := <-startSimulationCh
				if startMsg {
					fmt.Println("Received signal, starting file modification simulation")
					fileModificationCmd := fmt.Sprintf("sed -i 's/World/odo/g' %s", filepath.Join(tmpDir, "os-sample-python", "wsgi.py"))

					runCmd(fmt.Sprintf("mkdir -p %s/os-sample-python/tests", tmpDir))
					runCmd(fmt.Sprintf("touch %s/os-sample-python/tests/test_1.py", tmpDir))
					runCmd("mkdir -p " + tmpDir + "/os-sample-python" + "/'.a b'")
					runCmd("mkdir -p " + tmpDir + "/os-sample-python" + "/'a b'")
					runCmd("touch " + tmpDir + "/os-sample-python" + "/'a b.txt'")

					// Delete during watch
					runCmd(fmt.Sprintf("rm -fr %s/os-sample-python/tests", tmpDir))
					runCmd("rm -fr " + tmpDir + "/os-sample-python/'a b.txt'")
					runCmd(fileModificationCmd)
				}
			}()
			success, err := pollNonRetCmdStdOutForString(
				"odo watch python-watch -v 4",
				time.Duration(5)*time.Minute,
				func(output string) bool {
					url := runCmd("odo url list | grep `odo component get -q` | grep 8080 | awk '{print $2}'")
					curlOp := runCmd(fmt.Sprintf("curl %s", url))

					if strings.Contains(curlOp, "Hello odo") {
						podName := runCmd("oc get pods | grep python-watch | awk '{print $1}' | tr -d '\n'")
						runCmd("oc exec " + podName + " -c python-watch-testing -- ls -lai /tmp/src/ | grep 'a b';exit")

						// Verify delete from component pod
						runFailCmd("oc exec "+podName+" -c python-watch-testing -- ls -lai /tmp/src/tests;exit", 2)
						runFailCmd("oc exec "+podName+" -c python-watch-testing -- ls -lai /opt/app-root/src-backup/src/tests;exit", 2)
						runFailCmd("oc exec "+podName+" -c python-watch-testing -- ls -lai /tmp/src/ | grep 'a b.txt';exit", 1)
					}

					return strings.Contains(curlOp, "Hello odo")
				},
				startSimulationCh,
				func(output string) bool {
					return strings.Contains(output, "Waiting for something to change")
				})
			Expect(success).To(Equal(true))
			Expect(err).To(BeNil())

			// Verify memory limits to be same as configured
			getMemoryLimit := runCmd("oc get dc python-watch-" +
				appTestName +
				" -o go-template='{{range .spec.template.spec.containers}}{{.resources.limits.memory}}{{end}}'",
			)
			Expect(getMemoryLimit).To(ContainSubstring("400Mi"))
			getMemoryRequest := runCmd("oc get dc python-watch-" +
				appTestName +
				" -o go-template='{{range .spec.template.spec.containers}}{{.resources.requests.memory}}{{end}}'",
			)
			Expect(getMemoryRequest).To(ContainSubstring("400Mi"))
		})
		It("watch wildfly component created from local source", func() {
			runCmd("git clone " + wildflyURI + " " + tmpDir + "/katacoda-odo-backend")
			runCmd("odo create wildfly wildfly-watch --local " + tmpDir + "/katacoda-odo-backend --min-memory 400Mi --max-memory 700Mi")
			runCmd("odo push -v 4")
			// Test multiple push so as to avoid regressions like: https://github.com/redhat-developer/odo/issues/1054
			runCmd("odo push -v 4")
			runCmd("odo url create --port 8080")

			startSimulationCh := make(chan bool)
			go func() {
				startMsg := <-startSimulationCh
				if startMsg {
					fmt.Println("Received signal, starting file modification simulation")

					runCmd(fmt.Sprintf("mkdir -p %s/katacoda-odo-backend/tests", tmpDir))
					runCmd(fmt.Sprintf("touch %s/katacoda-odo-backend/tests/test_1.java", tmpDir))
					runCmd("mkdir -p " + tmpDir + "/katacoda-odo-backend/src" + "/'.a b'")
					runCmd("mkdir -p " + tmpDir + "/katacoda-odo-backend/src" + "/'a b'")
					runCmd("touch " + tmpDir + "/katacoda-odo-backend/src" + "/'a b.txt'")
					// Delete during watch
					runCmd(fmt.Sprintf("rm -fr %s/katacoda-odo-backend/tests", tmpDir))
					runCmd("rm -fr " + tmpDir + "/katacoda-odo-backend/src/'a b.txt'")

					fileModificationPath := filepath.Join(
						tmpDir,
						"katacoda-odo-backend",
						"src",
						"main",
						"java",
						"eu",
						"mjelen",
						"katacoda",
						"odo",
						"BackendServlet.java",
					)
					fmt.Printf("Triggering filemodification @; %s\n", fileModificationPath)
					fileModificationCmd := fmt.Sprintf(
						"sed -i 's/response.getWriter().println(counter.toString())/response.getWriter().println(\"Hello odo\" + counter.toString())/g' %s",
						fileModificationPath,
					)
					runCmd(fileModificationCmd)
				}
			}()
			success, err := pollNonRetCmdStdOutForString(
				"odo watch wildfly-watch -v 4",
				time.Duration(5)*time.Minute,
				func(output string) bool {
					url := runCmd("odo url list | grep `odo component get -q` | grep 8080 | awk '{print $2}' | tr -d '\n'")
					url = fmt.Sprintf("%s/counter", url)
					curlOp := runCmd(fmt.Sprintf("curl %s", url))
					if strings.Contains(curlOp, "Hello odo") {
						// Verify delete from component pod
						podName := runCmd("oc get pods | grep wildfly-watch | awk '{print $1}' | tr -d '\n'")
						runFailCmd("oc exec "+podName+" -c wildfly-watch-testing -- ls -lai /opt/s2i/destination/src/tests /opt/app-root/src-backup/src/tests; exit", 2)
						runCmd("oc exec " + podName + " -c wildfly-watch-testing -- ls -lai /opt/s2i/destination/src/src/ | grep 'a b';exit")
						runFailCmd("oc exec "+podName+" -c wildfly-watch-testing -- ls -lai /opt/s2i/destination/src/src/tests;exit", 2)
						runFailCmd("oc exec "+podName+" -c wildfly-watch-testing -- ls -lai /opt/app-root/src-backup/src/tests;exit", 2)
						runFailCmd("oc exec "+podName+" -c wildfly-watch-testing -- ls -lai /opt/s2i/destination/src/src/ | grep 'a b.txt';exit", 1)
					}
					return strings.Contains(curlOp, "Hello odo")
				},
				startSimulationCh,
				func(output string) bool {
					return strings.Contains(output, "Waiting for something to change")
				})
			Expect(success).To(Equal(true))
			Expect(err).To(BeNil())

			// Verify memory limits to be same as configured
			getMemoryLimit := runCmd("oc get dc wildfly-watch-" +
				appTestName +
				" -o go-template='{{range .spec.template.spec.containers}}{{.resources.limits.memory}}{{end}}'",
			)
			Expect(getMemoryLimit).To(ContainSubstring("700Mi"))
			getMemoryRequest := runCmd("oc get dc wildfly-watch-" +
				appTestName +
				" -o go-template='{{range .spec.template.spec.containers}}{{.resources.requests.memory}}{{end}}'",
			)
			Expect(getMemoryRequest).To(ContainSubstring("400Mi"))
		})

		It("watch openjdk component created from local source", func() {
			runCmd("git clone " + openjdkURI + " " + tmpDir + "/javalin-helloworld")
			runCmd("odo create openjdk18 openjdk-watch --local " + tmpDir + "/javalin-helloworld --min-memory 400Mi --max-memory 700Mi")
			runCmd("odo push -v 4")
			// Test multiple push so as to avoid regressions like: https://github.com/redhat-developer/odo/issues/1054
			runCmd("odo push -v 4")
			runCmd("odo url create --port 8080")

			startSimulationCh := make(chan bool)
			go func() {
				startMsg := <-startSimulationCh
				if startMsg {
					fmt.Println("Received signal, starting file modification simulation")
					runCmd("mkdir -p " + tmpDir + "/javalin-helloworld/src" + "/'.a b'")
					runCmd("mkdir -p " + tmpDir + "/javalin-helloworld/src" + "/'a b'")
					runCmd("touch " + tmpDir + "/javalin-helloworld/src" + "/'a b.txt'")
					runCmd(fmt.Sprintf("mkdir -p %s/javalin-helloworld/tests", tmpDir))
					runCmd(fmt.Sprintf("touch %s/javalin-helloworld/tests/test_1.java", tmpDir))

					// Delete during watch
					runCmd(fmt.Sprintf("rm -fr %s/javalin-helloworld/tests", tmpDir))
					runCmd("rm -fr " + tmpDir + "/javalin-helloworld/src/'a b.txt'")

					fileModificationPath := filepath.Join(
						tmpDir,
						"javalin-helloworld",
						"src",
						"main",
						"java",
						"Application.java",
					)
					fileModificationCmd := fmt.Sprintf(
						"sed -i 's/Hello World/Hello odo/g' %s",
						fileModificationPath,
					)
					runCmd(fileModificationCmd)
				}
			}()
			success, err := pollNonRetCmdStdOutForString(
				"odo watch openjdk-watch -v 4",
				time.Duration(5)*time.Minute,
				func(output string) bool {
					url := runCmd("odo url list | grep `odo component get -q` | grep 8080 | awk '{print $2}' | tr -d '\n'")
					curlOp := runCmd(fmt.Sprintf("curl %s", url))
					if strings.Contains(curlOp, "Hello odo") {
						// Verify delete from component pod
						podName := runCmd("oc get pods | grep openjdk-watch | awk '{print $1}' | tr -d '\n'")
						runFailCmd("oc exec "+podName+" -c openjdk-watch-testing -- ls -lai /tmp/src/tests/test_1.java /opt/app-root/src-backup/src/tests/test_1.java;exit", 2)
						runCmd("oc exec " + podName + " -c openjdk-watch-testing -- ls -lai /tmp/src/src/ | grep 'a b';exit")
						runFailCmd("oc exec "+podName+" -c openjdk-watch-testing -- ls -lai /tmp/src/tests;exit", 2)
						runFailCmd("oc exec "+podName+" -c openjdk-watch-testing -- ls -lai /opt/app-root/src-backup/src/tests;exit", 2)
						runFailCmd("oc exec "+podName+" -c openjdk-watch-testing -- ls -lai /tmp/src/src/ | grep 'a b.txt';exit", 1)
					}
					return strings.Contains(curlOp, "Hello odo")
				},
				startSimulationCh,
				func(output string) bool {
					return strings.Contains(output, "Waiting for something to change")
				})
			Expect(success).To(Equal(true))
			Expect(err).To(BeNil())

			// Verify memory limits to be same as configured
			getMemoryLimit := runCmd("oc get dc openjdk-watch-" +
				appTestName +
				" -o go-template='{{range .spec.template.spec.containers}}{{.resources.limits.memory}}{{end}}'",
			)
			Expect(getMemoryLimit).To(ContainSubstring("700Mi"))
			getMemoryRequest := runCmd("oc get dc openjdk-watch-" +
				appTestName +
				" -o go-template='{{range .spec.template.spec.containers}}{{.resources.requests.memory}}{{end}}'",
			)
			Expect(getMemoryRequest).To(ContainSubstring("400Mi"))
		})

		It("watch openjdk component created from local binary", func() {
			runCmd("git clone " + openjdkURI + " " + tmpDir + "/binary/javalin-helloworld")
			runCmd("mvn package -f " + tmpDir + "/binary/javalin-helloworld")
			runCmd("odo create openjdk18 openjdk-watch-binary --binary " + tmpDir + "/binary/javalin-helloworld/target/javalin-hello-world-0.1-SNAPSHOT.jar --min-memory 400Mi --max-memory 700Mi")
			runCmd("odo push -v 4")
			// Test multiple push so as to avoid regressions like: https://github.com/redhat-developer/odo/issues/1054
			runCmd("odo push -v 4")
			runCmd("odo url create --port 8080")

			startSimulationCh := make(chan bool)
			go func() {
				startMsg := <-startSimulationCh
				if startMsg {
					fmt.Println("Received signal, starting file modification simulation")

					fileModificationPath := filepath.Join(
						tmpDir,
						"binary",
						"javalin-helloworld",
						"src",
						"main",
						"java",
						"Application.java",
					)

					fileModificationCmd := fmt.Sprintf(
						"sed -i 's/Hello World/Hello odo/g' %s",
						fileModificationPath,
					)
					runCmd(fileModificationCmd)
					runCmd("mvn package -f " + tmpDir + "/binary/javalin-helloworld")
				}
			}()
			success, err := pollNonRetCmdStdOutForString(
				"odo watch openjdk-watch-binary -v 4",
				time.Duration(5)*time.Minute,
				func(output string) bool {
					url := runCmd("odo url list | grep `odo component get -q` | grep 8080 | awk '{print $2}' | tr -d '\n'")
					curlOp := runCmd(fmt.Sprintf("curl %s", url))
					return strings.Contains(curlOp, "Hello odo")
				},
				startSimulationCh,
				func(output string) bool {
					fmt.Println(output)
					return strings.Contains(output, "Waiting for something to change")
				},
			)
			Expect(success).To(Equal(true))
			Expect(err).To(BeNil())

			// Verify memory limits to be same as configured
			getMemoryLimit := runCmd("oc get dc openjdk-watch-binary-" +
				appTestName +
				" -o go-template='{{range .spec.template.spec.containers}}{{.resources.limits.memory}}{{end}}'",
			)
			Expect(getMemoryLimit).To(ContainSubstring("700Mi"))
			getMemoryRequest := runCmd("oc get dc openjdk-watch-binary-" +
				appTestName +
				" -o go-template='{{range .spec.template.spec.containers}}{{.resources.requests.memory}}{{end}}'",
			)
			Expect(getMemoryRequest).To(ContainSubstring("400Mi"))
		})
		It("watch wildfly component created from binary", func() {
			runCmd("git clone " + wildflyURI + " " + tmpDir + "/binary/katacoda-odo-backend")
			runCmd("mvn package -f " + tmpDir + "/binary/katacoda-odo-backend")
			runCmd("odo create wildfly wildfly-watch-binary --binary " + tmpDir + "/binary/katacoda-odo-backend/target/ROOT.war --min-memory 400Mi --max-memory 700Mi")
			runCmd("odo push -v 4")
			// Test multiple push so as to avoid regressions like: https://github.com/redhat-developer/odo/issues/1054
			runCmd("odo push -v 4")
			runCmd("odo url create --port 8080")

			startSimulationCh := make(chan bool)
			go func() {
				startMsg := <-startSimulationCh
				if startMsg {
					fmt.Println("Received signal, starting file modification simulation")

					runCmd(fmt.Sprintf("mkdir -p %s/binary/katacoda-odo-backend/tests", tmpDir))
					runCmd(fmt.Sprintf("touch %s/binary/katacoda-odo-backend/tests/test_1.java", tmpDir))
					runCmd("mkdir -p " + tmpDir + "/binary/katacoda-odo-backend/src" + "/'.a b'")
					runCmd("mkdir -p " + tmpDir + "/binary/katacoda-odo-backend/src" + "/'a b'")
					runCmd("touch " + tmpDir + "/binary/katacoda-odo-backend/src" + "/'a b.txt'")
					// Delete during watch
					runCmd(fmt.Sprintf("rm -fr %s/binary/katacoda-odo-backend/tests", tmpDir))
					runCmd("rm -fr " + tmpDir + "/binary/katacoda-odo-backend/src/'a b.txt'")

					fileModificationPath := filepath.Join(
						tmpDir,
						"binary",
						"katacoda-odo-backend",
						"src",
						"main",
						"java",
						"eu",
						"mjelen",
						"katacoda",
						"odo",
						"BackendServlet.java",
					)
					fmt.Printf("Triggering filemodification @; %s\n", fileModificationPath)
					fileModificationCmd := fmt.Sprintf(
						"sed -i 's/response.getWriter().println(counter.toString())/response.getWriter().println(\"Hello odo\" + counter.toString())/g' %s",
						fileModificationPath,
					)
					runCmd(fileModificationCmd)
					runCmd("mvn package -f " + tmpDir + "/binary/katacoda-odo-backend")
				}
			}()
			success, err := pollNonRetCmdStdOutForString(
				"odo watch wildfly-watch-binary -v 4",
				time.Duration(5)*time.Minute,
				func(output string) bool {
					url := runCmd("odo url list | grep `odo component get -q` | grep 8080 | awk '{print $2}' | tr -d '\n'")
					url = fmt.Sprintf("%s/counter", url)
					curlOp := runCmd(fmt.Sprintf("curl %s", url))
					return strings.Contains(curlOp, "Hello odo")
				},
				startSimulationCh,
				func(output string) bool {
					return strings.Contains(output, "Waiting for something to change")
				})
			Expect(success).To(Equal(true))
			Expect(err).To(BeNil())

			// Verify memory limits to be same as configured
			getMemoryLimit := runCmd("oc get dc wildfly-watch-binary-" +
				appTestName +
				" -o go-template='{{range .spec.template.spec.containers}}{{.resources.limits.memory}}{{end}}'",
			)
			Expect(getMemoryLimit).To(ContainSubstring("700Mi"))
			getMemoryRequest := runCmd("oc get dc wildfly-watch-binary-" +
				appTestName +
				" -o go-template='{{range .spec.template.spec.containers}}{{.resources.requests.memory}}{{end}}'",
			)
			Expect(getMemoryRequest).To(ContainSubstring("400Mi"))
		})
	})
})
