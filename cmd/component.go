package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/redhat-developer/ocdev/pkg/component"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	componentBinary    string
	componentGit       string
	componentDir       string
	componentShortFlag bool
)

// componentCmd represents the component command
var componentCmd = &cobra.Command{
	Use:   "component",
	Short: "components of application",
	Long:  "components of application",
	// 'ocdev component' is the same as 'ocdev component get'
	Run: componentGetCmd.Run,
}

var componentCreateCmd = &cobra.Command{
	Use:     "create",
	Short:   "create a new component",
	Long:    "create a new component",
	Example: "ocdev component create <component type> <component name, optional>",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("no component type has been provided")
		}
		if len(args) > 2 {
			return fmt.Errorf("extra arguments provided, accepted maximum 2: component create <type> <name>")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		log.Debugf("component create called")
		log.Debugf("args: %#v", strings.Join(args, " "))
		log.Debugf("flags: binary=%s, git=%s, dir=%s", componentBinary, componentGit, componentDir)

		if len(componentBinary) != 0 {
			fmt.Printf("--binary is not implemented yet\n\n")
			cmd.Help()
			os.Exit(1)
		}

		//TODO: check flags - only one of binary, git, dir can be specified

		//We don't have to check it anymore, Args check made sure that args has at least one item
		// and no more than two
		componentType := args[0]
		if component.ValidateType(componentType) != nil {
			fmt.Printf("Unsupported component type: %v\n", componentType)
			os.Exit(-1)
		}

		componentName := args[0]
		if len(args) == 2 {
			componentName = args[1]
		}

		if len(componentBinary) != 0 {
			fmt.Printf("--binary is not implemented yet\n\n")
			os.Exit(1)
		}

		if len(componentGit) != 0 {
			output, err := component.CreateFromGit(componentName, componentType, componentGit)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			fmt.Println(output)
		} else if len(componentDir) != 0 {
			output, err := component.CreateFromDir(componentName, componentType, componentDir)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			fmt.Println(output)
		} else {
			// no flag was set, create empty component
			output, err := component.CreateEmpty(componentName, componentType)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			fmt.Println(output)
		}

	},
}

var componentDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "component delete <component_name>",
	Long:  "delete existing component",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("Please specify component name")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		log.Debugf("component delete called")
		log.Debugf("args: %#v", strings.Join(args, " "))

		componentName := args[0]

		// no flag was set, create empty component
		output, err := component.Delete(componentName)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println(output)

	},
}

var componentGetCmd = &cobra.Command{
	Use:   "get",
	Short: "component get",
	Long:  "get current component",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debugf("component get called")

		component, err := component.GetCurrent()
		if err != nil {
			fmt.Println(errors.Wrap(err, "unable to get current component"))
			os.Exit(1)
		}
		if componentShortFlag {
			fmt.Print(component)
		} else {
			fmt.Printf("The current component is: %v\n", component)
		}
	},
}

var componentSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set component as active.",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("Please provide component name")
		}
		if len(args) > 1 {
			return fmt.Errorf("Only one argument (component name) is allowed")
		}
		return nil
	}, Run: func(cmd *cobra.Command, args []string) {
		err := component.SetCurrent(args[0])
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Printf("Switched to component: %v\n", args[0])
	},
}

func init() {
	componentCreateCmd.Flags().StringVar(&componentBinary, "binary", "", "binary artifact")
	componentCreateCmd.Flags().StringVar(&componentGit, "git", "", "git source")
	componentCreateCmd.Flags().StringVar(&componentDir, "dir", "", "local directory as source")

	componentGetCmd.Flags().BoolVarP(&componentShortFlag, "short", "q", false, "If true, display only the component name")
	// add flags from 'get' to component command
	componentCmd.Flags().AddFlagSet(applicationGetCmd.Flags())

	componentCmd.AddCommand(componentDeleteCmd)
	componentCmd.AddCommand(componentGetCmd)
	componentCmd.AddCommand(componentCreateCmd)
	componentCmd.AddCommand(componentSetCmd)

	rootCmd.AddCommand(componentCmd)
}
