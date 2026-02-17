package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"teine-andres/dbmodule"
	"teine-andres/matrixmodule"
)

const hardcodedPrompt = "You are an AI agent, help me."

const (
	defaultModel   = "openai/o3-mini"
	defaultBaseURL = "https://openrouter.ai/api/v1"
)

type message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type toolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function toolFunction `json:"function"`
}

type toolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type tool struct {
	Type     string   `json:"type"`
	Function toolSpec `json:"function"`
}

type toolSpec struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type reasoning struct {
	Effort string `json:"effort"`
}

type chatRequest struct {
	Model      string      `json:"model"`
	Messages   []message   `json:"messages"`
	Tools      []tool      `json:"tools,omitempty"`
	ToolChoice interface{} `json:"tool_choice,omitempty"`
	Reasoning  *reasoning  `json:"reasoning,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message message `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type dbArgs struct {
	Query string `json:"query"`
}

func main() {
	apiKey := strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY"))
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "OPENROUTER_API_KEY is required")
		os.Exit(1)
	}

	model := strings.TrimSpace(os.Getenv("OPENROUTER_MODEL"))
	if model == "" {
		model = defaultModel
	}

	baseURL := strings.TrimSpace(os.Getenv("OPENROUTER_BASE_URL"))
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	loopLimit := parseLoopLimit(os.Getenv("TOOL_LOOP_LIMIT"))

	ctx := context.Background()
	httpClient := &http.Client{Timeout: 30 * time.Second}

	pool, cleanup, err := dbmodule.NewPool(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to create db pool:", err)
		os.Exit(1)
	}
	if cleanup != nil {
		defer cleanup()
	}

	messages := []message{
		{
			Role:    "user",
			Content: hardcodedPrompt,
		},
	}

	tools := buildTools()

	for i := 0; i < loopLimit; i++ {
		respMsg, err := callChat(ctx, httpClient, apiKey, baseURL, chatRequest{
			Model:      model,
			Messages:   messages,
			Tools:      tools,
			ToolChoice: "auto",
			Reasoning:  &reasoning{Effort: "medium"},
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, "chat error:", err)
			os.Exit(1)
		}

		messages = append(messages, respMsg)

		if len(respMsg.ToolCalls) == 0 {
			fmt.Println(respMsg.Content)
			return
		}

		for _, call := range respMsg.ToolCalls {
			result := executeTool(ctx, httpClient, pool, call)
			messages = append(messages, message{
				Role:       "tool",
				ToolCallID: call.ID,
				Content:    result,
			})
		}
	}

	fmt.Fprintln(os.Stderr, "tool loop exceeded limit")
	os.Exit(1)
}

func parseLoopLimit(value string) int {
	const defaultLimit = 12
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return defaultLimit
	}
	limit, err := strconv.Atoi(trimmed)
	if err != nil || limit <= 0 {
		return defaultLimit
	}
	return limit
}

func buildTools() []tool {
	return []tool{
		{
			Type: "function",
			Function: toolSpec{
				Name:        "db_read",
				Description: "Run a read-only SQL query against the PostgreSQL database",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "SQL query string",
						},
					},
					"required": []string{"query"},
				},
			},
		},
		{
			Type: "function",
			Function: toolSpec{
				Name:        "db_modify",
				Description: "Run a write SQL query against the PostgreSQL database",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "SQL query string",
						},
					},
					"required": []string{"query"},
				},
			},
		},
		{
			Type: "function",
			Function: toolSpec{
				Name:        "matrix_write",
				Description: "Send a message to a Matrix room",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"room_id": map[string]interface{}{
							"type":        "string",
							"description": "Matrix room ID",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Message text to send",
						},
					},
					"required": []string{"room_id", "message"},
				},
			},
		},
		{
			Type: "function",
			Function: toolSpec{
				Name:        "matrix_read",
				Description: "Read messages from a Matrix room",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"room_id": map[string]interface{}{
							"type":        "string",
							"description": "Matrix room ID",
						},
						"limit": map[string]interface{}{
							"type":        "integer",
							"description": "Maximum number of events to return",
						},
						"from": map[string]interface{}{
							"type":        "string",
							"description": "Pagination token to start from",
						},
						"direction": map[string]interface{}{
							"type":        "string",
							"description": "Direction: b (backwards) or f (forwards)",
						},
					},
					"required": []string{"room_id"},
				},
			},
		},
	}
}

