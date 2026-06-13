package cli

import (
	"github.com/spf13/cobra"
	"github.com/tamnd/acmqueue-cli/acmqueue"
)

func (a *App) topicsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "topics",
		Short: "List ACM Queue topic areas",
		RunE: func(cmd *cobra.Command, _ []string) error {
			topics := acmqueue.Topics()
			return a.renderOrEmpty(topics, len(topics))
		},
	}
}
