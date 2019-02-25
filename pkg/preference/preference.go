package preference

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/redhat-developer/odo/pkg/util"
)

const (
	globalConfigEnvName = "GLOBALODOCONFIG"
	configFileName      = "odo-config.yaml"
	//DefaultTimeout for openshift server connection check
	DefaultTimeout = 1

	// UpdateNotificationSetting is the name of the setting controlling update notification
	UpdateNotificationSetting = "UpdateNotification"
	// UpdateNotificationSettingDescription is human-readable description for the update notification setting
	UpdateNotificationSettingDescription = "Controls if an update notification is shown or not (true or false)"
	// NamePrefixSetting is the name of the setting controlling name prefix
	NamePrefixSetting = "NamePrefix"
	// NamePrefixSettingDescription is human-readable description for the name prefix setting
	NamePrefixSettingDescription = "Default prefix is the current directory name. Use this value to set a default name prefix"
	// TimeoutSetting is the name of the setting controlling timeout for connection check
	TimeoutSetting = "Timeout"
	// TimeoutSettingDescription is human-readable description for the timeout setting
	TimeoutSettingDescription = "Timeout (in seconds) for OpenShift server connection check"
)

// This value can be provided to set a seperate directory for users 'homedir' resolution
// note for mocking purpose ONLY
var customHomeDir = os.Getenv("CUSTOM_HOMEDIR")

var (
	// records information on supported parameters
	supportedParameterDescriptions = map[string]string{
		UpdateNotificationSetting: UpdateNotificationSettingDescription,
		NamePrefixSetting:         NamePrefixSettingDescription,
		TimeoutSetting:            TimeoutSettingDescription,
	}

	// set-like map to quickly check if a parameter is supported
	lowerCaseParameters = util.GetLowerCaseParameters(GetSupportedParameters())
)

// GlobalConfigInfo wraps the global config and provides helpers to
// serialize it.
type GlobalConfigInfo struct {
	Filename     string `json:"FileName,omitempty"`
	GlobalConfig `json:",omitempty"`
}

// OdoSettings holds all odo specific configurations
type OdoSettings struct {
	// Controls if an update notification is shown or not
	UpdateNotification *bool `json:"UpdateNotification,omitempty"`
	// Holds the prefix part of generated random application name
	NamePrefix *string `json:"NamePrefix,omitempty"`
	// Timeout for openshift server connection check
	Timeout *int `json:"Timeout,omitempty"`
}

// ApplicationInfo holds all important information about one application
type ApplicationInfo struct {
	// name of the application
	Name string `json:"Name"`
	// is this application active? Only one application can be active at the time
	Active bool `json:"Active"`
	// name of the openshift project this application belongs to
	Project string `json:"Project"`
	// last active component for  this application
	ActiveComponent string `json:"ActiveComponent"`
}

// GlobalConfig stores all the config related to odo, its the superset of
// local config.
type GlobalConfig struct {
	// remember active applications and components per project
	// when project or applications is switched we can go back to last active app/component

	// Currently active application
	// multiple applications can be active but each one has to be in different project
	// there shouldn't be more active applications in one project
	ActiveApplications []ApplicationInfo `json:"ActiveApplications"`

	// Odo settings holds the odo specific global settings
	OdoSettings OdoSettings `json:"Settings,omitempty"`
}

func getGlobalConfigFile() (string, error) {
	if env, ok := os.LookupEnv(globalConfigEnvName); ok {
		return env, nil
	}

	if len(customHomeDir) != 0 {
		return filepath.Join(customHomeDir, ".odo", configFileName), nil
	}

	currentUser, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(currentUser.HomeDir, ".odo", configFileName), nil
}

// New returns the globalConfigInfo to retain the expected behavior
func New() (*GlobalConfigInfo, error) {
	return NewGlobalConfig()
}

