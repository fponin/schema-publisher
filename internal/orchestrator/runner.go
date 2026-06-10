package orchestrator

import (
	"context"
	"fmt"
	"os"

	"github.com/fponin/hpub/internal/config"
	"github.com/fponin/hpub/internal/hive"
	"github.com/fponin/hpub/internal/portforward"
	"github.com/fponin/hpub/internal/rover"
	"github.com/fponin/hpub/internal/token"
	"github.com/fponin/hpub/internal/ui"
	"github.com/fponin/hpub/internal/wizard"
)

// PipelineResult holds the outcome of a full pipeline run.
type PipelineResult struct {
	CheckResult hive.CheckResult
	SchemaFile  string
}

// RunFull executes the full introspection + check pipeline.
func RunFull(ctx context.Context, input wizard.Input, profile config.EnvProfile, verbose bool) (PipelineResult, error) {
	// Step 1: Fetch token
	ui.StartStep("Fetching auth token")
	tok, err := token.Fetch(ctx, profile.AuthURL, profile.AuthBearerToken)
	if err != nil {
		ui.StepFail(err.Error())
		return PipelineResult{}, fmt.Errorf("token: %w", err)
	}

	// Step 2: Start port-forward
	ui.StartStep("Starting port-forward")
	pf := portforward.New(profile.KubectlContext, input.K8sResource, input.Namespace, input.LocalPort, input.RemotePort)
	if err := pf.Start(ctx); err != nil {
		ui.StepFail(err.Error())
		return PipelineResult{}, fmt.Errorf("port-forward: %w", err)
	}
	defer func() {
		pf.Stop()
		ui.Info("Port-forward stopped.")
	}()
	ui.StepOK(fmt.Sprintf("%s → localhost:%d", input.K8sResource, input.LocalPort))

	// Step 3: Wait for endpoint readiness
	ui.StartStep("Waiting for endpoint readiness")
	if err := pf.WaitReady(ctx, input.GraphQLPath); err != nil {
		ui.StepFail(err.Error())
		return PipelineResult{}, fmt.Errorf("readiness: %w", err)
	}
	ui.StepOK()

	// Step 4: Run rover introspect
	ui.StartStep("Running rover introspect")
	schemaBytes, err := rover.Introspect(ctx, input.LocalPort, input.GraphQLPath, profile.JWTHeader, tok, verbose)
	if err != nil {
		ui.StepFail(err.Error())
		return PipelineResult{}, fmt.Errorf("introspect: %w", err)
	}

	outputFile := expandTilde(input.OutputFile)
	if err := os.WriteFile(outputFile, schemaBytes, 0o644); err != nil {
		ui.StepFail(err.Error())
		return PipelineResult{}, fmt.Errorf("writing schema: %w", err)
	}
	ui.StepOK(fmt.Sprintf("saved to %s (%d bytes)", input.OutputFile, len(schemaBytes)))

	// Step 5: Run hive schema:check
	return runCheck(ctx, input, profile, outputFile, verbose)
}

// RunSchemaCheck runs only hive schema:check on an existing schema file.
func RunSchemaCheck(ctx context.Context, input wizard.Input, profile config.EnvProfile, schemaFile string, verbose bool) (PipelineResult, error) {
	return runCheck(ctx, input, profile, schemaFile, verbose)
}

func hiveCreds(profile config.EnvProfile) hive.HiveCreds {
	return hive.HiveCreds{
		ConfigPath:  profile.HiveConfigPath,
		Endpoint:    profile.HiveEndpoint,
		AccessToken: profile.HiveAccessToken,
	}
}

func runCheck(ctx context.Context, input wizard.Input, profile config.EnvProfile, schemaFile string, verbose bool) (PipelineResult, error) {
	ui.StartStep("Running hive schema:check")
	result, err := hive.Check(ctx, hiveCreds(profile), input.Service, schemaFile, verbose)
	if err != nil {
		ui.StepFail(err.Error())
		return PipelineResult{CheckResult: result, SchemaFile: schemaFile}, fmt.Errorf("check: %w", err)
	}
	if result.OK {
		ui.StepOK()
	} else {
		ui.StepWarn("найдены breaking changes")
	}
	return PipelineResult{CheckResult: result, SchemaFile: schemaFile}, nil
}

// RunPublish executes hive schema:publish.
func RunPublish(ctx context.Context, input wizard.Input, profile config.EnvProfile, schemaFile, author, commit string, verbose bool) error {
	ui.StartStep("Running hive schema:publish")
	if err := hive.Publish(ctx, hiveCreds(profile), input.Service, input.PublishURL, schemaFile, author, commit, verbose); err != nil {
		ui.StepFail(err.Error())
		return fmt.Errorf("publish: %w", err)
	}
	ui.StepOK()
	return nil
}

func expandTilde(path string) string {
	if len(path) >= 2 && path[:2] == "~/" {
		home, _ := os.UserHomeDir()
		return home + path[1:]
	}
	return path
}
