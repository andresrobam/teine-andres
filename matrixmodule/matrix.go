package matrixmodule

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client provides Matrix API functionality with cached credentials
type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
}

// NewClient creates a new Matrix client with the provided credentials
func NewClient(httpClient *http.Client, baseURL, token string) *Client {
	return &Client{
		httpClient: httpClient,
		baseURL:    baseURL,
		token:      token,
	}
}

// writeArgs represents the arguments for the matrix_write tool
type writeArgs struct {
	RoomID  string `json:"room_id"`
	Message string `json:"message"`
}

// ReadArgs represents the arguments for reading messages from a Matrix room
type ReadArgs struct {
	RoomID    string `json:"room_id"`
	Limit     int    `json:"limit"`
	From      string `json:"from"`
	Direction string `json:"direction"`
}

// SyncResponse represents the response from the Matrix sync endpoint
type SyncResponse struct {
	NextBatch string `json:"next_batch"`
	Rooms     struct {
		Join   map[string]JoinedRoom `json:"join"`
		Invite map[string]any        `json:"invite"` // We only need the keys (Room IDs)
	} `json:"rooms"`
}

type JoinedRoom struct {
	Timeline struct {
		Events []json.RawMessage `json:"events"`
	} `json:"timeline"`
}

// WhoamiResponse represents the response from the Matrix whoami endpoint
type WhoamiResponse struct {
	UserID string `json:"user_id"`
}

// ToolSpec represents a tool specification for the AI model
type ToolSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// Tool represents a tool definition with type and spec
type Tool struct {
	Type     string   `json:"type"`
	Function ToolSpec `json:"function"`
}

