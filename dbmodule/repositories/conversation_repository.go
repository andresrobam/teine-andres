package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
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

func (r *ConversationRepository) CreateConversation(ctx context.Context, model string, endpoint string, metadata map[string]interface{}) (uuid.UUID, error) {
	if r.pool == nil {
		return uuid.UUID{}, errors.New("DATABASE_URL is not configured")
	}

	if metadata == nil {
		metadata = map[string]interface{}{}
	}

	payload, err := json.Marshal(metadata)
	if err != nil {
		return uuid.UUID{}, err
	}

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	id := uuid.New()
	_, err = r.pool.Exec(dbCtx,
		"INSERT INTO conversations (id, started_at, model, endpoint, metadata) VALUES ($1, $2, $3, $4, $5)",
		id, time.Now().UTC(), model, endpoint, payload)

	return id, err
}

func (r *ConversationRepository) InsertMessage(ctx context.Context, conversationID uuid.UUID, seq int, role string, content *string, reasoning *string, toolCalls []byte, toolCallID *string) error {
	if r.pool == nil {
		return errors.New("DATABASE_URL is not configured")
	}

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	id := uuid.New()
	_, err := r.pool.Exec(dbCtx,
		"INSERT INTO conversation_messages (id, conversation_id, seq, role, content, reasoning, tool_calls, tool_call_id, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)",
		id, conversationID, seq, role, content, reasoning, toolCalls, toolCallID, time.Now().UTC())

	return err
}

func (r *ConversationRepository) FinishConversation(ctx context.Context, conversationID uuid.UUID, finishReason string, errMsg string) error {
	if r.pool == nil {
		return errors.New("DATABASE_URL is not configured")
	}

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var reasonPtr *string
	if strings.TrimSpace(finishReason) != "" {
		reasonPtr = &finishReason
	}

	var errPtr *string
	if strings.TrimSpace(errMsg) != "" {
		errPtr = &errMsg
	}

	_, err := r.pool.Exec(dbCtx,
		"UPDATE conversations SET finish_reason = $1, error = $2, finished_at = $3 WHERE id = $4",
		reasonPtr, errPtr, time.Now().UTC(), conversationID)

	return err
}

// UpdateSummary updates the summary field of a conversation
func (r *ConversationRepository) UpdateSummary(ctx context.Context, conversationID uuid.UUID, summary string) error {
	if r.pool == nil {
		return errors.New("DATABASE_URL is not configured")
	}

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := r.pool.Exec(dbCtx,
		"UPDATE conversations SET summary = $1 WHERE id = $2",
		summary, conversationID,
	)
	return err
}
