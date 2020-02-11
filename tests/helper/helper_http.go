package helper

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// HttpWaitForWithStatus periodically (every interval) calls GET to given url
// ends when result response contains match string and status code, or after the maxRetry
func HttpWaitForWithStatus(url string, match string, maxRetry int, interval int, expectedCode int) {
	fmt.Fprintf(GinkgoWriter, "Checking %s, for %s\n", url, match)

	var body []byte

	for i := 0; i < maxRetry; i++ {
		fmt.Fprintf(GinkgoWriter, "try %d of %d\n", i, maxRetry)

		// #nosec
		// gosec:G107 -> This is safe since it's just used for testing.
		resp, err := http.Get(url)
		if err != nil {
			// we log the error and sleep again because this could mean the component is not up yet
			fmt.Fprintln(GinkgoWriter, "error while requesting:", err.Error())
			time.Sleep(time.Duration(interval) * time.Second)
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode == expectedCode {
			body, _ = ioutil.ReadAll(resp.Body)
			if strings.Contains(string(body), match) {
				return
			}

		}
		time.Sleep(time.Duration(interval) * time.Second)
	}
	fmt.Fprintf(GinkgoWriter, "Last output from %s: %s\n", url, string(body))
	Fail(fmt.Sprintf("Failed after %d retries. Content in %s doesn't include '%s'.", maxRetry, url, match))
}

// HttpWaitFor periodically (every interval) calls GET to given url
// ends when a 200 HTTP result response contains match string, or after the maxRetry
func HttpWaitFor(url string, match string, maxRetry int, interval int) {
	HttpWaitForWithStatus(url, match, maxRetry, interval, 200)
}

// HttpGetFreePort gets a free port from the system
func HttpGetFreePort() int {
	listener, err := net.Listen("tcp", "localhost:0")
	Expect(err).NotTo(HaveOccurred())
	freePort := listener.Addr().(*net.TCPAddr).Port
	err = listener.Close()
	Expect(err).NotTo(HaveOccurred())
	return freePort
}
