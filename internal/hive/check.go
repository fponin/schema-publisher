package hive

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/fponin/hpub/internal/ui"
)

// ErrInvalidToken is returned when hive reports an invalid/expired token.
var ErrInvalidToken = errors.New("invalid token")

// HiveCreds holds authentication info for the Hive registry.
// If ConfigPath is set, it takes priority (legacy --config flag).
// Otherwise Endpoint + AccessToken are passed as --registry.* flags.
type HiveCreds struct {
	ConfigPath  string
	Endpoint    string
	AccessToken string
}

func (c HiveCreds) args() []string {
	if c.ConfigPath != "" {
		return []string{"--config", c.ConfigPath}
	}
	return []string{"--registry.endpoint", c.Endpoint, "--registry.accessToken", c.AccessToken}
}

// CheckResult holds the outcome of a hive schema:check invocation.
type CheckResult struct {
	OK     bool
	Output string
}

// Check runs `hive schema:check` for the given service and schema file.
// It displays the full output to the user regardless of outcome.
func Check(ctx context.Context, creds HiveCreds, service, schemaFile string, verbose bool) (CheckResult, error) {
	args := []string{"schema:check"}
	args = append(args, creds.args()...)
	args = append(args, "--service", service, schemaFile)

	cmd := exec.CommandContext(ctx, "hive", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	combined := stdout.String()
	if verbose && stderr.Len() > 0 {
		combined += "\n[stderr]\n" + stderr.String()
	} else if stderr.Len() > 0 {
		combined += stderr.String()
	}

	// Always show the hive output
	if combined != "" {
		ui.Println(combined)
	}

	if strings.Contains(combined, "Invalid token provided") {
		return CheckResult{OK: false, Output: combined}, ErrInvalidToken
	}

	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			// hive завершился с ненулевым кодом — нашёл проблемы, не упал
			return CheckResult{OK: false, Output: combined}, nil
		}
		return CheckResult{OK: false, Output: combined}, fmt.Errorf("hive schema:check failed: %w", err)
	}

	return CheckResult{OK: true, Output: combined}, nil
}
