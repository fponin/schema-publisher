package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/fponin/hpub/internal/config"
	"github.com/fponin/hpub/internal/hive"
	"github.com/fponin/hpub/internal/orchestrator"
	"github.com/fponin/hpub/internal/preflight"
	"github.com/fponin/hpub/internal/state"
	"github.com/fponin/hpub/internal/ui"
	"github.com/fponin/hpub/internal/wizard"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Full wizard + pipeline (introspect, check, publish)",
	RunE:  runRun,
}

func init() {
	rootCmd.AddCommand(runCmd)
}

// Steps for the interactive wizard flow.
const (
	wsEnv       = 0 // env selection + hive credential setup
	wsScenario  = 1 // scenario selection
	wsFileOrCtx = 2 // schema file path OR kubectl context
	wsWizard    = 3 // subgraph/params wizard + pipeline
	wsDone      = 4
)

func runRun(cmd *cobra.Command, _ []string) error {
	if isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd()) {
		fmt.Print("\033[H\033[2J")
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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	schemaFlagSet := flagSchema != ""

	var (
		env              string
		profile          config.EnvProfile
		schemaFile       = flagSchema
		originalScenario string // what SelectScenario returned
		scenario         string // may be flipped to full by SelectSchemaFileWithValidation
		input            wizard.Input
	)

	cur := wsEnv
	for cur < wsDone {
		switch cur {

		case wsEnv:
			env, err = wizard.SelectEnv(cfg, flagEnv)
			if wizard.IsBack(err) || wizard.IsExit(err) {
				return nil
			}
			if err != nil {
				return fmt.Errorf("selecting env: %w", err)
			}
			profile, err = cfg.ResolveProfile(env)
			if err != nil {
				return err
			}
			profile, err = wizard.SetupHiveCredentials(cfg, cfgPath, env, profile)
			if wizard.IsBack(err) {
				continue // re-show env selection
			}
			if wizard.IsExit(err) {
				return nil
			}
			if err != nil {
				return fmt.Errorf("hive credentials setup: %w", err)
			}
			if schemaFlagSet {
				cur = wsWizard
			} else {
				cur = wsScenario
			}

		case wsScenario:
			originalScenario, err = wizard.SelectScenario()
			if wizard.IsBack(err) {
				cur = wsEnv
				continue
			}
			if wizard.IsExit(err) {
				return nil
			}
			if err != nil {
				return fmt.Errorf("selecting scenario: %w", err)
			}
			scenario = originalScenario
			cur = wsFileOrCtx

		case wsFileOrCtx:
			// Reset schemaFile so a re-entry always produces a fresh value.
			schemaFile = ""

			if originalScenario == wizard.ScenarioSchemaOnly {
				var resolvedScenario string
				schemaFile, resolvedScenario, err = wizard.SelectSchemaFileWithValidation(st, cfg)
				if wizard.IsBack(err) {
					cur = wsScenario
					continue
				}
				if wizard.IsExit(err) {
					return nil
				}
				if err != nil {
					return fmt.Errorf("selecting schema file: %w", err)
				}
				scenario = resolvedScenario
			}

			// Full pipeline needs a kubectl context.
			if scenario == wizard.ScenarioFull && profile.KubectlContext == "" {
				profile, err = wizard.SelectKubectlContext(cfg, cfgPath, env, profile)
				if wizard.IsBack(err) {
					cur = wsScenario
					continue
				}
				if wizard.IsExit(err) {
					return nil
				}
				if err != nil {
					return fmt.Errorf("kubectl context setup: %w", err)
				}
			}
			cur = wsWizard

		case wsWizard:
			ui.PrintEnvBanner(env)

			mode := preflight.FullMode
			if schemaFile != "" {
				mode = preflight.SchemaOnlyMode
			}
			pfResult := preflight.Run(preflight.Options{
				Mode:            mode,
				HiveConfigPath:  profile.HiveConfigPath,
				HiveEndpoint:    profile.HiveEndpoint,
				HiveAccessToken: profile.HiveAccessToken,
				KubectlContext:  profile.KubectlContext,
				SchemaFile:      schemaFile,
			})
			if !pfResult.OK() {
				ui.Error("Preflight checks failed:")
				for _, e := range pfResult.Errors {
					ui.Error("  • " + e.Error())
				}
				return fmt.Errorf("preflight failed")
			}
			for _, w := range pfResult.Warnings {
				ui.Warn("  ⚠  " + w)
			}
			ui.StepOK("preflight checks passed")

			if schemaFile != "" {
				input, err = wizard.RunSchemaOnly(cfg, profile, st, env)
			} else {
				input, err = wizard.Run(cfg, profile, st, env)
			}
			if wizard.IsBack(err) {
				if schemaFlagSet {
					cur = wsEnv
				} else {
					cur = wsFileOrCtx
				}
				continue
			}
			if wizard.IsExit(err) {
				return nil
			}
			if err != nil {
				return fmt.Errorf("wizard: %w", err)
			}

			// Run the pipeline.
			if schemaFile != "" {
				result, err := orchestrator.RunSchemaCheck(ctx, input, profile, schemaFile, flagVerbose)
				if err != nil {
					if handleInvalidToken(err, cfgPath, env, profile) {
						return nil
					}
					return err
				}
				if !result.CheckResult.OK {
					if !confirmPublishDespiteErrors() {
						ui.Info("Публикация отменена.")
						return nil
					}
				}
			} else {
				result, err := orchestrator.RunFull(ctx, input, profile, flagVerbose)
				if err != nil {
					if handleInvalidToken(err, cfgPath, env, profile) {
						return nil
					}
					return err
				}
				schemaFile = result.SchemaFile
				if !result.CheckResult.OK {
					if !confirmPublishDespiteErrors() {
						ui.Info("Публикация отменена.")
						return nil
					}
				}
			}

			if err := askAndPublish(ctx, input, profile, st, schemaFile, env); err != nil {
				return err
			}

			cur = wsDone
		}
	}

	return nil
}

