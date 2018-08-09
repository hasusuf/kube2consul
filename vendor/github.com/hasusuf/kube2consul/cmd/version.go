package cmd

import (
	"github.com/spf13/cobra"
	"fmt"
)

func NewCmdVersion() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the application version",
		Run: func(cmd *cobra.Command, args []string) {
			runVersion()
		},
	}

	return cmd
}

func runVersion() {
	versionNumber := "VERSION_NUMBER"
	fmt.Printf("Version: %#v\n", versionNumber)
}
