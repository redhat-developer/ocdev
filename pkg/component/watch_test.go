// +build !windows,!osx

package component

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/openshift/odo/pkg/occlient"
	"github.com/openshift/odo/pkg/testingutil"
	"github.com/pkg/errors"
)

// setUpF8AnalyticsComponentSrc sets up a mock analytics component source base for observing changes to source files.
// Parameters:
//	componentName: Name of the source directory
//	requiredFilePaths: list of required sources, their description like whether regularfile/directory, parent directory path of source and desired modification type like update/create/delete/append
// Returns:
//	absolute base path of source code
//	directory structure containing mappings from desired relative paths to their respective absolute path containing FileProperties.
func setUpF8AnalyticsComponentSrc(componentName string, requiredFilePaths []testingutil.FileProperties) (string, map[string]testingutil.FileProperties, error) {

	// retVal is mappings from desired relative paths to their respective absolute path containing FileProperties.
	// This is required because ioutil#TempFile and ioutil#TempFolder creates paths with random numeric suffixes.
	// So, to be able to refer to the file/folder at any later point in time the created paths returned by ioutil#TempFile or ioutil#TempFolder will need to be saved.
	retVal := make(map[string]testingutil.FileProperties)
	dirTreeMappings := make(map[string]string)
	basePath := ""

	// Create temporary directory for mock component source code
	srcPath, err := testingutil.TempMkdir(basePath, componentName)
	if err != nil {
		return "", retVal, errors.Wrapf(err, "failed to create dir %s under %s", componentName, basePath)
	}
	dirTreeMappings[componentName] = srcPath

	// For each of the passed(desired) files/folders under component source
	for _, fileProperties := range requiredFilePaths {

		// get relative path using file parent and file name passed
		relativePath := filepath.Join(fileProperties.FileParent, fileProperties.FilePath)

		// get its absolute path using the mappings preserved from previous creates
		if realParentPath, ok := dirTreeMappings[fileProperties.FileParent]; ok {
			// real path for the intended file operation is obtained from previously maintained directory tree mappings by joining parent path and file name
			realPath := filepath.Join(realParentPath, fileProperties.FilePath)
			// Preserve the new paths for further reference
			fileProperties.FilePath = filepath.Base(realPath)
			fileProperties.FileParent, _ = filepath.Rel(srcPath, filepath.Dir(realPath))
		}

		// Perform mock operation as requested by the parameter
		newPath, err := testingutil.SimulateFileModifications(srcPath, fileProperties)
		dirTreeMappings[relativePath] = newPath
		if err != nil {
			return "", retVal, errors.Wrapf(err, "unable to setup test env")
		}

		fileProperties.FilePath = filepath.Base(newPath)
		fileProperties.FileParent = filepath.Dir(newPath)
		retVal[relativePath] = fileProperties
	}

	// Return base source path and directory tree mappings
	return srcPath, retVal, nil
}

// ExpectedChangedFiles is required so that the mockPushLocal below can validate obtained changes set against the test expected changes
var ExpectedChangedFiles []string

// DeleteFiles is required to validate deleted changes set against the test expected changes
var DeleteFiles []string

// CompDirStructure is required to hold the directory structure of mock component created by the test which can be accessed by mockPushLocal
var (
	muLock           sync.Mutex
	CompDirStructure map[string]testingutil.FileProperties
)

// ExtChan is used to return from otherwise non-terminating(without SIGINT) end of ever running watch function
var ExtChan = make(chan bool)
var StartChan = make(chan bool)

type mockPushParameters struct {
	componentName   string
	applicationName string
	path            string
	isForcePush     bool
	globExps        []string
	show            bool
}

var mockPush mockPushParameters

