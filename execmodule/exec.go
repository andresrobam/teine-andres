package execmodule

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Config holds the SSH connection details for the remote sandbox VM.
type Config struct {
	Host    string // Remote VM hostname or IP
	Port    int    // SSH port (default "22")
	User    string // SSH username
	KeyPath string // Path to SSH private key file
}

// Client wraps an SSH config and provides tool execution over SSH.
type Client struct {
	config Config
}

// NewClient creates a new exec client with the given SSH configuration.
func NewClient(cfg Config) *Client {
	return &Client{config: cfg}
}

// execArgs represents the arguments for the exec tool.
type execArgs struct {
	Command        string `json:"command"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"`
}

// ToolSpec represents a tool specification for the AI model.
type ToolSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// Tool represents a tool definition with type and spec.
type Tool struct {
	Type     string   `json:"type"`
	Function ToolSpec `json:"function"`
}

// GetToolSpecs returns the tool specifications for the exec tool.
func GetToolSpecs() []Tool {
	return []Tool{
		{
			Type: "function",
			Function: ToolSpec{
				Name:        "exec",
				Description: "Execute a shell command on a remote sandboxed Linux VM via SSH",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"command": map[string]any{
							"type":        "string",
							"description": "The shell command to execute on the remote VM",
						},
						"timeout_seconds": map[string]any{
							"type":        "integer",
							"description": "Maximum time to wait for the command to complete in seconds (default: 30)",
						},
					},
					"required": []string{"command"},
				},
			},
		},
	}
}

// ExecuteTool executes the exec tool by name with the given raw JSON arguments.
func (c *Client) ExecuteTool(ctx context.Context, toolName, rawArgs string) (string, error) {
	switch toolName {
	case "exec":
		return c.run(ctx, rawArgs)
	default:
		return "", fmt.Errorf("unknown exec tool: %s", toolName)
	}
}

// run parses the arguments, builds an SSH command, executes it, and returns
// a JSON object with stdout, stderr, and exit_code.
func (c *Client) run(ctx context.Context, rawArgs string) (string, error) {
	args, err := parseExecArgs(rawArgs)
	if err != nil {
		return "", err
	}

	timeoutSec := args.TimeoutSeconds
	if timeoutSec == 0 {
		timeoutSec = 30
	}
	timeout := time.Duration(timeoutSec) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	sshArgs := []string{
		"-i", c.config.KeyPath,
		"-p", strconv.Itoa(c.config.Port),
		"-o", "StrictHostKeyChecking=no",
		"-o", "BatchMode=yes",
		fmt.Sprintf("%s@%s", c.config.User, c.config.Host),
		args.Command,
	}

	cmd := exec.CommandContext(ctx, "ssh", sshArgs...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	exitCode := 0
	err = cmd.Run()
	if err != nil {
		// Extract exit code from the error if possible
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				exitCode = status.ExitStatus()
			} else {
				exitCode = exitErr.ExitCode()
			}
		} else if ctx.Err() == context.DeadlineExceeded {
			// Command timed out
			result, _ := json.Marshal(map[string]any{
				"stdout":    stdout.String(),
				"stderr":    stderr.String(),
				"exit_code": -1,
				"error":     fmt.Sprintf("command timed out after %d seconds", timeoutSec),
			})
			return string(result), nil
		} else {
			return "", fmt.Errorf("failed to execute ssh command: %w", err)
		}
	}

	result, err := json.Marshal(map[string]any{
		"stdout":    stdout.String(),
		"stderr":    stderr.String(),
		"exit_code": exitCode,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal exec result: %w", err)
	}
	return string(result), nil
}

// parseExecArgs parses and validates the raw JSON arguments for the exec tool.
func parseExecArgs(rawArgs string) (execArgs, error) {
	var args execArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return execArgs{}, fmt.Errorf("invalid exec args: %w", err)
	}
	if strings.TrimSpace(args.Command) == "" {
		return execArgs{}, errors.New("command is required")
	}
	return args, nil
}
