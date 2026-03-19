package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// DefaultConfigPath returns the default path for the app config file.
func DefaultConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "hpub", "config.yaml")
}

// Load reads the AppConfig from the given path. Returns an error if the file
// does not exist or cannot be parsed.
func Load(path string) (*AppConfig, error) {
	path = ExpandTilde(path)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	var cfg AppConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}

// Save writes the AppConfig to the given path, creating parent directories as needed.
func Save(path string, cfg *AppConfig) error {
	path = ExpandTilde(path)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("serializing config: %w", err)
	}
	return os.WriteFile(path, data, 0o600)
}

// LoadOrDefault loads the config from path; if the file doesn't exist, returns the default config.
// For existing configs that predate the defaults section, wizard defaults are backfilled.
func LoadOrDefault(path string) (*AppConfig, error) {
	expandedPath := ExpandTilde(path)
	if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
		def := DefaultConfig()
		return &def, nil
	}
	cfg, err := Load(path)
	if err != nil {
		return nil, err
	}
	// Backfill any wizard defaults not present in the user's config.
	if cfg.Defaults.SchemaFile == "" || cfg.Defaults.HiveEndpoint == "" {
		def := DefaultConfig()
		if cfg.Defaults.SchemaFile == "" {
			cfg.Defaults.SchemaFile = def.Defaults.SchemaFile
		}
		if cfg.Defaults.HiveEndpoint == "" {
			cfg.Defaults.HiveEndpoint = def.Defaults.HiveEndpoint
		}
	}
	return cfg, nil
}

// ExpandTilde replaces a leading "~" with the user's home directory.
func ExpandTilde(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

// ResolveProfile looks up the named environment profile from the config.
func (c *AppConfig) ResolveProfile(env string) (EnvProfile, error) {
	profile, ok := c.Environments[env]
	if !ok {
		return EnvProfile{}, fmt.Errorf("environment %q not found in config", env)
	}
	// Expand tilde in HiveConfigPath
	profile.HiveConfigPath = ExpandTilde(profile.HiveConfigPath)
	return profile, nil
}

// UpdateEnvProfile saves an updated EnvProfile for the given environment to the config file.
func UpdateEnvProfile(cfgPath, env string, profile EnvProfile) error {
	cfg, err := LoadOrDefault(cfgPath)
	if err != nil {
		return fmt.Errorf("loading config for update: %w", err)
	}
	if cfg.Environments == nil {
		cfg.Environments = make(map[string]EnvProfile)
	}
	cfg.Environments[env] = profile
	return Save(cfgPath, cfg)
}

// FindSubgraph looks up a subgraph by name.
func (c *AppConfig) FindSubgraph(name string) (*SubgraphEntry, bool) {
	for i := range c.Subgraphs {
		if c.Subgraphs[i].Name == name {
			return &c.Subgraphs[i], true
		}
	}
	return nil, false
}
