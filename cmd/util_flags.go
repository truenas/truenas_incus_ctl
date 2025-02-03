package cmd

import (
	"strings"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func getCobraFlags(cmd *cobra.Command) (usedFlags, allFlags, allTypes map[string]string) {
	usedFlags = make(map[string]string)
	cmd.Flags().Visit(func(flag *pflag.Flag) {
		key := strings.ReplaceAll(flag.Name, "-", "_")
		usedFlags[key] = flag.Value.String()
	})

	allFlags = make(map[string]string)
	allTypes = make(map[string]string)
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		key := strings.ReplaceAll(flag.Name, "-", "_")
		allFlags[key] = flag.Value.String()
		allTypes[key] = flag.Value.Type()
	})

	RemoveGlobalFlags(usedFlags)
	RemoveGlobalFlags(allFlags)
	RemoveGlobalFlags(allTypes)
	return usedFlags, allFlags, allTypes
}
