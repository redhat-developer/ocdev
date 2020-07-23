package parser

import (
	"encoding/json"
	devfileCtx "github.com/openshift/odo/pkg/devfile/parser/context"
	"github.com/openshift/odo/pkg/devfile/parser/data"

	"github.com/openshift/odo/pkg/devfile/parser/data/common"
	"github.com/pkg/errors"
	"k8s.io/klog"
	"reflect"
)

// ParseDevfile func validates the devfile integrity.
// Creates devfile context and runtime objects
func parseDevfile(d DevfileObj) (DevfileObj, error) {

	// Validate devfile
	err := d.Ctx.Validate()
	if err != nil {
		return d, err
	}

	// Create a new devfile data object
	d.Data, err = data.NewDevfileData(d.Ctx.GetApiVersion())
	if err != nil {
		return d, err
	}

	// Unmarshal devfile content into devfile struct
	err = json.Unmarshal(d.Ctx.GetDevfileContent(), &d.Data)
	if err != nil {
		return d, errors.Wrapf(err, "failed to decode devfile content")
	}

	if !reflect.DeepEqual(d.Data.GetParent(), common.DevfileParent{}) && d.Data.GetParent().Uri != "" {
		err = parseParent(d)
		if err != nil {
			return DevfileObj{}, err
		}
	}

	// Successful
	return d, nil
}

// Parse func populates the devfile data, parses and validates the devfile integrity.
// Creates devfile context and runtime objects
func Parse(path string) (d DevfileObj, err error) {

	// NewDevfileCtx
	d.Ctx = devfileCtx.NewDevfileCtx(path)

	// Fill the fields of DevfileCtx struct
	err = d.Ctx.Populate()
	if err != nil {
		return d, err
	}
	return parseDevfile(d)
}

// ParseFromURL func parses and validates the devfile integrity.
// Creates devfile context and runtime objects
func ParseFromURL(url string) (d DevfileObj, err error) {
	d.Ctx = devfileCtx.NewURLDevfileCtx(url)

	// Fill the fields of DevfileCtx struct
	err = d.Ctx.PopulateFromURL()
	if err != nil {
		return d, err
	}
	return parseDevfile(d)
}

func parseParent(d DevfileObj) error {
	parent := d.Data.GetParent()

	parentData, err := ParseFromURL(parent.Uri)
	if err != nil {
		return err
	}

	klog.V(4).Infof("overriding data of devfile with URI: %v", parent.Uri)

	// override the parent's components, commands, projects and events
	err = parentData.OverrideComponents(d.Data.GetParent().Components)
	if err != nil {
		return err
	}

	err = parentData.OverrideCommands(d.Data.GetParent().Commands)
	if err != nil {
		return err
	}

	err = parentData.OverrideEvents(d.Data.GetParent().Events)
	if err != nil {
		return err
	}

	err = parentData.OverrideProjects(d.Data.GetParent().Projects)
	if err != nil {
		return err
	}

	klog.V(4).Infof("adding data of devfile with URI: %v", parent.Uri)

	// since the parent's data has been overriden
	// add the items back to the current devfile
	// error indicates that the item has been defined again in the current devfile
	err = d.Data.AddCommands(parentData.Data.GetCommands())
	if err != nil {
		return errors.Wrapf(err, "error while adding commands from the parent devfiles")
	}

	err = d.Data.AddComponents(parentData.Data.GetComponents())
	if err != nil {
		return errors.Wrapf(err, "error while adding components from the parent devfiles")
	}

	err = d.Data.AddProjects(parentData.Data.GetProjects())
	if err != nil {
		return errors.Wrapf(err, "error while adding projects from the parent devfiles")
	}

	err = d.Data.AddEvents(parentData.Data.GetEvents())
	if err != nil {
		return errors.Wrapf(err, "error while adding events from the parent devfiles")
	}
	return nil
}
