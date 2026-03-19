package hive

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"

	"github.com/fponin/hpub/internal/ui"
)

// Publish runs `hive schema:publish` for the given service and schema file.
func Publish(ctx context.Context, creds HiveCreds, service, publishURL, schemaFile, author, commit string, verbose bool) error {
	args := []string{"schema:publish"}
	args = append(args, creds.args()...)
	args = append(args,
		"--service", service,
		"--url", publishURL,
		"--author", author,
		"--commit", commit,
		schemaFile,
	)

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

	if combined != "" {
		ui.Println(combined)
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("hive schema:publish exited with code %d", exitErr.ExitCode())
		}
		return fmt.Errorf("hive schema:publish failed: %w", err)
	}

	return nil
}
