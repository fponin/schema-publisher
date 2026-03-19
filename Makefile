
BINARY  := hpub
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X github.com/fponin/hpub/cmd.Version=$(VERSION)"

.PHONY: build install test lint clean package

build:
	go build $(LDFLAGS) -o $(BINARY) .

install: build
	sudo cp $(BINARY) /usr/local/bin/$(BINARY)
	sudo chmod +x /usr/local/bin/$(BINARY)
	@echo "Installed to /usr/local/bin/$(BINARY)"

test:
	go test ./...

lint:
	go vet ./...

clean:
	rm -f $(BINARY)
	rm -rf dist/

package: build
	@mkdir -p dist
	@cp $(BINARY) dist/$(BINARY)
	@printf '# =============================================================\n'                                  > dist/defaults.yaml
	@printf '# hpub — GraphQL subgraph schema publisher\n'                                                      >> dist/defaults.yaml
	@printf '#\n'                                                                                                >> dist/defaults.yaml
	@printf '# INSTALL\n'                                                                                       >> dist/defaults.yaml
	@printf '#   sudo cp hpub /usr/local/bin/ && sudo chmod +x /usr/local/bin/hpub && hpub config init\n'       >> dist/defaults.yaml
	@printf '#\n'                                                                                                >> dist/defaults.yaml
	@printf '# PREREQUISITES\n'                                                                                 >> dist/defaults.yaml
	@printf '#   kubectl   — https://kubernetes.io/docs/tasks/tools/\n'                                         >> dist/defaults.yaml
	@printf '#   rover     — https://rover.apollo.dev\n'                                                        >> dist/defaults.yaml
	@printf '#   hive CLI  — npm install -g @graphql-hive/cli@0.42.1  (only 0.42.1 works stably)\n'            >> dist/defaults.yaml
	@printf '#\n'                                                                                                >> dist/defaults.yaml
	@printf '# USAGE\n'                                                                                         >> dist/defaults.yaml
	@printf '#   hpub run                            interactive wizard\n'                                      >> dist/defaults.yaml
	@printf '#   hpub run --schema ./schema.graphql  use existing schema file\n'                                >> dist/defaults.yaml
	@printf '#   hpub run --env dev                  skip env selection\n'                                      >> dist/defaults.yaml
	@printf '#   hpub check --schema ./schema.graphql  check only, no publish\n'                               >> dist/defaults.yaml
	@printf '#   hpub config show                    print current config\n'                                    >> dist/defaults.yaml
	@printf '#   hpub config edit                    open config in $$EDITOR\n'                                 >> dist/defaults.yaml
	@printf '# =============================================================\n'                                 >> dist/defaults.yaml
	@printf '\n'                                                                                                 >> dist/defaults.yaml
	@printf '# Fill in your real values below.\n'                                                               >> dist/defaults.yaml
	@printf '\n'                                                                                                 >> dist/defaults.yaml
	@printf 'defaults:\n'                                                                                       >> dist/defaults.yaml
	@printf '  schemaFile: "~/new.graphql"\n'                                                                   >> dist/defaults.yaml
	@printf '  hiveEndpoint: "https://hive.example.com/graphql"\n'                                              >> dist/defaults.yaml
	@printf '\n'                                                                                                 >> dist/defaults.yaml
	@printf 'environments:\n'                                                                                    >> dist/defaults.yaml
	@printf '  dev:\n'                                                                                          >> dist/defaults.yaml
	@printf '    authUrl: "https://auth.example.com/dshop/auth"\n'                                              >> dist/defaults.yaml
	@printf '    authBearerToken: "change-me"\n'                                                                >> dist/defaults.yaml
	@printf '    defaultLocalPort: 8080\n'                                                                      >> dist/defaults.yaml
	@printf '    jwtHeader: "jwt-token"\n'                                                                      >> dist/defaults.yaml
	@printf '  stage:\n'                                                                                        >> dist/defaults.yaml
	@printf '    authUrl: "https://auth-stage.example.com/dshop/auth"\n'                                        >> dist/defaults.yaml
	@printf '    authBearerToken: "change-me"\n'                                                                >> dist/defaults.yaml
	@printf '    defaultLocalPort: 8080\n'                                                                      >> dist/defaults.yaml
	@printf '    jwtHeader: "jwt-token"\n'                                                                      >> dist/defaults.yaml
	@printf '  prod:\n'                                                                                         >> dist/defaults.yaml
	@printf '    authUrl: "https://auth-prod.example.com/dshop/auth"\n'                                         >> dist/defaults.yaml
	@printf '    authBearerToken: "change-me"\n'                                                                >> dist/defaults.yaml
	@printf '    defaultLocalPort: 8080\n'                                                                      >> dist/defaults.yaml
	@printf '    jwtHeader: "jwt-token"\n'                                                                      >> dist/defaults.yaml
	@printf '\n'                                                                                                 >> dist/defaults.yaml
	@printf 'subgraphs:\n'                                                                                      >> dist/defaults.yaml
	@printf '  - name: test\n'                                                                                  >> dist/defaults.yaml
	@printf '    publishUrl: "http://test-service:8080/graphql"\n'                                              >> dist/defaults.yaml
	@printf '    k8sResource: "svc/test-service"\n'                                                             >> dist/defaults.yaml
	@printf '    namespace: default\n'                                                                          >> dist/defaults.yaml
	@printf '    remotePort: 8080\n'                                                                            >> dist/defaults.yaml
	@printf '    graphqlPath: "/graphql"\n'                                                                     >> dist/defaults.yaml
	@echo ""
	@echo "Package ready in dist/:"
	@echo "  dist/hpub"
	@echo "  dist/defaults.yaml"
	@echo ""
	@echo "Install command for recipient:"
	@echo "  sudo cp hpub /usr/local/bin/ && sudo chmod +x /usr/local/bin/hpub && hpub config init"
