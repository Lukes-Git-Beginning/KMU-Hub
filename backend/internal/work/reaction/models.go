package reaction

import (
	"time"

	"github.com/google/uuid"
)

// Reaction represents a single emoji reaction on a chat message
type Reaction struct {
	MessageID uuid.UUID `json:"message_id"`
	UserID    uuid.UUID `json:"user_id"`
	Emoji     string    `json:"emoji"`
	CreatedAt time.Time `json:"created_at"`
}

// ReactionWithUser includes denormalized user info for API responses
type ReactionWithUser struct {
	Reaction
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// ReactionSummary aggregates reactions per emoji for a message
type ReactionSummary struct {
	MessageID          uuid.UUID   `json:"message_id"`
	Emoji              string      `json:"emoji"`
	Count              int         `json:"count"`
	UserIDs            []uuid.UUID `json:"user_ids"`
	CurrentUserReacted bool        `json:"current_user_reacted"`
}
