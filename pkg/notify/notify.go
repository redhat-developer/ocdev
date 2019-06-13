package notify

import (
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/blang/semver"
	"github.com/pkg/errors"
)

const (
	// VersionFetchURL is the URL to fetch latest version number
	VersionFetchURL = "https://raw.githubusercontent.com/openshift/odo/master/build/VERSION"
	// InstallScriptURL is URL of the installation shell script
	InstallScriptURL = "https://raw.githubusercontent.com/openshift/odo/master/scripts/installer.sh"
)

// getLatestReleaseTag polls odo's upstream GitHub repository to get the
// tag of the latest release
func getLatestReleaseTag() (string, error) {

	resp, err := http.Get(VersionFetchURL)
	if err != nil {
		return "", errors.Wrap(err, "error getting latest release")
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "error getting latest release")
	}

	return strings.TrimSuffix(string(body), "\n"), nil
}

// CheckLatestReleaseTag returns the latest release tag if a newer latest
// release is available, else returns an empty string
func CheckLatestReleaseTag(currentVersion string) (string, error) {
	currentSemver, err := semver.Make(strings.TrimPrefix(currentVersion, "v"))
	if err != nil {
		return "", errors.Wrapf(err, "unable to make semver from the current version: %v", currentVersion)
	}

	latestTag, err := getLatestReleaseTag()
	if err != nil {
		return "", errors.Wrap(err, "unable to get latest release tag")
	}

	latestSemver, err := semver.Make(strings.TrimPrefix(latestTag, "v"))
	if err != nil {
		return "", errors.Wrapf(err, "unable to make semver from the latest release tag: %v", latestTag)
	}

	if currentSemver.LT(latestSemver) {
		return latestTag, nil
	}

	return "", nil
}
