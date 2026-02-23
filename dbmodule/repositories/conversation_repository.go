package repositories

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ConversationRepository struct {
	pool *pgxpool.Pool
}

func NewConversationRepository(pool *pgxpool.Pool) *ConversationRepository {
	return &ConversationRepository{pool: pool}
}

// CreateConversation starts a new conversation and returns its ID
func (r *ConversationRepository) CreateConversation(ctx context.Context, metadata map[string]interface{}) (uuid.UUID, error) {
	var id uuid.UUID
	metadataBytes, _ := json.Marshal(metadata)
	
	err := r.pool.QueryRow(ctx,
		"INSERT INTO conversations (metadata) VALUES ($1) RETURNING id",
		metadataBytes,
	).Scan(&id)
	
	return id, err
}

// FinishConversation marks a conversation as finished
func (r *ConversationRepository) FinishConversation(ctx context.Context, conversationID uuid.UUID, finishReason string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE conversations SET finished_at = $1, finish_reason = $2 WHERE id = $3",
		time.Now().UTC(), finishReason, conversationID,
	)
	return err
}

// UpdateSummary updates the summary field of a conversation
func (r *ConversationRepository) UpdateSummary(ctx context.Context, conversationID uuid.UUID, summary string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE conversations SET summary = $1 WHERE id = $2",
		summary, conversationID,
	)
	return err
}

// InsertChatMessage records a chat message
func (r *ConversationRepository) InsertChatMessage(ctx context.Context, conversationID uuid.UUID, role, content, finishReason, model string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx,
		"INSERT INTO chat_messages (conversation_id, role, content, finish_reason, model) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		conversationID, role, content, finishReason, model,
	).Scan(&id)
	return id, err
}

// InsertToolCall records a tool call
func (r *ConversationRepository) InsertToolCall(ctx context.Context, conversationID, messageID uuid.UUID, toolCallID, toolName string, arguments map[string]interface{}) (uuid.UUID, error) {
	var id uuid.UUID
	argsBytes, _ := json.Marshal(arguments)
	
	err := r.pool.QueryRow(ctx,
		"INSERT INTO tool_calls (conversation_id, message_id, tool_call_id, tool_name, arguments) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		conversationID, messageID, toolCallID, toolName, argsBytes,
	).Scan(&id)
	return id, err
}

// UpdateToolCallResult updates a tool call with its response or error
func (r *ConversationRepository) UpdateToolCallResult(ctx context.Context, toolCallID uuid.UUID, response map[string]interface{}, errMsg string) error {
	responseBytes, _ := json.Marshal(response)
	
	_, err := r.pool.Exec(ctx,
		"UPDATE tool_calls SET response = $1, error = $2, completed_at = $3 WHERE id = $4",
		responseBytes, errMsg, time.Now().UTC(), toolCallID,
	)
	return err
}

// InsertReasoning records reasoning content
func (r *ConversationRepository) InsertReasoning(ctx context.Context, conversationID, messageID uuid.UUID, reasoningContent string) error {
	_, err := r.pool.Exec(ctx,
		"INSERT INTO reasoning_logs (conversation_id, message_id, reasoning_content) VALUES ($1, $2, $3)",
		conversationID, messageID, reasoningContent,
	)
	return err
}

// GetRecentConversations retrieves recent conversations with messages
func (r *ConversationRepository) GetRecentConversations(ctx context.Context, limit int) ([]map[string]interface{}, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT c.id, c.started_at, c.finished_at, c.finish_reason, c.metadata,
		        json_agg(json_build_object(
					'id', m.id,
					'role', m.role,
					'content', m.content,
					'finish_reason', m.finish_reason,
					'model', m.model,
					'created_at', m.created_at
				) ORDER BY m.created_at) as messages
		 FROM conversations c
		 LEFT JOIN chat_messages m ON c.id = m.conversation_id
		 GROUP BY c.id
		 ORDER BY c.started_at DESC
		 LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var id uuid.UUID
		var startedAt, finishedAt time.Time
		var finishReason string
		var metadata []byte
		var messages []byte
		
		err := rows.Scan(&id, &startedAt, &finishedAt, &finishReason, &metadata, &messages)
		if err != nil {
			return nil, err
		}
		
		results = append(results, map[string]interface{}{
			"id":            id,
			"started_at":    startedAt,
			"finished_at":   finishedAt,
			"finish_reason": finishReason,
			"metadata":      json.RawMessage(metadata),
			"messages":      json.RawMessage(messages),
		})
	}
	
	return results, rows.Err()
}

// GetConversationToolCalls retrieves all tool calls for a conversation
func (r *ConversationRepository) GetConversationToolCalls(ctx context.Context, conversationID uuid.UUID) ([]map[string]interface{}, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tool_call_id, tool_name, arguments, response, error, started_at, completed_at
		 FROM tool_calls
		 WHERE conversation_id = $1
		 ORDER BY started_at`,
		conversationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var id uuid.UUID
		var toolCallID, toolName string
		var arguments, response []byte
		var errorMsg string
		var startedAt, completedAt time.Time
		
		err := rows.Scan(&id, &toolCallID, &toolName, &arguments, &response, &errorMsg, &startedAt, &completedAt)
		if err != nil {
			return nil, err
		}
		
		results = append(results, map[string]interface{}{
			"id":            id,
			"tool_call_id":  toolCallID,
			"tool_name":     toolName,
			"arguments":     json.RawMessage(arguments),
			"response":      json.RawMessage(response),
			"error":         errorMsg,
			"started_at":    startedAt,
			"completed_at":  completedAt,
		})
	}
	
	return results, rows.Err()
}

// GetConversationReasoning retrieves all reasoning for a conversation
func (r *ConversationRepository) GetConversationReasoning(ctx context.Context, conversationID uuid.UUID) ([]map[string]interface{}, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT r.id, r.message_id, r.reasoning_content, r.created_at, m.role, m.content
		 FROM reasoning_logs r
		 JOIN chat_messages m ON r.message_id = m.id
		 WHERE r.conversation_id = $1
		 ORDER BY r.created_at`,
		conversationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var id, messageID uuid.UUID
		var reasoningContent string
		var createdAt time.Time
		var role, content string
		
		err := rows.Scan(&id, &messageID, &reasoningContent, &createdAt, &role, &content)
		if err != nil {
			return nil, err
		}
		
		results = append(results, map[string]interface{}{
			"id":               id,
			"message_id":       messageID,
			"reasoning_content": reasoningContent,
			"created_at":       createdAt,
			"message_role":     role,
			"message_content":  content,
		})
	}
	
	return results, rows.Err()
}
