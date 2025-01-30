package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func getCobraFlags(cmd *cobra.Command) (usedFlags, allFlags map[string]string) {
	usedFlags = make(map[string]string)
	cmd.Flags().Visit(func(flag *pflag.Flag) {
		usedFlags[flag.Name] = flag.Value.String()
	})
	allFlags = make(map[string]string)
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		allFlags[flag.Name] = flag.Value.String()
	})

	RemoveGlobalFlags(usedFlags)
	RemoveGlobalFlags(allFlags)
	return usedFlags, allFlags
}
