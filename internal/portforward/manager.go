package portforward

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/fponin/hpub/internal/ui"
)

// Manager manages a kubectl port-forward subprocess.
type Manager struct {
	kubectlContext string
	k8sResource    string
	namespace      string
	localPort      int
	remotePort     int

	mu      sync.Mutex
	cmd     *exec.Cmd
	stopped bool
}

// New creates a new Manager with the given parameters.
func New(kubectlContext, k8sResource, namespace string, localPort, remotePort int) *Manager {
	return &Manager{
		kubectlContext: kubectlContext,
		k8sResource:    k8sResource,
		namespace:      namespace,
		localPort:      localPort,
		remotePort:     remotePort,
	}
}

// Start launches the port-forward subprocess and waits until it signals readiness.
// Returns an error if the process fails to start or doesn't become ready within 15 seconds.
func (m *Manager) Start(ctx context.Context) error {
	args := []string{
		"--context", m.kubectlContext,
		"port-forward",
		m.k8sResource,
		fmt.Sprintf("%d:%d", m.localPort, m.remotePort),
		"-n", m.namespace,
	}

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	m.cmd = cmd

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("creating stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting kubectl port-forward: %w", err)
	}

	ready := make(chan struct{}, 1)
	errCh := make(chan error, 1)

	// Watch stdout for readiness signal
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			ui.Info("  [pf] " + line)
			if strings.Contains(line, "Forwarding from 127.0.0.1") {
				select {
				case ready <- struct{}{}:
				default:
				}
			}
		}
	}()

	// Drain stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(strings.ToLower(line), "already in use") {
				errCh <- fmt.Errorf("port %d is already in use — change localPort in the wizard", m.localPort)
				return
			}
			ui.StepWarn("[pf stderr] " + line)
		}
	}()

	// Monitor process exit
	go func() {
		if err := cmd.Wait(); err != nil {
			select {
			case errCh <- fmt.Errorf("kubectl port-forward exited unexpectedly: %w", err):
			default:
			}
		}
	}()

	select {
	case <-ready:
		return nil
	case err := <-errCh:
		m.Stop()
		return err
	case <-time.After(15 * time.Second):
		m.Stop()
		return fmt.Errorf("port-forward readiness timeout (15s)")
	case <-ctx.Done():
		m.Stop()
		return ctx.Err()
	}
}

// WaitReady polls the local endpoint until it responds with any HTTP status.
// This confirms the upstream service is actually reachable.
func (m *Manager) WaitReady(ctx context.Context, graphqlPath string) error {
	url := fmt.Sprintf("http://localhost:%d%s", m.localPort, graphqlPath)
	client := &http.Client{Timeout: 2 * time.Second}

	for attempt := 0; attempt < 20; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("building probe request: %w", err)
		}
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			return nil // any HTTP response = service is alive
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
	return fmt.Errorf("endpoint not ready after 10 seconds (20 attempts)")
}

// Stop terminates the port-forward subprocess.
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.stopped || m.cmd == nil || m.cmd.Process == nil {
		return
	}
	m.stopped = true

	_ = m.cmd.Process.Kill()
	// We don't call cmd.Wait() here because it may have already been waited in the goroutine above.
}
