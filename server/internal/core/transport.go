package core

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"one-mcp/internal/model"
)

// Transport defines the interface for MCP communication
type Transport interface {
	// Start begins the transport connection/process and blocks until it ends.
	// onMessage is called for every incoming JSON-RPC message.
	// onReady is called when the transport is ready to send messages.
	Start(ctx context.Context, onMessage func([]byte), onReady func()) error
	
	// Send sends a JSON-RPC payload to the upstream.
	Send(payload []byte) error
	
	// Close cleans up resources and stops the transport.
	Close() error
}

// SSETransport implements Transport using Server-Sent Events and HTTP POST
type SSETransport struct {
	Config   model.UpstreamServer
	Endpoint string // The POST endpoint discovered via SSE
	Client   *http.Client
	
	mu       io.Closer // Used to close the response body of the long-polling GET
}

func NewSSETransport(cfg model.UpstreamServer) *SSETransport {
	return &SSETransport{
		Config: cfg,
		Client: &http.Client{Timeout: 0},
	}
}

func (t *SSETransport) Start(ctx context.Context, onMessage func([]byte), onReady func()) error {
	fmt.Printf("[SSETransport %s] Connecting to %s...\n", t.Config.Name, t.Config.URL)
	req, err := http.NewRequestWithContext(ctx, "GET", t.Config.URL, nil)
	if err != nil {
		return err
	}
	
	req.Header.Set("Accept", "text/event-stream")
	if t.Config.AuthToken != "" {
		// Sanitize AuthToken to prevent header injection
		token := strings.Map(func(r rune) rune {
			if r == '\n' || r == '\r' {
				return -1
			}
			return r
		}, t.Config.AuthToken)
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := t.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		return fmt.Errorf("bad status code: %d", resp.StatusCode)
	}
	
	t.mu = resp.Body

	scanner := bufio.NewScanner(resp.Body)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: endpoint") {
			if scanner.Scan() {
				dataLine := scanner.Text()
				if strings.HasPrefix(dataLine, "data: ") {
					endpoint := strings.TrimPrefix(dataLine, "data: ")
					u, err := url.Parse(t.Config.URL)
					if err == nil {
						ref, _ := url.Parse(endpoint)
						t.Endpoint = u.ResolveReference(ref).String()
					} else {
						t.Endpoint = endpoint
					}
					fmt.Printf("[SSETransport %s] Endpoint discovered: %s\n", t.Config.Name, t.Endpoint)
					if onReady != nil {
						go onReady()
					}
				}
			}
		} else if strings.HasPrefix(line, "data: ") {
			msgStr := strings.TrimPrefix(line, "data: ")
			if len(msgStr) > 0 {
				onMessage([]byte(msgStr))
			}
		}
	}
	
	return scanner.Err()
}

func (t *SSETransport) Send(payload []byte) error {
	if t.Endpoint == "" {
		return fmt.Errorf("endpoint not yet discovered")
	}

	fmt.Printf("[SSETransport %s] POST %s Payload: %s\n", t.Config.Name, t.Endpoint, string(payload))

	req, err := http.NewRequest("POST", t.Endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if t.Config.AuthToken != "" {
		// Sanitize AuthToken to prevent header injection
		token := strings.Map(func(r rune) rune {
			if r == '\n' || r == '\r' {
				return -1
			}
			return r
		}, t.Config.AuthToken)
		req.Header.Set("Authorization", "Bearer "+token)
	}
	
	resp, err := t.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		return fmt.Errorf("upstream returned error: %d", resp.StatusCode)
	}
	return nil
}

func (t *SSETransport) Close() error {
	if t.mu != nil {
		return t.mu.Close()
	}
	return nil
}

// StdioTransport implements Transport using local process execution
type StdioTransport struct {
	Config model.UpstreamServer
	cmd    *exec.Cmd
	stdin  io.WriteCloser
}

func NewStdioTransport(cfg model.UpstreamServer) *StdioTransport {
	return &StdioTransport{
		Config: cfg,
	}
}

func (t *StdioTransport) Start(ctx context.Context, onMessage func([]byte), onReady func()) error {
	var args []string
	if t.Config.Args != "" {
		if err := json.Unmarshal([]byte(t.Config.Args), &args); err != nil {
			return fmt.Errorf("invalid args: %v", err)
		}
	}

	// Validate command and args for potential injection
	if err := ValidateCommand(t.Config.Command, args); err != nil {
		return err
	}

	fmt.Printf("[StdioTransport %s] Starting command: %s %v\n", t.Config.Name, t.Config.Command, args)
	
	t.cmd = exec.CommandContext(ctx, t.Config.Command, args...)
	
	// Set Environment
	t.cmd.Env = os.Environ() // Inherit current env
	if t.Config.Env != "" {
		var envMap map[string]string
		if err := json.Unmarshal([]byte(t.Config.Env), &envMap); err == nil {
			for k, v := range envMap {
				t.cmd.Env = append(t.cmd.Env, fmt.Sprintf("%s=%s", k, v))
			}
		}
	}

	stdin, err := t.cmd.StdinPipe()
	if err != nil {
		return err
	}
	t.stdin = stdin

	stdout, err := t.cmd.StdoutPipe()
	if err != nil {
		return err
	}
	
	stderr, err := t.cmd.StderrPipe()
	if err != nil {
		return err
	}

	// Handle Stderr logging in background
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Printf("[StdioTransport %s] STDERR: %s\n", t.Config.Name, scanner.Text())
		}
	}()

	if err := t.cmd.Start(); err != nil {
		return err
	}

	if onReady != nil {
		go onReady()
	}

	// Read Stdout in this goroutine (blocking)
	scanner := bufio.NewScanner(stdout)
	// Large buffer just in case
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		// Copy buffer because scanner reuses it
		msg := make([]byte, len(line))
		copy(msg, line)

		onMessage(msg)
	}

	if err := t.cmd.Wait(); err != nil {
		fmt.Printf("[StdioTransport %s] Process exited with error: %v\n", t.Config.Name, err)
		return err
	}

	fmt.Printf("[StdioTransport %s] Process exited cleanly\n", t.Config.Name)
	return nil
}

func ValidateCommand(command string, args []string) error {
	if command == "" {
		return fmt.Errorf("command is empty")
	}

	// Forbid shell metacharacters in command
	forbidden := ";|&><$()!`*?[]{}~\\\"'\n\r"
	if strings.ContainsAny(command, forbidden) {
		return fmt.Errorf("malicious characters in command")
	}

	// Forbid shell metacharacters in arguments
	for _, arg := range args {
		if strings.ContainsAny(arg, forbidden) {
			return fmt.Errorf("malicious characters in argument: %s", arg)
		}
	}

	return nil
}

func (t *StdioTransport) Send(payload []byte) error {
	if t.stdin == nil {
		return fmt.Errorf("stdin not open")
	}
	
	// JSON-RPC over stdio is typically line-delimited
	// Ensure newline
	if !bytes.HasSuffix(payload, []byte("\n")) {
		payload = append(payload, '\n')
	}
	
	_, err := t.stdin.Write(payload)
	return err
}

func (t *StdioTransport) Close() error {
	if t.cmd != nil && t.cmd.Process != nil {
		return t.cmd.Process.Kill()
	}
	return nil
}
