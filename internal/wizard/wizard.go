package wizard

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/fponin/hpub/internal/config"
	"github.com/fponin/hpub/internal/state"
	"github.com/fponin/hpub/internal/ui"
)

const customOption = "[ custom... ]"
const backOption = "← Back"

// surveyOpt applies consistent prompt styling across all survey calls.
// magenta+b is readable on both light and dark terminal themes.
var surveyOpt = survey.WithIcons(func(icons *survey.IconSet) {
	icons.Question.Format     = "magenta+b"
	icons.SelectFocus.Format  = "magenta+b"
	icons.MarkedOption.Format = "magenta"
})

func init() {
	// survey hardcodes {{color "cyan"}} for the confirmed answer text in its templates.
	// Replace it with magenta+b to match the rest of the prompt styling.
	swapCyan := func(s string) string {
		return strings.ReplaceAll(s, `{{color "cyan"}}`, `{{color "magenta+b"}}`)
	}
	survey.SelectQuestionTemplate       = swapCyan(survey.SelectQuestionTemplate)
	survey.InputQuestionTemplate        = swapCyan(survey.InputQuestionTemplate)
	survey.ConfirmQuestionTemplate      = swapCyan(survey.ConfirmQuestionTemplate)
	survey.PasswordQuestionTemplate     = swapCyan(survey.PasswordQuestionTemplate)
	survey.MultiSelectQuestionTemplate  = swapCyan(survey.MultiSelectQuestionTemplate)
}

const (
	ScenarioFull       = "full"
	ScenarioSchemaOnly = "schema-only"
)

// ErrGoBack is returned when the user requests to go back to the previous step.
var ErrGoBack = errors.New("go back")

// IsBack reports whether err signals a "go back" intent (← Back selected).
func IsBack(err error) bool {
	return errors.Is(err, ErrGoBack)
}

// IsExit reports whether err is a Ctrl+C interrupt (clean exit, no error message).
func IsExit(err error) bool {
	return errors.Is(err, terminal.InterruptErr)
}

// SelectScenario prompts the user to choose between full pipeline or schema-only mode.
func SelectScenario() (string, error) {
	options := []string{
		"Full pipeline (download schema → check → publish)",
		"Check & publish from schema file",
		backOption,
	}
	var selected string
	if err := survey.AskOne(&survey.Select{
		Message: "Select scenario:",
		Options: options,
	}, &selected, surveyOpt); err != nil {
		return "", err
	}
	if selected == backOption {
		return "", ErrGoBack
	}
	if strings.HasPrefix(selected, "Full") {
		return ScenarioFull, nil
	}
	return ScenarioSchemaOnly, nil
}

// SelectSchemaFileWithValidation prompts for a schema file path, validates it exists,
// and offers recovery options if the file is not found.
func SelectSchemaFileWithValidation(st *state.State, cfg *config.AppConfig) (string, string, error) {
	defaultFile := cfg.Defaults.SchemaFile
	if len(st.RecentOutputFiles) > 0 {
		defaultFile = st.RecentOutputFiles[0]
	}
	for {
		var path string
		if err := survey.AskOne(&survey.Input{
			Message: "Schema file path:",
			Default: defaultFile,
		}, &path, surveyOpt); err != nil {
			return "", ScenarioSchemaOnly, err // includes interrupt → caller checks IsBack
		}

		expanded := path
		if strings.HasPrefix(path, "~/") {
			if home, err := os.UserHomeDir(); err == nil {
				expanded = filepath.Join(home, path[2:])
			}
		}

		if _, err := os.Stat(expanded); err == nil {
			return expanded, ScenarioSchemaOnly, nil
		}

		fmt.Printf("\n  File not found: %s\n\n", path)
		var action string
		if err := survey.AskOne(&survey.Select{
			Message: "What would you like to do?",
			Options: []string{
				"Enter a different path",
				"Switch to full pipeline",
				backOption,
			},
		}, &action, surveyOpt); err != nil {
			return "", ScenarioSchemaOnly, err
		}
		if strings.HasPrefix(action, "Switch") {
			return "", ScenarioFull, nil
		}
		if action == backOption {
			return "", ScenarioSchemaOnly, ErrGoBack
		}
		defaultFile = path
	}
}

