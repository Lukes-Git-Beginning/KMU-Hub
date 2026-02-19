package message

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// subjectPrefixRe strips reply/forward prefixes for subject-based threading.
// Handles: Re:, Fwd:, AW: (German reply), WG: (German forward), plus Fw: variant.
var subjectPrefixRe = regexp.MustCompile(`(?i)^(re|fwd?|aw|wg)\s*:\s*`)

// subjectFallbackWindow is the time window for subject-based threading fallback.
const subjectFallbackWindow = 7 * 24 * time.Hour

// AssignThread assigns a thread ID to a message based on its headers.
//
// Algorithm:
// 1. Check InReplyTo -- look up message with matching message_id header
// 2. If found, use that message's thread_id
// 3. If not, check References array (last element first)
// 4. If found, use that message's thread_id
// 5. If still not found, try subject-based fallback (7-day window)
// 6. If nothing matches, generate new thread_id
func (s *Service) AssignThread(ctx context.Context, msg *models.EmailMessage) error {
	// Try header-based threading first
	if msg.InReplyTo != "" || len(msg.References) > 0 {
		threadID, err := s.repo.FindThreadByReferences(ctx, msg.AccountID, msg.References, msg.InReplyTo)
		if err == nil && threadID != uuid.Nil {
			msg.ThreadID = &threadID
			return s.repo.UpdateThreadID(ctx, msg.ID, threadID)
		}
	}

	// Subject-based fallback for broken clients
	normalizedSubject := NormalizeSubject(msg.Subject)
	if normalizedSubject != "" {
		since := msg.Date.Add(-subjectFallbackWindow)
		until := msg.Date.Add(subjectFallbackWindow)

		existing, err := s.repo.FindBySubjectAndParticipants(
			ctx, msg.AccountID, normalizedSubject, msg.FromEmail, since, until,
		)
		if err == nil && existing != nil && existing.ThreadID != nil {
			threadID := *existing.ThreadID
			msg.ThreadID = &threadID
			return s.repo.UpdateThreadID(ctx, msg.ID, threadID)
		}
	}

	// New conversation -- generate thread_id
	threadID := uuid.New()
	msg.ThreadID = &threadID
	return s.repo.UpdateThreadID(ctx, msg.ID, threadID)
}

// ReconstructThreads rebuilds thread assignments for a batch of messages.
// Used after initial sync or UIDVALIDITY reset.
func (s *Service) ReconstructThreads(ctx context.Context, messages []*models.EmailMessage) error {
	// Build a map of message_id -> thread_id for quick lookups
	messageIDToThread := make(map[string]uuid.UUID)

	for _, msg := range messages {
		// Try to find existing thread via references
		var threadID uuid.UUID
		found := false

		// Check InReplyTo
		if msg.InReplyTo != "" {
			if tid, ok := messageIDToThread[msg.InReplyTo]; ok {
				threadID = tid
				found = true
			}
		}

		// Check References (last first)
		if !found {
			for i := len(msg.References) - 1; i >= 0; i-- {
				if tid, ok := messageIDToThread[msg.References[i]]; ok {
					threadID = tid
					found = true
					break
				}
			}
		}

		// New thread
		if !found {
			threadID = uuid.New()
		}

		// Update message
		msg.ThreadID = &threadID
		if err := s.repo.UpdateThreadID(ctx, msg.ID, threadID); err != nil {
			slog.Warn("thread reconstruction: failed to update thread_id",
				"message_id", msg.ID,
				"error", err,
			)
			continue
		}

		// Register this message's ID for future lookups
		if msg.MessageID != "" {
			messageIDToThread[msg.MessageID] = threadID
		}
	}

	slog.Info("thread reconstruction complete", "messages", len(messages))
	return nil
}

// NormalizeSubject strips reply/forward prefixes and trims whitespace.
// Used for subject-based threading fallback.
func NormalizeSubject(subject string) string {
	normalized := subject
	for {
		stripped := subjectPrefixRe.ReplaceAllString(normalized, "")
		stripped = strings.TrimSpace(stripped)
		if stripped == normalized {
			break
		}
		normalized = stripped
	}
	return normalized
}
