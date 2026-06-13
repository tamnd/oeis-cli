package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) searchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search sequences by name, keywords, or values",
		Example: `  oeis search "fibonacci"
  oeis search "1 1 2 3 5 8"
  oeis search "prime" -n 5`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			n := a.effectiveLimit(10)
			query := args[0]
			a.progressf("searching for %q...", query)
			seqs, err := a.client.Search(cmd.Context(), query, n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(seqs, len(seqs))
		},
	}
	return cmd
}
