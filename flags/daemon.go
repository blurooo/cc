package flags

import (
	"github.com/spf13/cobra"

	"github.com/blurooo/cc/tc"
)

var daemonCommand = &cobra.Command{
	Use:    "daemon",
	Short:  "启用常驻进程",
	Hidden: true,
	RunE:   commandDaemon,
}

func commandDaemon(_ *cobra.Command, _ []string) error {
	return tc.StartDaemon()
}

func registerDaemonCmd() {
	addToRootCmd(daemonCommand)
}
