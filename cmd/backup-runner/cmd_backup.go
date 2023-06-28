package main

import (
	"github.com/spf13/cobra"
)

var cmdBackup = &cobra.Command{
	Use:   "backup",
	Short: "Starts the backup routine",
	RunE:  cmdBackupRunE,
}

func init() {
	cmdRoot.AddCommand(cmdBackup)
}

func cmdBackupRunE(cmd *cobra.Command, _ []string) (err error) {
	return triggerIPCRequest(cmd, ipcPayload{
		Action: "backup",
		Args:   nil,
	})
}
