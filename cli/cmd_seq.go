package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tamnd/oeis-cli/oeis"
)

func (a *App) seqCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "seq <A-number>",
		Short: "Fetch a specific sequence by A-number",
		Example: `  oeis seq A000045
  oeis seq 45
  oeis seq a000079`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, err := oeis.ParseANumber(args[0])
			if err != nil {
				return codeError(exitUsage, fmt.Errorf("invalid A-number %q: %w", args[0], err))
			}
			seq, err := a.client.GetSeq(cmd.Context(), number)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.render(seq)
		},
	}
	return cmd
}
