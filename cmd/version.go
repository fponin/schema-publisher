package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print hpub version",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Println("hpub version " + Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