// Input holds all values collected by the wizard.
type Input struct {
	Env         string
	Service     string
	PublishURL  string
	Namespace   string
	K8sResource string
	RemotePort  int
	LocalPort   int
	GraphQLPath string
	OutputFile  string
}

// SelectEnv prompts the user to choose an environment if one wasn't provided.
func SelectEnv(cfg *config.AppConfig, defaultEnv string) (string, error) {
	if defaultEnv != "" {
		if _, err := cfg.ResolveProfile(defaultEnv); err != nil {
			return "", err
		}
		return defaultEnv, nil
	}

	envNames := make([]string, 0, len(cfg.Environments))
	for k := range cfg.Environments {
		envNames = append(envNames, k)
	}

	var selected string
	prompt := &survey.Select{
		Message: "Select environment:",
		Options: envNames,
	}
	if err := survey.AskOne(prompt, &selected, surveyOpt); err != nil {
		return "", err
	}
	return selected, nil
}

// Run executes the full interactive wizard and returns the collected Input.
// Returns ErrGoBack if the user requests to go back past the first sub-step.
func Run(cfg *config.AppConfig, profile config.EnvProfile, st *state.State, env string) (Input, error) {
	input := Input{
		Env:       env,
		LocalPort: profile.DefaultLocalPort,
	}

	step := 0
	for {
		switch step {
		case 0: // Select subgraph
			service, err := selectSubgraph(cfg, st)
			if IsBack(err) {
				return input, ErrGoBack
			}
			if err != nil {
				return input, err
			}
			input.Service = service
			if sg, ok := cfg.FindSubgraph(service); ok {
				input.PublishURL = sg.PublishURL
				input.Namespace = sg.Namespace
				input.K8sResource = sg.K8sResource
				input.RemotePort = sg.RemotePort
				input.GraphQLPath = sg.GraphQLPath
			}
			step = 1

		case 1: // Confirm/override parameters
			if err := confirmParams(&input, profile, st, cfg); err != nil {
				if IsBack(err) {
					step = 0
					continue
				}
				return input, err
			}
			step = 2

		case 2: // Show summary and confirm
			if err := showSummaryAndConfirm(&input); err != nil {
				if IsBack(err) {
					step = 1
					continue
				}
				return input, err
			}
			return input, nil
		}
	}
}

// RunSchemaOnly asks only for service selection (no port-forward params).
// Returns ErrGoBack if the user requests to go back past the first sub-step.
func RunSchemaOnly(cfg *config.AppConfig, profile config.EnvProfile, st *state.State, env string) (Input, error) {
	input := Input{Env: env}

	step := 0
	for {
		switch step {
		case 0: // Select subgraph
			service, err := selectSubgraph(cfg, st)
			if IsBack(err) {
				return input, ErrGoBack
			}
			if err != nil {
				return input, err
			}
			input.Service = service
			if sg, ok := cfg.FindSubgraph(service); ok {
				input.PublishURL = sg.PublishURL
			}
			step = 1

		case 1: // Edit publish URL
			if err := survey.AskOne(&survey.Input{
				Message: "Publish URL:",
				Default: input.PublishURL,
			}, &input.PublishURL, surveyOpt); err != nil {
				if IsBack(err) {
					step = 0
					continue
				}
				return input, err
			}
			return input, nil
		}
	}
}

func selectSubgraph(cfg *config.AppConfig, st *state.State) (string, error) {
	options := make([]string, 0, len(cfg.Subgraphs)+2)

	lastService := st.LastUsed.Service
	if lastService != "" {
		options = append(options, lastService+" (last used)")
	}

	for _, sg := range cfg.Subgraphs {
		if sg.Name != lastService {
			options = append(options, sg.Name)
		}
	}
	options = append(options, "─────────────────")
	options = append(options, customOption)
	options = append(options, backOption)

	var selected string
	prompt := &survey.Select{
		Message:  "Select subgraph:",
		Options:  options,
		PageSize: 15,
	}
	if err := survey.AskOne(prompt, &selected, surveyOpt); err != nil {
		return "", err
	}

	if selected == backOption {
		return "", ErrGoBack
	}

	if selected == customOption || selected == "─────────────────" {
		var custom string
		if err := survey.AskOne(&survey.Input{Message: "Enter subgraph name:"}, &custom, surveyOpt); err != nil {
			return "", err
		}
		return custom, nil
	}

	if lastService != "" && selected == lastService+" (last used)" {
		return lastService, nil
	}
	return selected, nil
}

