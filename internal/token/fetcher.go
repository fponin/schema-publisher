package token

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/fponin/hpub/internal/ui"
)

type authResponse struct {
	JWTToken string `json:"jwtToken"`
}

// Fetch retrieves an auth token from authURL using the provided bearerToken.
// The token value is never logged in full; only the first 8 characters are shown.
func Fetch(ctx context.Context, authURL, bearerToken string) (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, authURL, nil)
	if err != nil {
		return "", fmt.Errorf("building auth request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+bearerToken)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("auth endpoint unreachable: check VPN/network")
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("auth endpoint returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading auth response: %w", err)
	}

	var ar authResponse
	if err := json.Unmarshal(body, &ar); err != nil {
		return "", fmt.Errorf("parsing auth response: %w", err)
	}

	if ar.JWTToken == "" {
		return "", fmt.Errorf("auth response missing jwtToken field")
	}

	// Log only first 8 characters for safety
	preview := ar.JWTToken
	if len(preview) > 8 {
		preview = preview[:8] + "..."
	}
	ui.StepOK("token " + preview)

	return ar.JWTToken, nil
}
