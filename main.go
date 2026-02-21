package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"teine-andres/dbmodule"
	"teine-andres/dbmodule/repositories"
	"teine-andres/execmodule"
	"teine-andres/matrixmodule"
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
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type reasoning struct {
	Effort string `json:"effort"`
}

type chatRequest struct {
	Model      string     `json:"model"`
	Messages   []message  `json:"messages"`
	Tools      []tool     `json:"tools,omitempty"`
	ToolChoice any        `json:"tool_choice,omitempty"`
	Reasoning  *reasoning `json:"reasoning,omitempty"`
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

func syncPromptsFromFiles(ctx context.Context, promptRepo *repositories.PromptRepository) error {
	promptsDir := "prompts"

	entries, err := os.ReadDir(promptsDir)
	if err != nil {
		return fmt.Errorf("failed to read prompts directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		title := strings.TrimSuffix(entry.Name(), ".md")
		filePath := filepath.Join(promptsDir, entry.Name())

		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", filePath, err)
		}

		if err := promptRepo.UpsertIdentityPrompt(ctx, title, string(content)); err != nil {
			return fmt.Errorf("failed to upsert prompt %s: %w", title, err)
		}

		fmt.Printf("Synced prompt: %s\n", title)
	}

	return nil
}

func getRequiredEnv(key string) string {
	out := os.Getenv(key)
	if out == "" {
		panic(fmt.Sprint("Required environment variable \"%s\" not defined.", key))
	}
	return out
}

func getEnvNumber(key string, defaultValue int) int {
	str := os.Getenv(key)
	if str == "" {
		return defaultValue
	}
	out, err := strconv.Atoi(str)
	if err != nil {
		panic(fmt.Sprint("Cannot convert value of \"%s\": \"%s\" to an integer.", key, str))
	}
	return out
}

func getRequiredSystemCredential(repo *repositories.CredentialRepository, ctx context.Context, key string) string {
	secret, err := repo.GetSystemCredential(ctx, "OPENROUTER_API_KEY")
	if err != nil {
		panic(fmt.Sprintf("Error getting system credential %s: %w", key, err))
	}
	return secret
}

func main() {
	ctx := context.Background()
	httpClient := &http.Client{Timeout: 120 * time.Second}

	systemPromptVariables := make(map[string]string)

	baseDatabaseURL := getRequiredEnv("DATABASE_URL")
	agentPassword := getRequiredEnv("DATABASE_PASSWORD_AGENT")
	ownerPassword := getRequiredEnv("DATABASE_PASSWORD_OWNER")
	systemPromptVariables["ownerFirstName"] = getRequiredEnv("OWNER_FIRST_NAME")
	systemPromptVariables["ownerMatrixId"] = getRequiredEnv("MATRIX_OWNER_ID")
	loopLimit := getEnvNumber("TOOL_LOOP_LIMIT", 12)
	systemPromptVariables["loopLimit"] = strconv.Itoa(loopLimit)

	execClient := execmodule.NewClient(execmodule.Config{
		Host:    getRequiredEnv("EXEC_SSH_HOST"),
		Port:    getEnvNumber("EXEC_SSH_PORT", 22),
		User:    getRequiredEnv("EXEC_SSH_USER"),
		KeyPath: getRequiredEnv("EXEC_SSH_KEY_PATH"),
	})

	dualPool, cleanup, err := dbmodule.NewPool(ctx, baseDatabaseURL, agentPassword, ownerPassword)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to create DB pools:", err)
		os.Exit(1)
	}
	if cleanup != nil {
		defer cleanup()
	}

	credRepo := repositories.NewCredentialRepository(dualPool.Owner)
	promptRepo := repositories.NewPromptRepository(dualPool.Owner)
	syncRepo := repositories.NewSyncStateRepository(dualPool.Owner)
	taskRepo := repositories.NewTaskRepository(dualPool.Owner)

	if err := syncPromptsFromFiles(ctx, promptRepo); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to sync prompts from files:", err)
		os.Exit(1)
	}

	apiKey := getRequiredSystemCredential(credRepo, ctx, "OPENROUTER_API_KEY")
	model := getRequiredSystemCredential(credRepo, ctx, "OPENROUTER_MODEL")
	matrixBaseURL := getRequiredSystemCredential(credRepo, ctx, "MATRIX_BASE_URL")
	matrixToken := getRequiredSystemCredential(credRepo, ctx, "MATRIX_ACCESS_TOKEN")

	matrixClient := matrixmodule.NewClient(httpClient, matrixBaseURL, matrixToken)
	whoamiResp, err := matrixClient.Whoami(ctx)
	if err != nil {
		panic(fmt.Sprint(os.Stderr, "failed to get Matrix user ID: ", err))
	}
	agentMatrixId := whoamiResp.UserID
	systemPromptVariables["agentMatrixId"] = agentMatrixId

	tools := buildTools()
	lastHourlyLoop := time.Now()