// NewGlobalConfig gets the GlobalConfigInfo from global config file and global creates the config file in case it's
// not present then it
func NewGlobalConfig() (*GlobalConfigInfo, error) {
	configFile, err := getGlobalConfigFile()
	if err != nil {
		return nil, errors.Wrap(err, "unable to get odo config file")
	}
	// Check whether directory and file are not present if they aren't then create them
	if err = util.CreateIfNotExists(configFile); err != nil {
		return nil, err
	}
	c := GlobalConfigInfo{
		GlobalConfig: GlobalConfig{},
	}
	c.Filename = configFile
	err = util.GetFromFile(&c.GlobalConfig, c.Filename)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// SetConfiguration modifies Odo configurations in the config file
// as of now being used for nameprefix, timeout, updatenotification
// TODO: Use reflect to set parameters
func (c *GlobalConfigInfo) SetConfiguration(parameter string, value string) error {
	if p, ok := asSupportedParameter(parameter); ok {
		// processing values according to the parameter names
		switch p {

		case "timeout":
			typedval, err := strconv.Atoi(value)
			if err != nil {
				return errors.Wrapf(err, "unable to set %s to %s", parameter, value)
			}
			if typedval < 0 {
				return errors.Errorf("cannot set timeout to less than 0")
			}
			c.OdoSettings.Timeout = &typedval
		case "updatenotification":
			val, err := strconv.ParseBool(strings.ToLower(value))
			if err != nil {
				return errors.Wrapf(err, "unable to set %s to %s", parameter, value)
			}
			c.OdoSettings.UpdateNotification = &val

		case "nameprefix":
			c.OdoSettings.NamePrefix = &value
		}
	} else {
		return errors.Errorf("unknown parameter :'%s' is not a parameter in global odo config", parameter)
	}

	err := util.WriteToFile(c.GlobalConfig, c.Filename)
	if err != nil {
		return errors.Wrapf(err, "unable to set %s", parameter)
	}
	return nil
}

// DeleteConfiguration delete Odo configurations in the global config file
// as of now being used for nameprefix, timeout, updatenotification
func (c *GlobalConfigInfo) DeleteConfiguration(parameter string) error {
	if p, ok := asSupportedParameter(parameter); ok {
		// processing values according to the parameter names

		if err := util.DeleteConfiguration(&c.OdoSettings, p); err != nil {
			return err
		}
	} else {
		return errors.Errorf("unknown parameter :'%s' is not a parameter in global odo config", parameter)
	}

	err := util.WriteToFile(c.GlobalConfig, c.Filename)
	if err != nil {
		return errors.Wrapf(err, "unable to set %s", parameter)
	}
	return nil
}

// GetConfiguration gets the value of the specified parameter, it returns false in
// case the value is not part of the struct
func (c *GlobalConfigInfo) GetConfiguration(parameter string) (interface{}, bool) {
	return util.GetConfiguration(c.OdoSettings, parameter)
}

// GetTimeout returns the value of Timeout from config
// and if absent then returns default
func (c *GlobalConfigInfo) GetTimeout() int {
	// default timeout value is 1
	if c.OdoSettings.Timeout == nil {
		return DefaultTimeout
	}
	return *c.OdoSettings.Timeout
}

// GetUpdateNotification returns the value of UpdateNotification from config
// and if absent then returns default
func (c *GlobalConfigInfo) GetUpdateNotification() bool {
	if c.OdoSettings.UpdateNotification == nil {
		return true
	}
	return *c.OdoSettings.UpdateNotification
}

// GetNamePrefix returns the value of Prefix from config
// and if absent then returns default
func (c *GlobalConfigInfo) GetNamePrefix() string {
	if c.OdoSettings.NamePrefix == nil {
		return ""
	}
	return *c.OdoSettings.NamePrefix
}

// SetActiveComponent sets active component for given project and application.
// application must exist
func (c *GlobalConfigInfo) SetActiveComponent(componentName string, applicationName string, projectName string) error {
	found := false

	if c.ActiveApplications != nil {
		for i, app := range c.ActiveApplications {
			if app.Project == projectName && app.Name == applicationName {
				c.ActiveApplications[i].ActiveComponent = componentName
				found = true
				break
			}
		}
	}

	if !found {
		return errors.Errorf("unable to set %s componentName as active, applicationName %s in %s projectName doesn't exists", componentName, applicationName, projectName)
	}

	err := util.WriteToFile(c.GlobalConfig, c.Filename)
	if err != nil {
		return errors.Wrapf(err, "unable to set %s as active componentName", componentName)
	}
	return nil
}

// UnsetActiveComponent sets the active component as blank of the given project in the configuration file
func (c *GlobalConfigInfo) UnsetActiveComponent(project string) error {
	if c.ActiveApplications == nil {
		c.ActiveApplications = []ApplicationInfo{}
	}

	for i, app := range c.ActiveApplications {
		if app.Project == project && c.ActiveApplications[i].ActiveComponent != "" {
			c.ActiveApplications[i].ActiveComponent = ""
		}
	}

	// Write the configuration to file
	err := util.WriteToFile(c.GlobalConfig, c.Filename)
	if err != nil {
		return errors.Wrapf(err, "unable to write configuration file")
	}
	return nil

}

// UnsetActiveApplication sets the active application as blank of the given project in the configuration file
func (c *GlobalConfigInfo) UnsetActiveApplication(project string) error {
	if c.ActiveApplications == nil {
		c.ActiveApplications = []ApplicationInfo{}
	}

	for i, cfgApp := range c.ActiveApplications {
		if cfgApp.Project == project && c.ActiveApplications[i].Active {
			c.ActiveApplications[i].Active = false
		}
	}

	err := util.WriteToFile(c.GlobalConfig, c.Filename)
	if err != nil {
		return errors.Wrap(err, "unable to write configuration file")
	}
	return nil
}

// GetActiveComponent if no component is set as current returns empty string
func (c *GlobalConfigInfo) GetActiveComponent(application string, project string) string {
	if c.ActiveApplications != nil {
		for _, app := range c.ActiveApplications {
			if app.Project == project && app.Name == application && app.Active == true {
				return app.ActiveComponent
			}
		}
	}
	return ""
}

// GetActiveApplication get currently active application for given project
// if no application is active return empty string
func (c *GlobalConfigInfo) GetActiveApplication(project string) string {
	if c.ActiveApplications != nil {
		for _, app := range c.ActiveApplications {
			if app.Project == project && app.Active == true {
				return app.Name
			}
		}
	}
	return ""
}

// SetActiveApplication set application as active for given project
func (c *GlobalConfigInfo) SetActiveApplication(application string, project string) error {
	if c.ActiveApplications == nil {
		c.ActiveApplications = []ApplicationInfo{}
	}

	found := false
	for i, app := range c.ActiveApplications {
		// if application exists set is as Active
		if app.Name == application && app.Project == project {
			c.ActiveApplications[i].Active = true
			found = true
			break
		}
	}

	// if application doesn't exists, add it as Active
	if !found {
		return fmt.Errorf("unable set application %s as active in config, it doesn't exist", application)
	}
	// make sure that no other application is active
	for i, app := range c.ActiveApplications {
		if !(app.Name == application && app.Project == project) {
			c.ActiveApplications[i].Active = false
		}
	}

	err := util.WriteToFile(c.GlobalConfig, c.Filename)
	if err != nil {
		return errors.Wrap(err, "unable to set current application")
	}
	return nil
}

// AddApplication add  new application to the config file
// Newly create application is NOT going to be se as Active.
func (c *GlobalConfigInfo) AddApplication(application string, project string) error {
	if c.ActiveApplications == nil {
		c.ActiveApplications = []ApplicationInfo{}
	}

	for _, app := range c.ActiveApplications {
		if app.Name == application && app.Project == project {
			return fmt.Errorf("unable to add %s application, it already exists in config file", application)
		}
	}

	// if application doesn't exists add it to slice
	c.ActiveApplications = append(c.ActiveApplications,
		ApplicationInfo{
			Name:    application,
			Project: project,
			Active:  false,
		})

	err := util.WriteToFile(c.GlobalConfig, c.Filename)
	if err != nil {
		return errors.Wrapf(err, "unable to set add %s application", application)
	}
	return nil
}

// DeleteApplication deletes application from given project from config file
func (c *GlobalConfigInfo) DeleteApplication(application string, project string) error {
	if c.ActiveApplications == nil {
		c.ActiveApplications = []ApplicationInfo{}
	}

	found := false
	for i, app := range c.ActiveApplications {
		// if application exists set is as Active
		if app.Name == application && app.Project == project {
			// remove current item from array
			c.ActiveApplications = append(c.ActiveApplications[:i], c.ActiveApplications[i+1:]...)
			found = true
		}
	}

	if !found {
		return fmt.Errorf("application %s doesn't exist", application)

	}

	err := util.WriteToFile(c.GlobalConfig, c.Filename)
	if err != nil {
		return errors.Wrapf(err, "unable to delete application %s", application)
	}
	return nil
}

// DeleteProject deletes applications belonging to the given project from the config file
func (c *GlobalConfigInfo) DeleteProject(projectName string) error {
	// looping in reverse and removing to avoid panic from index out of bounds
	for i := len(c.ActiveApplications) - 1; i >= 0; i-- {
		if c.ActiveApplications[i].Project == projectName {
			// remove current item from array
			c.ActiveApplications = append(c.ActiveApplications[:i], c.ActiveApplications[i+1:]...)
		}
	}
	err := util.WriteToFile(c.GlobalConfig, c.Filename)
	if err != nil {
		return errors.Wrapf(err, "unable to delete project from config")
	}
	return nil
}

// FormatSupportedParameters outputs supported parameters and their description
func FormatSupportedParameters() (result string) {
	for _, v := range GetSupportedParameters() {
		result = result + v + " - " + supportedParameterDescriptions[v] + "\n"
	}
	return "\nAvailable Parameters:\n" + result
}

// asSupportedParameter checks that the given parameter is supported and returns a lower case version of it if it is
func asSupportedParameter(param string) (string, bool) {
	lower := strings.ToLower(param)
	return lower, lowerCaseParameters[lower]
}

// GetSupportedParameters returns the name of the supported parameters
func GetSupportedParameters() []string {
	return util.GetSortedKeys(supportedParameterDescriptions)
}
