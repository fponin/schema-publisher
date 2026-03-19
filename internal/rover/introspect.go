package rover

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// Introspect runs `rover subgraph introspect` against the given local endpoint
// and returns the schema bytes. The JWT token is passed as a request header.
func Introspect(ctx context.Context, localPort int, graphqlPath, jwtHeader, token string, verbose bool) ([]byte, error) {
	endpoint := fmt.Sprintf("http://localhost:%d%s", localPort, graphqlPath)
	headerValue := fmt.Sprintf("%s: %s", jwtHeader, token)

	args := []string{
		"subgraph", "introspect",
		endpoint,
		"--header", headerValue,
	}

	cmd := exec.CommandContext(ctx, "rover", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = err.Error()
		}
		return nil, fmt.Errorf("rover introspect failed: %s", errMsg)
	}

	schema := stdout.Bytes()
	if len(schema) == 0 {
		return nil, fmt.Errorf("rover introspect returned empty schema")
	}

	if verbose {
		fmt.Printf("[rover stderr] %s\n", stderr.String())
	}

	return schema, nil
}