func confirmPublishDespiteErrors() bool {
	fmt.Println()
	msg := "  Опубликовать схему несмотря на ошибки проверки? (y/N): "
	if !flagNoColor {
		msg = ui.StepFailStyle.Render(msg)
	}
	fmt.Print(msg)
	var ans string
	fmt.Scanln(&ans)
	return ans == "y" || ans == "Y"
}

// handleInvalidToken checks if err is ErrInvalidToken, clears saved credentials,
// prints an actionable message, and returns true so the caller can exit cleanly.
func handleInvalidToken(err error, cfgPath, env string, profile config.EnvProfile) bool {
	if !errors.Is(err, hive.ErrInvalidToken) {
		return false
	}
	profile.HiveAccessToken = ""
	profile.HiveEndpoint = ""
	profile.HiveConfigPath = ""
	if updateErr := config.UpdateEnvProfile(cfgPath, env, profile); updateErr != nil {
		ui.Warn("Could not clear invalid token from config: " + updateErr.Error())
	} else {
		ui.Error("Invalid token — saved credentials cleared. Re-run hpub to enter a new token.")
	}
	return true
}

func askAndPublish(ctx context.Context, input wizard.Input, profile config.EnvProfile, st *state.State, schemaFile, env string) error {
	// Prod safeguard
	if env == "prod" {
		if err := wizard.ProdSafeguard(); err != nil {
			ui.Info("Publish cancelled.")
			return nil
		}
	} else {
		// Ask to publish for non-prod
		var confirm bool
		fmt.Println()
		fmt.Print(ui.EnvStyle(env)("  Publish schema to "+strings.ToUpper(env)+"?")+" (y/N): ")
		var input2 string
		fmt.Scanln(&input2)
		if input2 == "y" || input2 == "Y" {
			confirm = true
		}
		if !confirm {
			ui.Info("Publish skipped.")
			return nil
		}
	}

	if err := wizard.AskPublishURL(&input); err != nil {
		return fmt.Errorf("publish url: %w", err)
	}

	author, commit, err := wizard.CollectPublishInfo(st)
	if err != nil {
		return fmt.Errorf("collecting publish info: %w", err)
	}

	if err := orchestrator.RunPublish(ctx, input, profile, schemaFile, author, commit, flagVerbose); err != nil {
		return err
	}

	// Save state
	st.LastAuthor = author
	st.AddCommitMessage(commit)
	st.AddService(input.Service)
	st.AddOutputFile(input.OutputFile)
	st.LastUsed = state.LastUsed{
		Env:        env,
		Service:    input.Service,
		Namespace:  input.Namespace,
		LocalPort:  input.LocalPort,
		RemotePort: input.RemotePort,
		OutputFile: input.OutputFile,
	}

	if err := state.Save(state.DefaultStatePath(), st); err != nil {
		ui.Warn("Failed to save state: " + err.Error())
	}

	ui.Println("\n  Schema published successfully.")
	return nil
}
