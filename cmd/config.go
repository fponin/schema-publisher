package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/fponin/hpub/internal/config"
	"github.com/fponin/hpub/internal/ui"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage hpub configuration",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Print current configuration",
	RunE: func(_ *cobra.Command, _ []string) error {
		cfgPath := flagConfig
		if cfgPath == "" {
			cfgPath = config.DefaultConfigPath()
		}
		cfg, err := config.Load(cfgPath)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		data, err := yaml.Marshal(cfg)
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	},
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Open config in $EDITOR",
	RunE: func(_ *cobra.Command, _ []string) error {
		cfgPath := flagConfig
		if cfgPath == "" {
			cfgPath = config.DefaultConfigPath()
		}
		// Ensure it exists first
		if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
			ui.Info("Config file not found, creating default...")
			def := config.DefaultConfig()
			if err := config.Save(cfgPath, &def); err != nil {
				return err
			}
		}
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}
		cmd := exec.Command(editor, cfgPath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create default config if it does not exist",
	RunE: func(_ *cobra.Command, _ []string) error {
		cfgPath := flagConfig
		if cfgPath == "" {
			cfgPath = config.DefaultConfigPath()
		}

		if _, err := os.Stat(cfgPath); err == nil {
			ui.Info("Config already exists at " + cfgPath)
			return nil
		}

		def := config.DefaultConfig()
		if err := config.Save(cfgPath, &def); err != nil {
			return fmt.Errorf("saving default config: %w", err)
		}
		ui.Info("Config written to " + cfgPath)
		ui.Warn("Default config contains placeholder values — edit before use: hpub config edit")
		return nil
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configInitCmd)
	rootCmd.AddCommand(configCmd)
}
