package cli

import (
	"github.com/spf13/cobra"
	"github.com/tamnd/acmqueue-cli/acmqueue"
)

func (a *App) articleCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "article <id-or-url>",
		Short: "Show a single ACM Queue article",
		Long: `Show a single ACM Queue article by its numeric ID or full URL.

The article is looked up in the current RSS feed (20 most recent articles).
If it is not in the feed, exit 3 is returned.

Examples:
  acmq article 3807964
  acmq article https://queue.acm.org/detail.cfm?ref=rss&id=3807964`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := acmqueue.ParseArticleID(args[0])
			if err != nil {
				return codeError(exitUsage, err)
			}
			a.progressf("fetching article %s...", id)
			art, err := a.client.Article(cmd.Context(), id)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.render([]acmqueue.Article{art})
		},
	}
}
