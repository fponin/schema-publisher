package preflight

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

const (
	requiredHiveVersion = "0.42.1"
	hiveInstallCmd      = "npm install -g @graphql-hive/cli@" + requiredHiveVersion
	hiveDownloadURL     = "https://www.npmjs.com/package/@graphql-hive/cli/v/" + requiredHiveVersion
)

// Mode controls which checks are performed.
type Mode int

const (
	// FullMode checks all tools including kubectl and rover.
	FullMode Mode = iota
	// SchemaOnlyMode only checks hive CLI and hive config.
	SchemaOnlyMode
)

// Error represents a single preflight check failure.
type Error struct {
	Check   string
	Message string
}

func (e Error) Error() string {
	return fmt.Sprintf("[%s] %s", e.Check, e.Message)
}

// CheckResult holds all preflight check failures and non-blocking warnings.
type CheckResult struct {
	Errors   []Error
	Warnings []string
}

func (r CheckResult) OK() bool { return len(r.Errors) == 0 }

// Options configures the preflight checks.
type Options struct {
	Mode            Mode
	HiveConfigPath  string // legacy: path to hive JSON config file
	HiveEndpoint    string // hive registry endpoint URL
	HiveAccessToken string // hive registry access token
	KubectlContext  string
	SchemaFile      string
}

// Run performs the preflight checks and returns a CheckResult.
func Run(opts Options) CheckResult {
	var result CheckResult

	if opts.Mode == FullMode {
		if err := checkCommand("kubectl", "version", "--client"); err != nil {
			result.Errors = append(result.Errors, Error{Check: "kubectl", Message: "kubectl not found in PATH"})
		}
		if err := checkCommand("rover", "--version"); err != nil {
			result.Errors = append(result.Errors, Error{Check: "rover", Message: "rover not found — install from https://rover.apollo.dev"})
		}
	}

	hiveErrs, hiveWarns := checkHiveCLI()
	result.Errors = append(result.Errors, hiveErrs...)
	result.Warnings = append(result.Warnings, hiveWarns...)

	if errs := checkHiveConfig(opts.HiveConfigPath, opts.HiveEndpoint, opts.HiveAccessToken); len(errs) > 0 {
		result.Errors = append(result.Errors, errs...)
	}

	if opts.Mode == FullMode && opts.KubectlContext != "" {
		if err := checkKubectlContext(opts.KubectlContext); err != nil {
			result.Errors = append(result.Errors, Error{Check: "kubectl-context", Message: fmt.Sprintf("kubectl context %q not reachable: %v", opts.KubectlContext, err)})
		}
	}

	if opts.SchemaFile != "" {
		if _, err := os.Stat(opts.SchemaFile); os.IsNotExist(err) {
			result.Errors = append(result.Errors, Error{Check: "schema-file", Message: fmt.Sprintf("schema file not found: %s", opts.SchemaFile)})
		}
	}

	return result
}

var versionRE = regexp.MustCompile(`\b(\d+\.\d+\.\d+)\b`)

// checkHiveCLI verifies that hive CLI is installed and matches the required version.
// Missing → hard error. Wrong version → non-blocking warning.
func checkHiveCLI() (errs []Error, warnings []string) {
	out, err := exec.Command("hive", "--version").CombinedOutput()
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			// Not found in PATH.
			errs = append(errs, Error{
				Check: "hive",
				Message: fmt.Sprintf(
					"hive CLI not found — required version is v%s\n    Install: %s\n    Info:    %s",
					requiredHiveVersion, hiveInstallCmd, hiveDownloadURL,
				),
			})
			return
		}
	}

	// Parse version from output (e.g. "@graphql-hive/cli/0.42.1 darwin-arm64 ...").
	m := versionRE.FindSubmatch(out)
	if m == nil {
		return // installed but version unparseable — allow it
	}

	version := string(m[1])
	if version != requiredHiveVersion {
		warnings = append(warnings, fmt.Sprintf(
			"hive CLI v%s detected — only v%s is known to work stably\n    Downgrade: %s",
			version, requiredHiveVersion, hiveInstallCmd,
		))
	}
	return
}

// checkCommand verifies that a command exists by running it with the provided args.
func checkCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return nil // exit code > 0 but command exists
		}
		return err
	}
	return nil
}

// hiveConfigJSON is used only for structural validation.
type hiveConfigJSON struct {
	Registry struct {
		Endpoint    string `json:"endpoint"`
		AccessToken string `json:"accessToken"`
	} `json:"registry"`
}

func checkHiveConfig(configPath, endpoint, accessToken string) []Error {
	if configPath == "" && endpoint != "" && accessToken != "" {
		return nil
	}

	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if os.IsNotExist(err) {
			return []Error{{Check: "hive-config", Message: fmt.Sprintf("hive config not found: %s", configPath)}}
		}
		if err != nil {
			return []Error{{Check: "hive-config", Message: fmt.Sprintf("cannot read hive config: %v", err)}}
		}
		var cfg hiveConfigJSON
		if err := json.Unmarshal(data, &cfg); err != nil {
			return []Error{{Check: "hive-config", Message: "hive config is invalid JSON"}}
		}
		if cfg.Registry.AccessToken == "" {
			return []Error{{Check: "hive-config", Message: "hive config missing accessToken in registry"}}
		}
		return nil
	}

	return []Error{{Check: "hive-config", Message: "hive credentials not configured (run hpub to set up)"}}
}

func checkKubectlContext(ctx string) error {
	out, err := exec.Command("kubectl", "--context", ctx, "cluster-info").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	return nil
}
