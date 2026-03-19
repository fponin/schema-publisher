package config

// WizardDefaults holds prompt default values that can be overridden in config.yaml.
type WizardDefaults struct {
	SchemaFile   string `yaml:"schemaFile"`
	HiveEndpoint string `yaml:"hiveEndpoint"`
}

// AppConfig is the top-level configuration structure.
type AppConfig struct {
	Defaults     WizardDefaults        `yaml:"defaults,omitempty"`
	Environments map[string]EnvProfile `yaml:"environments"`
	Subgraphs    []SubgraphEntry       `yaml:"subgraphs"`
}

// EnvProfile holds environment-specific settings.
type EnvProfile struct {
	AuthURL          string `yaml:"authUrl"`
	AuthBearerToken  string `yaml:"authBearerToken"`
	HiveConfigPath   string `yaml:"hiveConfigPath,omitempty"`   // legacy: path to hive JSON config file
	HiveEndpoint     string `yaml:"hiveEndpoint,omitempty"`     // hive registry endpoint URL
	HiveAccessToken  string `yaml:"hiveAccessToken,omitempty"`  // hive registry access token
	KubectlContext   string `yaml:"kubectlContext,omitempty"`
	DefaultLocalPort int    `yaml:"defaultLocalPort"`
	JWTHeader        string `yaml:"jwtHeader"`
}

// SubgraphEntry describes a known subgraph service.
type SubgraphEntry struct {
	Name        string `yaml:"name"`
	PublishURL  string `yaml:"publishUrl"`
	K8sResource string `yaml:"k8sResource"`
	Namespace   string `yaml:"namespace"`
	RemotePort  int    `yaml:"remotePort"`
	GraphQLPath string `yaml:"graphqlPath"`
}
