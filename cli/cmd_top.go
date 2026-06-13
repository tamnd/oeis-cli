package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) topCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "top",
		Short: "List popular and core sequences (keyword:nice,core)",
		Example: `  oeis top
  oeis top -n 20`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			n := a.effectiveLimit(10)
			a.progressf("fetching popular sequences...")
			seqs, err := a.client.Top(cmd.Context(), n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(seqs, len(seqs))
		},
	}
	return cmd
}
