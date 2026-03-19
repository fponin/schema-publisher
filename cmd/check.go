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

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Run hive schema:check against an existing schema file",
	RunE:  runCheck,
}

var checkFlagService string

func init() {
	checkCmd.Flags().StringVar(&checkFlagService, "service", "", "subgraph service name")
	rootCmd.AddCommand(checkCmd)
}

func runCheck(_ *cobra.Command, _ []string) error {
	if flagSchema == "" {
		return fmt.Errorf("--schema is required for hpub check")
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

	service := checkFlagService
	if service == "" {
		input, err := wizard.RunSchemaOnly(cfg, profile, st, env)
		if err != nil {
			return err
		}
		service = input.Service
	}

	if _, err := os.Stat(flagSchema); os.IsNotExist(err) {
		return fmt.Errorf("schema file not found: %s", flagSchema)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	input := wizard.Input{Env: env, Service: service}
	result, err := orchestrator.RunSchemaCheck(ctx, input, profile, flagSchema, flagVerbose)
	if err != nil {
		return err
	}

	if result.CheckResult.OK {
		ui.StepOK("Schema check passed")

		// Save state on successful check
		st.AddService(service)
		_ = state.Save(state.DefaultStatePath(), st)
	}
	return nil
}
