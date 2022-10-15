package flags

import (
	"github.com/blurooo/cc/config"
	"github.com/spf13/cobra"
)

const DaemonName = "__daemon"

func GetDaemonCommand(app config.Application) *cobra.Command {
	return &cobra.Command{
		Use:    DaemonName,
		Short:  "启用常驻进程",
		Hidden: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return app.Handler.OnDaemon(cmd)
		},
	}
}
