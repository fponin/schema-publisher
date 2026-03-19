package config

// DefaultConfig returns a template AppConfig with placeholder values.
// Recipients should replace all placeholder values before use.
func DefaultConfig() AppConfig {
	return AppConfig{
		Defaults: WizardDefaults{
			SchemaFile:   "~/new.graphql",
			HiveEndpoint: "https://hive.example.com/graphql",
		},
		Environments: map[string]EnvProfile{
			"dev": {
				AuthURL:          "https://auth.example.com/dshop/auth",
				AuthBearerToken:  "change-me",
				DefaultLocalPort: 8080,
				JWTHeader:        "jwt-token",
			},
		},
		Subgraphs: []SubgraphEntry{
			{
				Name:        "test",
				PublishURL:  "http://test-service:8080/graphql",
				K8sResource: "svc/test-service",
				Namespace:   "default",
				RemotePort:  8080,
				GraphQLPath: "/graphql",
			},
		},
	}
}
