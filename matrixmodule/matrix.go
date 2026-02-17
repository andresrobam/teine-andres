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

type writeArgs struct {
	RoomID  string `json:"room_id"`
	Message string `json:"message"`
}

type readArgs struct {
	RoomID    string `json:"room_id"`
	Limit     int    `json:"limit"`
	From      string `json:"from"`
	Direction string `json:"direction"`
}

func Write(ctx context.Context, client *http.Client, baseURL, token, rawArgs string) (string, error) {
	var args writeArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return "", fmt.Errorf("invalid matrix_write args: %w", err)
	}
	if strings.TrimSpace(args.RoomID) == "" {
		return "", errors.New("room_id is required")
	}
	if strings.TrimSpace(args.Message) == "" {
		return "", errors.New("message is required")
	}

	txnID := fmt.Sprintf("%d", time.Now().UnixNano())
	endpoint := strings.TrimRight(baseURL, "/") + "/_matrix/client/v3/rooms/" + url.PathEscape(args.RoomID) + "/send/m.room.message/" + txnID

	body, err := json.Marshal(map[string]string{
		"msgtype": "m.text",
		"body":    args.Message,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var out map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if errText, ok := out["error"].(string); ok {
		return "", errors.New(errText)
	}

	payload, err := json.Marshal(map[string]interface{}{
		"event_id": out["event_id"],
	})
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func Read(ctx context.Context, client *http.Client, baseURL, token, rawArgs string) (string, error) {
	var args readArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return "", fmt.Errorf("invalid matrix_read args: %w", err)
	}
	if strings.TrimSpace(args.RoomID) == "" {
		return "", errors.New("room_id is required")
	}

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

	endpoint := strings.TrimRight(baseURL, "/") + "/_matrix/client/v3/rooms/" + url.PathEscape(args.RoomID) + "/messages"
	if encoded := query.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var out map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if errText, ok := out["error"].(string); ok {
		return "", errors.New(errText)
	}

	payload, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}