MainLoop:
	for {
		time.Sleep(1 * time.Second)
		initialPrompt, err := loadInitialPrompt(ctx, promptRepo)
		nextBatch, err := syncRepo.GetNextBatch(ctx)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to get Matrix sync token:", err)
			continue
		}

		syncResp, err := matrixClient.Sync(ctx, nextBatch)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to run Matrix sync:", err)
			continue
		}

		if syncResp.Rooms.Invite != nil {
			for roomID := range syncResp.Rooms.Invite {
				if err := matrixClient.JoinRoom(ctx, roomID); err != nil {
					fmt.Fprintf(os.Stderr, "failed to join room %s: %v\n", roomID, err)
					continue MainLoop
				}
				fmt.Printf("Joined room %s\n", roomID)
			}
		}

		if err := syncRepo.UpdateNextBatch(ctx, syncResp.NextBatch); err != nil {
			fmt.Fprintln(os.Stderr, "failed to update sync token:", err)
			continue
		}

		isHourlyLoop := time.Since(lastHourlyLoop) >= 1*time.Hour

		statuses := []string{"pending", "in_progress"}

		if isHourlyLoop {
			statuses = append(statuses, "blocked")
			lastHourlyLoop = time.Now()
		}

		var contents []string

		tasks, err := taskRepo.GetTasksByStatuses(ctx, statuses)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to get tasks:", err)
			continue
		} else if len(tasks) > 0 {
			contents = append(contents, "Here is the information about current tasks:\n")
			taskMessages := []string{}
			for _, task := range tasks {
				desc := ""
				if task.Description != nil {
					desc = *task.Description
				}
				status := ""
				if task.Status != nil {
					status = *task.Status
				}
				taskMap := map[string]string{
					"id":          task.ID.String(),
					"title":       task.Title,
					"description": desc,
					"status":      status,
				}
				if task.ParentID != nil {
					taskMap["parent_task_id"] = task.ParentID.String()
				}
				taskJSON, _ := json.Marshal(taskMap)
				taskMessages = append(taskMessages, string(taskJSON))
			}
			contents = append(contents, "["+strings.Join(taskMessages, ", ")+"]")
		}

		if syncResp.Rooms.Join != nil {
			for roomID, joinedRoom := range syncResp.Rooms.Join {
				events := []string{}
				for _, event := range joinedRoom.Timeline.Events {
					var eventMap map[string]any
					json.Unmarshal(event, &eventMap)
					if eventMap["sender"] != agentMatrixId {
						events = append(events, string(event))
					}
				}
				if len(events) > 0 {
					contents = append(contents, fmt.Sprintf("Here are the events that happened in the Matrix room \"%s\" since the last sync:\n[%s]", roomID, strings.Join(events, ", ")))
				}
			}
		}

		if len(contents) == 0 {
			fmt.Println("No messages or tasks, waiting...")
			continue
		}

		initialPrompt = append(initialPrompt, strings.Join(contents, "\n\n"))
		messages := []message{{
			Role:    "system",
			Content: strings.Join(initialPrompt, "\n\n"),
		}}
		messages = append(messages, message{
			Role:    "user",
			Content: "Help your owner achieve what he wants",
		})

		fmt.Println("%v", messages)

		for range loopLimit {

			respMsg, err := callChat(ctx, httpClient, apiKey, chatRequest{
				Model:      model,
				Messages:   messages,
				Tools:      tools,
				ToolChoice: "auto",
				Reasoning:  &reasoning{Effort: "medium"},
			})
			if err != nil {
				fmt.Fprintln(os.Stderr, "chat error:", err)
				continue MainLoop
			}

			messages = append(messages, respMsg)

			if len(respMsg.ToolCalls) == 0 {
				fmt.Println(respMsg.Content)
				break
			}

			for _, call := range respMsg.ToolCalls {
				fmt.Println("TOOL CALL: %v", call)
				result := executeTool(ctx, httpClient, dualPool.Agent, credRepo, matrixClient, execClient, call)
				fmt.Println("TOOL CALL RESULT: %v", result)
				messages = append(messages, message{
					Role:       "tool",
					ToolCallID: call.ID,
					Content:    result,
				})
			}
		}
	}
}

