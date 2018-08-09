package cmd

import (
	"github.com/golang/glog"
	"github.com/hasusuf/kube2consul/templates"
	"github.com/hasusuf/kube2consul/util/i18n"
	"github.com/spf13/cobra"
)

var (
	debugFlag bool
	baseCmd   *cobra.Command
)

func NewKube2ConsulCommand() *cobra.Command {
	cmds := &cobra.Command{
		Use:   "kube2consul",
		Short: "Mirror your Kubernetes secrets/configMaps to Consul",
		Long:  templates.LongDesc(i18n.T(`Mirror your Kubernetes secrets/configMaps to Consul`)),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return setSyncKube2ConsulRequiredFlags(cmd)
		},
		Run: runHelp,
	}

	cmds.PersistentFlags().BoolVarP(
		&debugFlag,
		"debug",
		"",
		false,
		"Turn on debug logging.")

	cmds.AddCommand(NewCmdSync())
	cmds.AddCommand(NewCmdVersion())

	return cmds
}

func Execute() {
	baseCmd = NewKube2ConsulCommand()
	err := baseCmd.Execute()
	errorHandler(err)
}

func runHelp(cmd *cobra.Command, args []string) {
	cmd.Help()
}

func errorHandler(err error) {
	if err != nil && debugFlag {
		glog.Fatalf("something went wrong: %v", err)
	}
}