// Mock PushLocal to collect changed files and compare against expected changed files
func mockPushLocal(client *occlient.Client, componentName string, applicationName string, path string, out io.Writer, files []string, delFiles []string, isPushForce bool, globExps []string, show bool) error {
	muLock.Lock()
	defer muLock.Unlock()
	if componentName != mockPush.componentName || applicationName != mockPush.applicationName || isPushForce != mockPush.isForcePush || show != mockPush.show {
		fmt.Printf("some of the push parameters are different, wanted: %v, got: %v", mockPush, []string{
			componentName, applicationName, "isPushForce:" + strconv.FormatBool(isPushForce), "show:" + strconv.FormatBool(show),
		})
		os.Exit(1)
	}
	sort.Strings(globExps)
	sort.Strings(mockPush.globExps)
	if !reflect.DeepEqual(globExps, mockPush.globExps) {
		fmt.Printf("some of the push parameters are different, wanted: %v, got: %v", mockPush.globExps, globExps)
		os.Exit(1)
	}

	for _, expChangedFile := range ExpectedChangedFiles {
		found := false
		// Verify every file in expected file changes to be actually observed to be changed
		// If found exactly same or different, return from PushLocal and signal exit for watch so that the watch terminates gracefully
		for _, gotChangedFile := range files {
			wantedFileDetail := CompDirStructure[expChangedFile]
			if filepath.Join(wantedFileDetail.FileParent, wantedFileDetail.FilePath) == gotChangedFile {
				found = true
			}
		}
		if !found {
			ExtChan <- true
			fmt.Printf("received %+v which is not same as expected list %+v", files, strings.Join(ExpectedChangedFiles, ","))
			os.Exit(1)
		}
	}

	for _, deletedFile := range DeleteFiles {
		found := false
		// Verify every file in expected deleted file changes to be actually observed to be changed
		// If found exactly same or different, return from PushLocal and signal exit for watch so that the watch terminates gracefully
		for _, gotChangedFile := range delFiles {
			wantedFileDetail := CompDirStructure[deletedFile]
			if filepath.Join(wantedFileDetail.FileParent, wantedFileDetail.FilePath) == filepath.Join(wantedFileDetail.FileParent, filepath.Base(gotChangedFile)) {
				found = true
			}
		}
		if !found {
			ExtChan <- true
			fmt.Printf("received deleted files: %+v which is not same as expected list %+v", delFiles, strings.Join(DeleteFiles, ","))
			os.Exit(1)
		}
	}

	ExtChan <- true
	return nil
}