// GetToolSpecs returns the tool specifications for Matrix operations
func GetToolSpecs() []Tool {
	return []Tool{
		{
			Type: "function",
			Function: ToolSpec{
				Name:        "matrix_write",
				Description: "Send a message to a Matrix room",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"room_id": map[string]any{
							"type":        "string",
							"description": "Matrix room ID",
						},
						"message": map[string]any{
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
			Function: ToolSpec{
				Name:        "matrix_read",
				Description: "Read messages from a Matrix room",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"room_id": map[string]any{
							"type":        "string",
							"description": "Matrix room ID",
						},
						"limit": map[string]any{
							"type":        "integer",
							"description": "Maximum number of events to return",
						},
						"from": map[string]any{
							"type":        "string",
							"description": "Pagination token to start from",
						},
						"direction": map[string]any{
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

// ExecuteTool executes a Matrix tool by name with the given arguments
func (c *Client) ExecuteTool(ctx context.Context, toolName, rawArgs string) (string, error) {
	switch toolName {
	case "matrix_write":
		return c.write(ctx, rawArgs)
	case "matrix_read":
		return c.readTool(ctx, rawArgs)
	default:
		return "", fmt.Errorf("unknown matrix tool: %s", toolName)
	}
}

// write sends a message to a Matrix room
func (c *Client) write(ctx context.Context, rawArgs string) (string, error) {
	args, err := parseWriteArgs(rawArgs)
	if err != nil {
		return "", err
	}

	txnID := fmt.Sprintf("%d", time.Now().UnixNano())
	endpoint := c.buildURL("/_matrix/client/v3/rooms/"+url.PathEscape(args.RoomID)+"/send/m.room.message/"+txnID, nil)

	body, err := json.Marshal(map[string]string{
		"msgtype": "m.text",
		"body":    args.Message,
	})
	if err != nil {
		return "", err
	}

	var result map[string]any
	if err := c.doPut(ctx, endpoint, body, &result); err != nil {
		return "", err
	}

	payload, err := json.Marshal(map[string]any{
		"event_id": result["event_id"],
	})
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

// readTool is the tool wrapper that parses JSON and calls Read
func (c *Client) readTool(ctx context.Context, rawArgs string) (string, error) {
	args, err := parseReadArgs(rawArgs)
	if err != nil {
		return "", err
	}

	result, err := c.Read(ctx, args)
	if err != nil {
		return "", err
	}

	payload, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

// Read reads messages from a Matrix room with the given arguments
func (c *Client) Read(ctx context.Context, args ReadArgs) (map[string]any, error) {
	query := url.Values{}
	if args.Limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", args.Limit))
	}
	if strings.TrimSpace(args.From) != "" {
		query.Set("from", strings.TrimSpace(args.From))
	}
	dir := strings.TrimSpace(args.Direction)
	if dir == "" {
		dir = "b"
	}
	query.Set("dir", dir)

	endpoint := c.buildURL("/_matrix/client/v3/rooms/"+url.PathEscape(args.RoomID)+"/messages", query)

	var result map[string]any
	if err := c.doGet(ctx, endpoint, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// parseWriteArgs parses and validates arguments for the write operation
func parseWriteArgs(rawArgs string) (writeArgs, error) {
	var args writeArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return writeArgs{}, fmt.Errorf("invalid matrix_write args: %w", err)
	}
	if strings.TrimSpace(args.RoomID) == "" {
		return writeArgs{}, errors.New("room_id is required")
	}
	if strings.TrimSpace(args.Message) == "" {
		return writeArgs{}, errors.New("message is required")
	}
	return args, nil
}

// parseReadArgs parses and validates arguments for the read operation
func parseReadArgs(rawArgs string) (ReadArgs, error) {
	var args ReadArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return ReadArgs{}, fmt.Errorf("invalid matrix_read args: %w", err)
	}
	if strings.TrimSpace(args.RoomID) == "" {
		return ReadArgs{}, errors.New("room_id is required")
	}
	return args, nil
}

// buildURL constructs a full URL from a path and query parameters
func (c *Client) buildURL(path string, query url.Values) string {
	endpoint := strings.TrimRight(c.baseURL, "/") + path
	if encoded := query.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}
	return endpoint
}

// doPost performs an authenticated POST request and decodes the response
func (c *Client) doPost(ctx context.Context, endpoint string, body []byte, result any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	return c.doRequest(req, result)
}

// doPut performs an authenticated PUT request and decodes the response
func (c *Client) doPut(ctx context.Context, endpoint string, body []byte, result any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	return c.doRequest(req, result)
}

// doGet performs an authenticated GET request and decodes the response
func (c *Client) doGet(ctx context.Context, endpoint string, result any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	return c.doRequest(req, result)
}

// doRequest executes an HTTP request and handles common response patterns
func (c *Client) doRequest(req *http.Request, result any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return err
	}

	if errText, ok := data["error"].(string); ok {
		return errors.New(errText)
	}

	if result != nil {
		// Copy data into the result by re-marshaling and unmarshaling
		jsonData, err := json.Marshal(data)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(jsonData, result); err != nil {
			return err
		}
	}

	return nil
}

// Sync performs a sync request to get unread messages since the last sync token
func (c *Client) Sync(ctx context.Context, since string) (*SyncResponse, error) {
	query := url.Values{}
	query.Set("timeout", "1")
	if since != "" {
		query.Set("since", since)
	}

	endpoint := c.buildURL("/_matrix/client/v3/sync", query)

	var result SyncResponse
	if err := c.doGet(ctx, endpoint, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) Whoami(ctx context.Context) (*WhoamiResponse, error) {
	endpoint := c.buildURL("/_matrix/client/v3/account/whoami", nil)

	var result WhoamiResponse
	if err := c.doGet(ctx, endpoint, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// JoinRoom joins a Matrix room by ID
func (c *Client) JoinRoom(ctx context.Context, roomID string) error {
	endpoint := c.buildURL("/_matrix/client/v3/rooms/"+url.PathEscape(roomID)+"/join", nil)
	var result map[string]any
	if err := c.doPost(ctx, endpoint, []byte("{}"), &result); err != nil {
		return err
	}
	return nil
}