func callChat(ctx context.Context, client *http.Client, apiKey, baseURL string, payload chatRequest) (message, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return message{}, err
	}

	endpoint := strings.TrimRight(baseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return message{}, err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return message{}, err
	}
	defer resp.Body.Close()

	var out chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return message{}, err
	}
	if out.Error != nil {
		return message{}, errors.New(out.Error.Message)
	}
	if len(out.Choices) == 0 {
		return message{}, errors.New("no choices returned")
	}

	return out.Choices[0].Message, nil
}

func executeTool(ctx context.Context, client *http.Client, pool *dbmodule.Pool, call toolCall) string {
	switch call.Function.Name {
	case "db_read":
		return runReadTool(ctx, pool, call.Function.Arguments)
	case "db_modify":
		return runModifyTool(ctx, pool, call.Function.Arguments)
	case "matrix_write":
		return runMatrixWriteTool(ctx, client, call.Function.Arguments)
	case "matrix_read":
		return runMatrixReadTool(ctx, client, call.Function.Arguments)
	default:
		return toolError(fmt.Sprintf("unknown tool: %s", call.Function.Name))
	}
}

func runReadTool(ctx context.Context, pool *dbmodule.Pool, rawArgs string) string {
	query, err := parseQueryArgs(rawArgs, "read")
	if err != nil {
		return toolError(err.Error())
	}
	result, err := dbmodule.Read(ctx, pool, query)
	if err != nil {
		return toolError(err.Error())
	}
	return result
}

func runModifyTool(ctx context.Context, pool *dbmodule.Pool, rawArgs string) string {
	query, err := parseQueryArgs(rawArgs, "modify")
	if err != nil {
		return toolError(err.Error())
	}
	result, err := dbmodule.Modify(ctx, pool, query)
	if err != nil {
		return toolError(err.Error())
	}
	return result
}

func parseQueryArgs(rawArgs, toolName string) (string, error) {
	var args dbArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return "", fmt.Errorf("invalid %s args: %w", toolName, err)
	}
	if strings.TrimSpace(args.Query) == "" {
		return "", errors.New("query is required")
	}
	return strings.TrimSpace(args.Query), nil
}

func runMatrixWriteTool(ctx context.Context, client *http.Client, rawArgs string) string {
	baseURL := strings.TrimSpace(os.Getenv("MATRIX_BASE_URL"))
	if baseURL == "" {
		return toolError("MATRIX_BASE_URL is not configured")
	}
	token := strings.TrimSpace(os.Getenv("MATRIX_ACCESS_TOKEN"))
	if token == "" {
		return toolError("MATRIX_ACCESS_TOKEN is not configured")
	}

	result, err := matrixmodule.Write(ctx, client, baseURL, token, rawArgs)
	if err != nil {
		return toolError(err.Error())
	}
	return result
}

func runMatrixReadTool(ctx context.Context, client *http.Client, rawArgs string) string {
	baseURL := strings.TrimSpace(os.Getenv("MATRIX_BASE_URL"))
	if baseURL == "" {
		return toolError("MATRIX_BASE_URL is not configured")
	}
	token := strings.TrimSpace(os.Getenv("MATRIX_ACCESS_TOKEN"))
	if token == "" {
		return toolError("MATRIX_ACCESS_TOKEN is not configured")
	}

	result, err := matrixmodule.Read(ctx, client, baseURL, token, rawArgs)
	if err != nil {
		return toolError(err.Error())
	}
	return result
}

func toolError(msg string) string {
	payload, _ := json.Marshal(map[string]string{"error": msg})
	return string(payload)
}
