package main

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const flagRestoreMode = "mode"

var cmdRestore = &cobra.Command{
	Use:   "restore identifier",
	Short: "Restores to the given point in time",
	RunE:  cmdRestoreRunE,
}

func init() {
	cmdRestore.Flags().String(flagRestoreMode, "point-in-time", "restore-mode to use (point-in-time / name)")
	cmdRoot.AddCommand(cmdRestore)
}

func cmdRestoreRunE(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return errors.New("expected identifier for backup to restore")
	}

	restoreMode, err := cmd.Flags().GetString(flagRestoreMode)
	if err != nil {
		return errors.Wrapf(err, "getting %s flag value", flagRestoreMode)
	}

	return triggerIPCRequest(cmd, ipcPayload{
		Action: "restore",
		Args:   []string{restoreMode, args[0]},
	})
}
