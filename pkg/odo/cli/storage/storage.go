package storage

import (
	"fmt"
	"github.com/openshift/odo/pkg/occlient"
	odoutil "github.com/openshift/odo/pkg/odo/util"
	"github.com/openshift/odo/pkg/storage"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	ktemplates "k8s.io/kubernetes/pkg/kubectl/cmd/templates"
)

// RecommendedCommandName is the recommended command name
const RecommendedCommandName = "storage"

var (
	storageShortDesc = `Perform storage operations`
	storageLongDesc  = ktemplates.LongDesc(`Perform storage operations`)
)

// NewCmdStorage implements the odo storage command
func NewCmdStorage(name, fullName string) *cobra.Command {
	storageCreateCmd := NewCmdStorageCreate(createRecommendedCommandName, odoutil.GetFullName(fullName, createRecommendedCommandName))
	storageDeleteCmd := NewCmdStorageDelete(deleteRecommendedCommandName, odoutil.GetFullName(fullName, deleteRecommendedCommandName))
	storageListCmd := NewCmdStorageList(listRecommendedCommandName, odoutil.GetFullName(fullName, listRecommendedCommandName))
	//storageMountCmd := NewCmdStorageMount(mountRecommendedCommandName, odoutil.GetFullName(fullName, mountRecommendedCommandName))
	//storageUnMountCmd := NewCmdStorageUnMount(unMountRecommendedCommandName, odoutil.GetFullName(fullName, unMountRecommendedCommandName))

	var storageCmd = &cobra.Command{
		Use:   name,
		Short: storageShortDesc,
		Long:  storageLongDesc,
		Example: fmt.Sprintf("%s\n%s\n%s",
			storageCreateCmd.Example,
			storageDeleteCmd.Example,
			//storageUnMountCmd.Example,
			storageListCmd.Example),
	}

	storageCmd.AddCommand(storageCreateCmd)
	storageCmd.AddCommand(storageDeleteCmd)
	//storageCmd.AddCommand(storageUnMountCmd)
	storageCmd.AddCommand(storageListCmd)
	//storageCmd.AddCommand(storageMountCmd)

	// Add a defined annotation in order to appear in the help menu
	storageCmd.Annotations = map[string]string{"command": "other"}
	storageCmd.SetUsageTemplate(odoutil.CmdUsageTemplate)

	return storageCmd
}

// validateStoragePath will validate storagePath, if there is any existing storage with similar path, it will give an error
func validateStoragePath(client *occlient.Client, storagePath, componentName, applicationName string) error {
	storeList, err := storage.List(client, componentName, applicationName)
	if err != nil {
		return err
	}
	for _, store := range storeList.Items {
		if store.Status.Path == storagePath {
			return errors.Errorf("there already is a storage mounted at %s", storagePath)
		}
	}
	return nil
}