func confirmParams(input *Input, profile config.EnvProfile, st *state.State, cfg *config.AppConfig) error {
	defaultOutputFile := cfg.Defaults.SchemaFile
	if len(st.RecentOutputFiles) > 0 {
		defaultOutputFile = st.RecentOutputFiles[0]
	}

	qs := []*survey.Question{
		{
			Name:   "namespace",
			Prompt: &survey.Input{Message: "Namespace:", Default: input.Namespace},
		},
		{
			Name:   "k8sResource",
			Prompt: &survey.Input{Message: "K8s resource:", Default: input.K8sResource},
		},
		{
			Name:   "remotePort",
			Prompt: &survey.Input{Message: "Remote port:", Default: fmt.Sprintf("%d", input.RemotePort)},
		},
		{
			Name:   "localPort",
			Prompt: &survey.Input{Message: "Local port:", Default: fmt.Sprintf("%d", profile.DefaultLocalPort)},
		},
		{
			Name:   "outputFile",
			Prompt: &survey.Input{Message: "Output schema file:", Default: defaultOutputFile},
		},
	}

	answers := struct {
		Namespace   string
		K8sResource string
		RemotePort  string
		LocalPort   string
		OutputFile  string
	}{}

	if err := survey.Ask(qs, &answers, surveyOpt); err != nil {
		if IsBack(err) {
			return ErrGoBack
		}
		return err
	}

	input.Namespace = answers.Namespace
	input.K8sResource = answers.K8sResource
	input.OutputFile = answers.OutputFile

	if _, err := fmt.Sscanf(answers.RemotePort, "%d", &input.RemotePort); err != nil {
		return fmt.Errorf("invalid remote port: %s", answers.RemotePort)
	}
	if _, err := fmt.Sscanf(answers.LocalPort, "%d", &input.LocalPort); err != nil {
		return fmt.Errorf("invalid local port: %s", answers.LocalPort)
	}
	return nil
}

func showSummaryAndConfirm(input *Input) error {
	ui.Divider()
	ui.Println("Pipeline Summary")
	ui.Divider()
	ui.Printf("  %-16s: %s\n", "Environment", input.Env)
	ui.Printf("  %-16s: %s\n", "Service", input.Service)
	ui.Printf("  %-16s: %s\n", "Namespace", input.Namespace)
	ui.Printf("  %-16s: %s → localhost:%d\n", "Port-forward", input.K8sResource, input.LocalPort)
	ui.Printf("  %-16s: %s\n", "Output file", input.OutputFile)
	ui.Divider()

	var proceed bool
	if err := survey.AskOne(&survey.Confirm{
		Message: "Proceed?",
		Default: true,
	}, &proceed, surveyOpt); err != nil {
		if IsBack(err) {
			return ErrGoBack
		}
		return err
	}

	if !proceed {
		return fmt.Errorf("cancelled by user")
	}
	return nil
}

// AskPublishURL prompts the user to confirm or edit the publish URL before publishing.
func AskPublishURL(input *Input) error {
	return survey.AskOne(&survey.Input{
		Message: "Publish URL (edit if needed):",
		Default: input.PublishURL,
	}, &input.PublishURL, surveyOpt)
}

// CollectPublishInfo prompts for author and commit message.
func CollectPublishInfo(st *state.State) (author, commit string, err error) {
	qs := []*survey.Question{
		{
			Name: "author",
			Prompt: &survey.Input{
				Message: "Author:",
				Default: st.LastAuthor,
			},
		},
		{
			Name: "commit",
			Prompt: &survey.Select{
				Message: "Commit message (select or type):",
				Options: append(st.RecentCommitMessages, "[ enter new... ]"),
			},
		},
	}

	answers := struct {
		Author string
		Commit string
	}{}

	if err = survey.Ask(qs, &answers, surveyOpt); err != nil {
		return
	}

	author = answers.Author
	commit = answers.Commit

	if commit == "[ enter new... ]" {
		if err = survey.AskOne(&survey.Input{Message: "Commit message:"}, &commit, surveyOpt); err != nil {
			return
		}
	}
	return
}