func loadInitialPrompt(ctx context.Context, promptRepo *repositories.PromptRepository) ([]string, error) {
	identityPrompts, err := promptRepo.GetIdentityPrompts(ctx)
	if err != nil {
		return []string{}, err
	}
	selfPrompts, err := promptRepo.GetSelfPrompts(ctx)
	if err != nil {
		return []string{}, err
	}

	var allPrompts []string
	for _, p := range identityPrompts {
		allPrompts = append(allPrompts, p.Prompt)
	}
	for _, p := range selfPrompts {
		allPrompts = append(allPrompts, p.Prompt)
	}

	return allPrompts, nil
}

func buildTools() []tool {
	dbTools := []tool{
		{
			Type: "function",
			Function: toolSpec{
				Name:        "db_read",
				Description: "Run a read-only SQL query against the PostgreSQL database",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"query": map[string]any{
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
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"query": map[string]any{
							"type":        "string",
							"description": "SQL query string",
						},
					},
					"required": []string{"query"},
				},
			},
		},
	}

	// Convert Matrix tools to match main's tool type
	matrixToolSpecs := matrixmodule.GetToolSpecs()
	matrixTools := make([]tool, len(matrixToolSpecs))
	for i, mt := range matrixToolSpecs {
		matrixTools[i] = tool{
			Type:     mt.Type,
			Function: toolSpec(mt.Function),
		}
	}

	// Convert exec tools to match main's tool type
	execToolSpecs := execmodule.GetToolSpecs()
	execTools := make([]tool, len(execToolSpecs))
	for i, et := range execToolSpecs {
		execTools[i] = tool{
			Type:     et.Type,
			Function: toolSpec(et.Function),
		}
	}

	allTools := append(dbTools, matrixTools...)
	return append(allTools, execTools...)
}

func callChat(ctx context.Context, client *http.Client, apiKey string, payload chatRequest) (message, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return message{}, err
	}

	endpoint := "https://openrouter.ai/api/v1/chat/completions"
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

func executeTool(ctx context.Context, client *http.Client, pool *dbmodule.Pool, credRepo *repositories.CredentialRepository, matrixClient *matrixmodule.Client, execClient *execmodule.Client, call toolCall) string {
	switch call.Function.Name {
	case "db_read":
		return runReadTool(ctx, pool, call.Function.Arguments)
	case "db_modify":
		return runModifyTool(ctx, pool, call.Function.Arguments)
	case "matrix_write", "matrix_read":
		result, err := matrixClient.ExecuteTool(ctx, call.Function.Name, call.Function.Arguments)
		if err != nil {
			return toolError(err.Error())
		}
		return result
	case "exec":
		result, err := execClient.ExecuteTool(ctx, call.Function.Name, call.Function.Arguments)
		if err != nil {
			return toolError(err.Error())
		}
		return result
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

func toolError(msg string) string {
	payload, _ := json.Marshal(map[string]string{"error": msg})
	return string(payload)
}
