package cmd

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

// evalServer manages a user EvalServer process launched for the duration of a
// test run (the `agk eval --serve` workflow).
type evalServer struct {
	cmd    *exec.Cmd
	output *syncBuffer
	once   sync.Once
}

// syncBuffer is a goroutine-safe buffer for capturing child process output.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// startEvalServer launches the project in EvalServer mode (AGK_EVAL_MODE=true).
// The default command is `go run .` in dir; customCmd overrides it. The process is
// started in its own process group so `go run`'s compiled child can be reliably killed.
func startEvalServer(dir, customCmd string, streamOutput bool) (*evalServer, error) {
	name, args := parseServeCmd(customCmd)

	c := exec.Command(name, args...) //nolint:gosec // command is user-provided by design
	c.Dir = dir
	c.Env = append(os.Environ(), "AGK_EVAL_MODE=true")
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	out := &syncBuffer{}
	var w io.Writer = out
	if streamOutput {
		w = io.MultiWriter(out, &prefixWriter{prefix: "[server] ", w: os.Stderr})
	}
	c.Stdout = w
	c.Stderr = w

	if err := c.Start(); err != nil {
		return nil, fmt.Errorf("failed to start eval server (%s): %w", name, err)
	}
	return &evalServer{cmd: c, output: out}, nil
}

// parseServeCmd returns the command name and args, defaulting to `go run .`.
func parseServeCmd(customCmd string) (string, []string) {
	if fields := strings.Fields(customCmd); len(fields) > 0 {
		return fields[0], fields[1:]
	}
	return "go", []string{"run", "."}
}

// Stop terminates the server process group (idempotent), escalating SIGTERM→SIGKILL.
func (s *evalServer) Stop() {
	s.once.Do(func() {
		if s.cmd.Process == nil {
			return
		}
		s.signalGroup(syscall.SIGTERM)

		done := make(chan struct{})
		go func() { _ = s.cmd.Wait(); close(done) }()

		select {
		case <-done:
		case <-time.After(3 * time.Second):
			s.signalGroup(syscall.SIGKILL)
			<-done
		}
	})
}

func (s *evalServer) signalGroup(sig syscall.Signal) {
	if pgid, err := syscall.Getpgid(s.cmd.Process.Pid); err == nil {
		_ = syscall.Kill(-pgid, sig)
	} else {
		_ = s.cmd.Process.Signal(sig)
	}
}

// Output returns everything the server printed to stdout/stderr so far.
func (s *evalServer) Output() string { return s.output.String() }

// waitForHealthy polls url + "/health" until it returns 200 or the timeout elapses.
func waitForHealthy(url string, timeout time.Duration) error {
	client := &http.Client{Timeout: 3 * time.Second}
	deadline := time.Now().Add(timeout)
	healthURL := strings.TrimRight(url, "/") + "/health"

	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := client.Get(healthURL)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
			lastErr = fmt.Errorf("health returned HTTP %d", resp.StatusCode)
		} else {
			lastErr = err
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("server not healthy within %s: %w", timeout, lastErr)
}

// launchAndWait starts the eval server and blocks until it is healthy. On failure it
// stops the server and surfaces its captured output to aid debugging.
func launchAndWait(dir, customCmd, targetURL string, waitSecs int, verbose bool) (*evalServer, error) {
	if targetURL == "" {
		return nil, fmt.Errorf("--serve requires a target URL in the test file")
	}

	fmt.Printf("🚀 Launching EvalServer from %s (AGK_EVAL_MODE=true)...\n", dir)
	srv, err := startEvalServer(dir, customCmd, verbose)
	if err != nil {
		return nil, err
	}

	fmt.Printf("⏳ Waiting up to %ds for %s to become healthy...\n", waitSecs, targetURL)
	if err := waitForHealthy(targetURL, time.Duration(waitSecs)*time.Second); err != nil {
		out := srv.Output()
		srv.Stop()
		if strings.TrimSpace(out) != "" {
			fmt.Fprintf(os.Stderr, "\n--- server output ---\n%s\n---------------------\n", out)
		}
		return nil, fmt.Errorf("eval server did not start: %w", err)
	}

	fmt.Println("✓ Server is healthy")
	return srv, nil
}

// prefixWriter prefixes each write with a label (used to tag streamed server output).
type prefixWriter struct {
	prefix string
	w      io.Writer
}

func (p *prefixWriter) Write(b []byte) (int, error) {
	if _, err := io.WriteString(p.w, p.prefix); err != nil {
		return 0, err
	}
	return p.w.Write(b)
}
