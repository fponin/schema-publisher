package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	flagEnv     string
	flagConfig  string
	flagNoColor bool
	flagVerbose bool
	flagSchema  string
)

var rootCmd = &cobra.Command{
	Use:   "hpub",
	Short: "HivePublisher — automate GraphQL subgraph schema pipeline",
	Long: `HivePublisher (hpub) automates the full GraphQL subgraph schema publishing
pipeline: token fetch → port-forward → introspect → hive check → hive publish.`,
	RunE: runRun,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagEnv, "env", "", "environment (dev/stage/prod)")
	rootCmd.PersistentFlags().StringVar(&flagConfig, "config", "", "path to config file (default: ~/.config/hpub/config.yaml)")
	rootCmd.PersistentFlags().BoolVar(&flagNoColor, "no-color", false, "disable terminal colors")
	rootCmd.PersistentFlags().BoolVar(&flagVerbose, "verbose", false, "show full subprocess output")
	rootCmd.PersistentFlags().StringVar(&flagSchema, "schema", "", "use existing schema file instead of introspecting")
}
