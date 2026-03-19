package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/fponin/hpub/internal/config"
	"github.com/fponin/hpub/internal/orchestrator"
	"github.com/fponin/hpub/internal/preflight"
	"github.com/fponin/hpub/internal/state"
	"github.com/fponin/hpub/internal/ui"
	"github.com/fponin/hpub/internal/wizard"
)

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Run hive schema:publish from an existing schema file",
	RunE:  runPublish,
}

var (
	publishFlagService string
	publishFlagURL     string
	publishFlagAuthor  string
	publishFlagCommit  string
)

func init() {
	publishCmd.Flags().StringVar(&publishFlagService, "service", "", "subgraph service name")
	publishCmd.Flags().StringVar(&publishFlagURL, "url", "", "override publish URL")
	publishCmd.Flags().StringVar(&publishFlagAuthor, "author", "", "author name")
	publishCmd.Flags().StringVar(&publishFlagCommit, "commit", "", "commit message")
	rootCmd.AddCommand(publishCmd)
}

func runPublish(_ *cobra.Command, _ []string) error {
	if flagSchema == "" {
		return fmt.Errorf("--schema is required for hpub publish")
	}

	ui.SetNoColor(flagNoColor)

	cfgPath := flagConfig
	if cfgPath == "" {
		cfgPath = config.DefaultConfigPath()
	}

	cfg, err := config.LoadOrDefault(cfgPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	st, err := state.Load(state.DefaultStatePath())
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	env, err := wizard.SelectEnv(cfg, flagEnv)
	if err != nil {
		return err
	}

	profile, err := cfg.ResolveProfile(env)
	if err != nil {
		return err
	}

	// Setup hive credentials if not configured
	profile, err = wizard.SetupHiveCredentials(cfg, cfgPath, env, profile)
	if err != nil {
		return fmt.Errorf("hive credentials setup: %w", err)
	}

	ui.PrintEnvBanner(env)

	pfResult := preflight.Run(preflight.Options{
		Mode:            preflight.SchemaOnlyMode,
		HiveConfigPath:  profile.HiveConfigPath,
		HiveEndpoint:    profile.HiveEndpoint,
		HiveAccessToken: profile.HiveAccessToken,
		SchemaFile:      flagSchema,
	})
	if !pfResult.OK() {
		for _, e := range pfResult.Errors {
			ui.Error(e.Error())
		}
		return fmt.Errorf("preflight failed")
	}

	// Determine service
	var input wizard.Input
	if publishFlagService == "" {
		input, err = wizard.RunSchemaOnly(cfg, profile, st, env)
		if err != nil {
			return err
		}
	} else {
		input = wizard.Input{Env: env, Service: publishFlagService}
		if sg, ok := cfg.FindSubgraph(publishFlagService); ok {
			input.PublishURL = sg.PublishURL
		}
	}

	if publishFlagURL != "" {
		input.PublishURL = publishFlagURL
	}

	if input.PublishURL == "" {
		return fmt.Errorf("publish URL not set (use --url or add service to subgraphs registry)")
	}

	if _, err := os.Stat(flagSchema); os.IsNotExist(err) {
		return fmt.Errorf("schema file not found: %s", flagSchema)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Prod safeguard
	if env == "prod" {
		if err := wizard.ProdSafeguard(); err != nil {
			ui.Info("Publish cancelled.")
			return nil
		}
	}

	// Collect author/commit if not provided via flags
	author := publishFlagAuthor
	commit := publishFlagCommit
	if author == "" || commit == "" {
		a, c, err := wizard.CollectPublishInfo(st)
		if err != nil {
			return err
		}
		if author == "" {
			author = a
		}
		if commit == "" {
			commit = c
		}
	}

	if err := orchestrator.RunPublish(ctx, input, profile, flagSchema, author, commit, flagVerbose); err != nil {
		return err
	}

	// Save state
	st.LastAuthor = author
	st.AddCommitMessage(commit)
	st.AddService(input.Service)
	st.LastUsed = state.LastUsed{
		Env:     env,
		Service: input.Service,
	}

	if err := state.Save(state.DefaultStatePath(), st); err != nil {
		ui.Warn("Failed to save state: " + err.Error())
	}

	ui.Println("\n  Schema published successfully.")
	return nil
}