// SetupHiveCredentials checks if hive credentials are configured for the env.
// If not, prompts the user for endpoint + token and saves them to the config file.
func SetupHiveCredentials(cfg *config.AppConfig, cfgPath, env string, profile config.EnvProfile) (config.EnvProfile, error) {
	// Already configured via file or direct credentials
	if profile.HiveConfigPath != "" || (profile.HiveEndpoint != "" && profile.HiveAccessToken != "") {
		return profile, nil
	}

	fmt.Printf("\n  Hive credentials for [%s] are not configured.\n", env)

	qs := []*survey.Question{
		{
			Name: "endpoint",
			Prompt: &survey.Input{
				Message: "Hive registry endpoint:",
				Default: cfg.Defaults.HiveEndpoint,
			},
		},
		{
			Name: "accessToken",
			Prompt: &survey.Password{
				Message: "Hive access token:",
			},
			Validate: func(ans interface{}) error {
				token := strings.TrimSpace(fmt.Sprintf("%v", ans))
				if strings.ContainsAny(token, `"{}:, `) || strings.Contains(token, "accessToken") {
					return fmt.Errorf("invalid format — paste only the token value, e.g. a1408bb9d195b69482ebf83502f29e21")
				}
				return nil
			},
		},
	}

	answers := struct {
		Endpoint    string
		AccessToken string
	}{}

	if err := survey.Ask(qs, &answers, surveyOpt); err != nil {
		return profile, err
	}

	profile.HiveEndpoint = answers.Endpoint
	profile.HiveAccessToken = answers.AccessToken

	if cfg.Environments == nil {
		cfg.Environments = make(map[string]config.EnvProfile)
	}
	cfg.Environments[env] = profile

	if err := config.UpdateEnvProfile(cfgPath, env, profile); err != nil {
		return profile, fmt.Errorf("saving hive credentials: %w", err)
	}

	return profile, nil
}

// SelectKubectlContext runs `kubectl config get-contexts -o name`, shows a selector,
// and saves the chosen context to the config file.
func SelectKubectlContext(cfg *config.AppConfig, cfgPath, env string, profile config.EnvProfile) (config.EnvProfile, error) {
	out, err := exec.Command("kubectl", "config", "get-contexts", "-o", "name").Output()
	if err != nil {
		return profile, fmt.Errorf("kubectl config get-contexts: %w", err)
	}

	var contexts []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			contexts = append(contexts, line)
		}
	}

	if len(contexts) == 0 {
		return profile, fmt.Errorf("no kubectl contexts found")
	}

	options := append(contexts, backOption)
	var selected string
	if err := survey.AskOne(&survey.Select{
		Message:  "Select kubectl context:",
		Options:  options,
		PageSize: 15,
	}, &selected, surveyOpt); err != nil {
		return profile, err
	}

	if selected == backOption {
		return profile, ErrGoBack
	}

	profile.KubectlContext = selected

	if cfg.Environments == nil {
		cfg.Environments = make(map[string]config.EnvProfile)
	}
	cfg.Environments[env] = profile

	if err := config.UpdateEnvProfile(cfgPath, env, profile); err != nil {
		return profile, fmt.Errorf("saving kubectl context: %w", err)
	}

	return profile, nil
}

// ProdSafeguard asks for double confirmation before publishing to production.
// Returns an error if the user cancels or does not type "PROD".
func ProdSafeguard() error {
	ui.Warn("You are about to publish to PRODUCTION")
	fmt.Println()

	var confirm bool
	if err := survey.AskOne(&survey.Confirm{
		Message: "Publish schema to production?",
		Default: false,
	}, &confirm, surveyOpt); err != nil {
		return err
	}
	if !confirm {
		return fmt.Errorf("publish cancelled")
	}

	var typed string
	if err := survey.AskOne(&survey.Input{
		Message: "Type PROD to confirm:",
	}, &typed, surveyOpt); err != nil {
		return err
	}
	if typed != "PROD" {
		return fmt.Errorf("publish cancelled (confirmation text mismatch)")
	}
	return nil
}
