package ui

import (
	"fmt"
	"io"
	"os"
	"strings"
)

var noColor bool
var out io.Writer = os.Stdout

// SetNoColor disables terminal color output.
func SetNoColor(v bool) {
	noColor = v
}

// SetOutput redirects all UI output (useful in tests).
func SetOutput(w io.Writer) {
	out = w
}

// StartStep prints the beginning of a pipeline step.
func StartStep(name string) {
	msg := fmt.Sprintf("  ▶ %s...", name)
	if !noColor {
		msg = StepInfoStyle.Render(msg)
	}
	fmt.Fprintln(out, msg)
}

// StepOK prints a success indicator for the current step.
func StepOK(detail ...string) {
	suffix := ""
	if len(detail) > 0 && detail[0] != "" {
		suffix = " — " + detail[0]
	}
	msg := "    ✓ OK" + suffix
	if !noColor {
		msg = StepOKStyle.Render(msg)
	}
	fmt.Fprintln(out, msg)
}

// StepFail prints a failure indicator for the current step.
func StepFail(detail string) {
	msg := "    ✗ FAILED: " + detail
	if !noColor {
		msg = StepFailStyle.Render(msg)
	}
	fmt.Fprintln(out, msg)
}

// StepWarn prints a warning for the current step.
func StepWarn(detail string) {
	msg := "    ⚠ " + detail
	if !noColor {
		msg = StepWarnStyle.Render(msg)
	}
	fmt.Fprintln(out, msg)
}

// Info prints an informational message.
func Info(msg string) {
	fmt.Fprintln(out, "  "+msg)
}

// Warn prints a warning message.
func Warn(msg string) {
	line := "  ⚠  " + msg
	if !noColor {
		line = WarningStyle.Render(line)
	}
	fmt.Fprintln(out, line)
}

// Error prints an error message.
func Error(msg string) {
	line := "  ✗  " + msg
	if !noColor {
		line = StepFailStyle.Render(line)
	}
	fmt.Fprintln(out, line)
}

// EnvStyle renders a string with the foreground color matching the given environment.
func EnvStyle(env string) func(string) string {
	if noColor {
		return func(s string) string { return s }
	}
	switch env {
	case "prod":
		return func(s string) string { return EnvProdStyle.Render(s) }
	case "stage":
		return func(s string) string { return EnvStageStyle.Render(s) }
	default:
		return func(s string) string { return EnvDevStyle.Render(s) }
	}
}

// Divider prints a horizontal divider line.
func Divider() {
	fmt.Fprintln(out, strings.Repeat("─", 50))
}

// Println prints a plain line.
func Println(args ...interface{}) {
	fmt.Fprintln(out, args...)
}

// Printf prints a formatted line.
func Printf(format string, args ...interface{}) {
	fmt.Fprintf(out, format, args...)
}
