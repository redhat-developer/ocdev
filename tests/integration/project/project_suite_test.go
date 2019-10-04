package project

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/odo/tests/helper/reporter"
)

func TestProject(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecsWithDefaultAndCustomReporters(t, "Project Suite", []Reporter{reporter.JunitReport(t, "../../../reports")})
}