func TestWatchAndPush(t *testing.T) {
	tests := []struct {
		name              string
		componentName     string
		applicationName   string
		path              string
		ignores           []string
		show              bool
		forcePush         bool
		delayInterval     int
		wantErr           bool
		want              []string
		wantDeleted       []string
		fileModifications []testingutil.FileProperties
		requiredFilePaths []testingutil.FileProperties
		setupEnv          func(componentName string, requiredFilePaths []testingutil.FileProperties) (string, map[string]testingutil.FileProperties, error)
	}{
		{
			name:            "Case 1: Valid watch with list of files to be ignored with a append event",
			componentName:   "license-analysis",
			applicationName: "fabric8-analytics",
			path:            "fabric8-analytics-license-analysis",
			ignores:         []string{".git", "tests/", "LICENSE"},
			delayInterval:   1,
			wantErr:         false,
			show:            false,
			forcePush:       false,
			requiredFilePaths: []testingutil.FileProperties{
				{
					FilePath:         "src",
					FileParent:       "",
					FileType:         testingutil.Directory,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "tests",
					FileParent:       "",
					FileType:         testingutil.Directory,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         ".git",
					FileParent:       "",
					FileType:         testingutil.Directory,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "LICENSE",
					FileParent:       "",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "__init__.py",
					FileParent:       "",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "__init__.py",
					FileParent:       "src",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "main.py",
					FileParent:       "src",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "__init__.py",
					FileParent:       "tests",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "test1.py",
					FileParent:       "tests",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
			},
			fileModifications: []testingutil.FileProperties{
				{
					FilePath:         "__init__.py",
					FileParent:       "",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.APPEND,
				},
				{
					FilePath:         "read_licenses.py",
					FileParent:       "src",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "read_licenses.py",
					FileParent:       "src",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.APPEND,
				},
				{
					FilePath:         "test_read_licenses.py",
					FileParent:       "tests",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "tests",
					FileParent:       "",
					FileType:         testingutil.Directory,
					ModificationType: testingutil.DELETE,
				},
			},
			want:        []string{"src/read_licenses.py", "__init__.py"},
			wantDeleted: []string{},
			setupEnv:    setUpF8AnalyticsComponentSrc,
		},
		{
			name:            "Case 2: Valid watch with list of files to be ignored with a append and a delete event",
			componentName:   "license-analysis",
			applicationName: "fabric8-analytics",
			path:            "fabric8-analytics-license-analysis",
			ignores:         []string{".git", "tests/", "LICENSE"},
			delayInterval:   1,
			wantErr:         false,
			show:            false,
			forcePush:       false,
			requiredFilePaths: []testingutil.FileProperties{
				{
					FilePath:         "src",
					FileParent:       "",
					FileType:         testingutil.Directory,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "tests",
					FileParent:       "",
					FileType:         testingutil.Directory,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         ".git",
					FileParent:       "",
					FileType:         testingutil.Directory,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "LICENSE",
					FileParent:       "",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "__init__.py",
					FileParent:       "",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "__init__.py",
					FileParent:       "src",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "main.py",
					FileParent:       "src",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "__init__.py",
					FileParent:       "tests",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "test1.py",
					FileParent:       "tests",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "read_licenses.py",
					FileParent:       "src",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
			},
			fileModifications: []testingutil.FileProperties{
				{
					FilePath:         "__init__.py",
					FileParent:       "",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.APPEND,
				},
				{
					FilePath:         "read_licenses.py",
					FileParent:       "src",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.DELETE,
				},
				{
					FilePath:         "test_read_licenses.py",
					FileParent:       "tests",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "tests",
					FileParent:       "",
					FileType:         testingutil.Directory,
					ModificationType: testingutil.DELETE,
				},
			},
			want:        []string{"__init__.py"},
			wantDeleted: []string{"src/read_licenses.py"},
			setupEnv:    setUpF8AnalyticsComponentSrc,
		},
		{
			name:            "Case 3: Valid watch with list of files to be ignored with a create and a delete event",
			componentName:   "license-analysis",
			applicationName: "fabric8-analytics",
			path:            "fabric8-analytics-license-analysis",
			ignores:         []string{".git", "tests/", "LICENSE"},
			delayInterval:   1,
			wantErr:         false,
			show:            false,
			forcePush:       false,
			requiredFilePaths: []testingutil.FileProperties{
				{
					FilePath:         "src",
					FileParent:       "",
					FileType:         testingutil.Directory,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "tests",
					FileParent:       "",
					FileType:         testingutil.Directory,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         ".git",
					FileParent:       "",
					FileType:         testingutil.Directory,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "LICENSE",
					FileParent:       "",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "__init__.py",
					FileParent:       "src",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "main.py",
					FileParent:       "src",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "__init__.py",
					FileParent:       "tests",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "test1.py",
					FileParent:       "tests",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "read_licenses.py",
					FileParent:       "src",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
			},
			fileModifications: []testingutil.FileProperties{
				{
					FilePath:         "__init__.py",
					FileParent:       "",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "read_licenses.py",
					FileParent:       "src",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.DELETE,
				},
				{
					FilePath:         "test_read_licenses.py",
					FileParent:       "tests",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "tests",
					FileParent:       "",
					FileType:         testingutil.Directory,
					ModificationType: testingutil.DELETE,
				},
			},
			want:        []string{"__init__.py"},
			wantDeleted: []string{"src/read_licenses.py"},
			setupEnv:    setUpF8AnalyticsComponentSrc,
		},
		{
			name:            "Case 4: Valid watch with list of files to be ignored with a folder create event",
			componentName:   "license-analysis",
			applicationName: "fabric8-analytics",
			path:            "fabric8-analytics-license-analysis",
			ignores:         []string{".git", "tests/", "LICENSE"},
			delayInterval:   1,
			wantErr:         false,
			show:            false,
			forcePush:       false,
			requiredFilePaths: []testingutil.FileProperties{
				{
					FilePath:         "src",
					FileParent:       "",
					FileType:         testingutil.Directory,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "tests",
					FileParent:       "",
					FileType:         testingutil.Directory,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         ".git",
					FileParent:       "",
					FileType:         testingutil.Directory,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "LICENSE",
					FileParent:       "",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "__init__.py",
					FileParent:       "src",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "main.py",
					FileParent:       "src",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "__init__.py",
					FileParent:       "tests",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "test1.py",
					FileParent:       "tests",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "read_licenses.py",
					FileParent:       "src",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
			},
			fileModifications: []testingutil.FileProperties{
				{
					FilePath:         "__init__.py",
					FileParent:       "",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "read_licenses.py",
					FileParent:       "src",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.DELETE,
				},
				{
					FilePath:         "test_read_licenses.py",
					FileParent:       "tests",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "tests",
					FileParent:       "",
					FileType:         testingutil.Directory,
					ModificationType: testingutil.DELETE,
				},
				{
					FilePath:         "bin",
					FileParent:       "",
					FileType:         testingutil.Directory,
					ModificationType: testingutil.CREATE,
				},
			},
			want:        []string{"__init__.py"},
			wantDeleted: []string{"src/read_licenses.py"},
			setupEnv:    setUpF8AnalyticsComponentSrc,
		},
		{
			name:            "Case 5: Valid watch with list of files to be ignored with a folder delete event",
			componentName:   "license-analysis",
			applicationName: "fabric8-analytics",
			path:            "fabric8-analytics-license-analysis",
			ignores:         []string{".git", "tests/", "LICENSE"},
			delayInterval:   1,
			wantErr:         false,
			show:            false,
			forcePush:       false,
			requiredFilePaths: []testingutil.FileProperties{
				{
					FilePath:         "src",
					FileParent:       "",
					FileType:         testingutil.Directory,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "tests",
					FileParent:       "",
					FileType:         testingutil.Directory,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         ".git",
					FileParent:       "",
					FileType:         testingutil.Directory,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "LICENSE",
					FileParent:       "",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "__init__.py",
					FileParent:       "src",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "main.py",
					FileParent:       "src",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "__init__.py",
					FileParent:       "tests",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "test1.py",
					FileParent:       "tests",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "read_licenses.py",
					FileParent:       "src",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "bin",
					FileParent:       "",
					FileType:         testingutil.Directory,
					ModificationType: testingutil.CREATE,
				},
			},
			fileModifications: []testingutil.FileProperties{
				{
					FilePath:         "__init__.py",
					FileParent:       "",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "read_licenses.py",
					FileParent:       "src",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.DELETE,
				},
				{
					FilePath:         "test_read_licenses.py",
					FileParent:       "tests",
					FileType:         testingutil.RegularFile,
					ModificationType: testingutil.CREATE,
				},
				{
					FilePath:         "tests",
					FileParent:       "",
					FileType:         testingutil.Directory,
					ModificationType: testingutil.DELETE,
				},
				{
					FilePath:         "bin",
					FileParent:       "",
					FileType:         testingutil.Directory,
					ModificationType: testingutil.DELETE,
				},
			},
			want:        []string{"__init__.py"},
			wantDeleted: []string{"src/read_licenses.py"},
			setupEnv:    setUpF8AnalyticsComponentSrc,
		},
	}

	for _, tt := range tests {
		ExtChan = make(chan bool)
		StartChan = make(chan bool)
		t.Log("Running test: ", tt.name)
		t.Run(tt.name, func(t *testing.T) {
			mockPush = mockPushParameters{
				componentName:   tt.componentName,
				applicationName: tt.applicationName,
				path:            tt.path,
				isForcePush:     tt.forcePush,
				globExps:        tt.ignores,
				show:            tt.show,
			}

			ExpectedChangedFiles = tt.want
			DeleteFiles = tt.wantDeleted
			// Create mock component source
			basePath, dirStructure, err := tt.setupEnv(tt.path, tt.requiredFilePaths)
			CompDirStructure = dirStructure
			if err != nil {
				t.Errorf("failed to setup test environment. Error %v", err)
			}

			fkclient, _ := occlient.FakeNew()

			// Clear all the created temporary files
			defer os.RemoveAll(basePath)
			t.Logf("Done with basePath creation and client init will trigger WatchAndPush and file modifications next...\n%+v\n", CompDirStructure)

			go func() {
				t.Logf("Starting file simulations \n%+v\n", tt.fileModifications)
				// Simulating file modifications for watch to observe
				pingTimeout := time.After(time.Duration(1) * time.Minute)
				for {
					select {
					case startMsg := <-StartChan:
						if startMsg {
							for _, fileModification := range tt.fileModifications {

								intendedFileRelPath := fileModification.FilePath
								if fileModification.FileParent != "" {
									intendedFileRelPath = filepath.Join(fileModification.FileParent, fileModification.FilePath)
								}

								fileModification.FileParent = CompDirStructure[fileModification.FileParent].FilePath
								if _, ok := CompDirStructure[intendedFileRelPath]; ok {
									fileModification.FilePath = CompDirStructure[intendedFileRelPath].FilePath
								}

								newFilePath, err := testingutil.SimulateFileModifications(basePath, fileModification)
								if err != nil {
									t.Errorf("CompDirStructure: %+v\nFileModification %+v\nError %v\n", CompDirStructure, fileModification, err)
								}

								// If file operation is create, store even such modifications in dir structure for future references
								if _, ok := CompDirStructure[intendedFileRelPath]; !ok && fileModification.ModificationType == testingutil.CREATE {
									muLock.Lock()
									CompDirStructure[intendedFileRelPath] = testingutil.FileProperties{
										FilePath:         filepath.Base(newFilePath),
										FileParent:       filepath.Dir(newFilePath),
										FileType:         testingutil.Directory,
										ModificationType: testingutil.CREATE,
									}
									muLock.Unlock()
								}
							}
						}
						t.Logf("The CompDirStructure is \n%+v\n", CompDirStructure)
						return
					case <-pingTimeout:
						break
					}
				}
			}()

			// Start WatchAndPush, the unit tested function
			t.Logf("Starting WatchAndPush now\n")
			err = WatchAndPush(
				fkclient,
				new(bytes.Buffer),
				WatchParameters{
					ComponentName:   tt.componentName,
					ApplicationName: tt.applicationName,
					Path:            basePath,
					FileIgnores:     tt.ignores,
					PushDiffDelay:   tt.delayInterval,
					StartChan:       StartChan,
					ExtChan:         ExtChan,
					Show:            tt.show,
					WatchHandler:    mockPushLocal,
				},
			)
			if err != nil && err != ErrUserRequestedWatchExit {
				t.Errorf("error in WatchAndPush %+v", err)
			}
		})
	}
}
