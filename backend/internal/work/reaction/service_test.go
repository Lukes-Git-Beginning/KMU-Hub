package reaction

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
)

// --- Mock Reaction Repository ---

type mockReactionRepo struct {
	reactions []Reaction
}

func newMockReactionRepo() *mockReactionRepo {
	return &mockReactionRepo{}
}

func (m *mockReactionRepo) AddReaction(_ context.Context, messageID, userID uuid.UUID, emoji string) error {
	// Check for duplicate (mimics ON CONFLICT DO NOTHING)
	for _, r := range m.reactions {
		if r.MessageID == messageID && r.UserID == userID && r.Emoji == emoji {
			return nil
		}
	}
	m.reactions = append(m.reactions, Reaction{
		MessageID: messageID,
		UserID:    userID,
		Emoji:     emoji,
		CreatedAt: time.Now(),
	})
	return nil
}

func (m *mockReactionRepo) RemoveReaction(_ context.Context, messageID, userID uuid.UUID, emoji string) error {
	var kept []Reaction
	for _, r := range m.reactions {
		if r.MessageID == messageID && r.UserID == userID && r.Emoji == emoji {
			continue
		}
		kept = append(kept, r)
	}
	m.reactions = kept
	return nil
}

func (m *mockReactionRepo) ReactionExists(_ context.Context, messageID, userID uuid.UUID, emoji string) (bool, error) {
	for _, r := range m.reactions {
		if r.MessageID == messageID && r.UserID == userID && r.Emoji == emoji {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockReactionRepo) ListReactions(_ context.Context, messageID uuid.UUID) ([]Reaction, error) {
	var result []Reaction
	for _, r := range m.reactions {
		if r.MessageID == messageID {
			result = append(result, r)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result, nil
}

func (m *mockReactionRepo) GetReactionSummary(_ context.Context, messageID, currentUserID uuid.UUID) ([]ReactionSummary, error) {
	type emojiGroup struct {
		emoji     string
		userIDs   []uuid.UUID
		firstSeen time.Time
	}
	groups := make(map[string]*emojiGroup)

	for _, r := range m.reactions {
		if r.MessageID != messageID {
			continue
		}
		g, ok := groups[r.Emoji]
		if !ok {
			g = &emojiGroup{emoji: r.Emoji, firstSeen: r.CreatedAt}
			groups[r.Emoji] = g
		}
		g.userIDs = append(g.userIDs, r.UserID)
		if r.CreatedAt.Before(g.firstSeen) {
			g.firstSeen = r.CreatedAt
		}
	}

	var summaries []ReactionSummary
	for _, g := range groups {
		currentReacted := false
		for _, uid := range g.userIDs {
			if uid == currentUserID {
				currentReacted = true
				break
			}
		}
		summaries = append(summaries, ReactionSummary{
			MessageID:          messageID,
			Emoji:              g.emoji,
			Count:              len(g.userIDs),
			UserIDs:            g.userIDs,
			CurrentUserReacted: currentReacted,
		})
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Emoji < summaries[j].Emoji
	})
	return summaries, nil
}

func (m *mockReactionRepo) GetBatchReactionSummary(_ context.Context, messageIDs []uuid.UUID, currentUserID uuid.UUID) (map[uuid.UUID][]ReactionSummary, error) {
	result := make(map[uuid.UUID][]ReactionSummary)
	idSet := make(map[uuid.UUID]bool)
	for _, id := range messageIDs {
		idSet[id] = true
	}

	type emojiGroup struct {
		emoji     string
		userIDs   []uuid.UUID
		firstSeen time.Time
	}

	// Group by message_id, then by emoji
	msgGroups := make(map[uuid.UUID]map[string]*emojiGroup)
	for _, r := range m.reactions {
		if !idSet[r.MessageID] {
			continue
		}
		if msgGroups[r.MessageID] == nil {
			msgGroups[r.MessageID] = make(map[string]*emojiGroup)
		}
		g, ok := msgGroups[r.MessageID][r.Emoji]
		if !ok {
			g = &emojiGroup{emoji: r.Emoji, firstSeen: r.CreatedAt}
			msgGroups[r.MessageID][r.Emoji] = g
		}
		g.userIDs = append(g.userIDs, r.UserID)
	}

	for msgID, groups := range msgGroups {
		for _, g := range groups {
			currentReacted := false
			for _, uid := range g.userIDs {
				if uid == currentUserID {
					currentReacted = true
					break
				}
			}
			result[msgID] = append(result[msgID], ReactionSummary{
				MessageID:          msgID,
				Emoji:              g.emoji,
				Count:              len(g.userIDs),
				UserIDs:            g.userIDs,
				CurrentUserReacted: currentReacted,
			})
		}
	}
	return result, nil
}

// --- Tests ---

func TestService_ToggleReaction_AddWhenNotExists(t *testing.T) {
	ctx := context.Background()
	repo := newMockReactionRepo()
	svc := NewService(repo)

	msgID := uuid.New()
	userID := uuid.New()

	result, err := svc.ToggleReaction(ctx, msgID, userID, "thumbsup")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result.Added {
		t.Error("expected Added=true when reaction did not exist")
	}
	if len(result.Reactions) != 1 {
		t.Fatalf("expected 1 reaction summary, got %d", len(result.Reactions))
	}
	if result.Reactions[0].Emoji != "thumbsup" {
		t.Errorf("expected emoji 'thumbsup', got %q", result.Reactions[0].Emoji)
	}
	if result.Reactions[0].Count != 1 {
		t.Errorf("expected count 1, got %d", result.Reactions[0].Count)
	}
	if !result.Reactions[0].CurrentUserReacted {
		t.Error("expected CurrentUserReacted=true")
	}
}

func TestService_ToggleReaction_RemoveWhenExists(t *testing.T) {
	ctx := context.Background()
	repo := newMockReactionRepo()
	svc := NewService(repo)

	msgID := uuid.New()
	userID := uuid.New()

	// Add first
	_, err := svc.ToggleReaction(ctx, msgID, userID, "heart")
	if err != nil {
		t.Fatalf("first toggle: %v", err)
	}

	// Toggle again should remove
	result, err := svc.ToggleReaction(ctx, msgID, userID, "heart")
	if err != nil {
		t.Fatalf("second toggle: %v", err)
	}
	if result.Added {
		t.Error("expected Added=false when reaction existed (should be removed)")
	}
	if len(result.Reactions) != 0 {
		t.Errorf("expected 0 reaction summaries after removal, got %d", len(result.Reactions))
	}
}

func TestService_ToggleReaction_ReturnsUpdatedSummary(t *testing.T) {
	ctx := context.Background()
	repo := newMockReactionRepo()
	svc := NewService(repo)

	msgID := uuid.New()
	user1 := uuid.New()
	user2 := uuid.New()

	// User1 adds thumbsup
	_, err := svc.ToggleReaction(ctx, msgID, user1, "thumbsup")
	if err != nil {
		t.Fatalf("user1 toggle: %v", err)
	}

	// User2 adds thumbsup (queried with user2 as current)
	result, err := svc.ToggleReaction(ctx, msgID, user2, "thumbsup")
	if err != nil {
		t.Fatalf("user2 toggle: %v", err)
	}
	if !result.Added {
		t.Error("expected Added=true")
	}

	// Find the thumbsup summary
	var found bool
	for _, s := range result.Reactions {
		if s.Emoji == "thumbsup" {
			found = true
			if s.Count != 2 {
				t.Errorf("expected count 2, got %d", s.Count)
			}
			if len(s.UserIDs) != 2 {
				t.Errorf("expected 2 user_ids, got %d", len(s.UserIDs))
			}
			if !s.CurrentUserReacted {
				t.Error("expected CurrentUserReacted=true for user2")
			}
		}
	}
	if !found {
		t.Error("thumbsup summary not found in result")
	}
}

func TestService_ToggleReaction_EmptyEmojiError(t *testing.T) {
	ctx := context.Background()
	repo := newMockReactionRepo()
	svc := NewService(repo)

	_, err := svc.ToggleReaction(ctx, uuid.New(), uuid.New(), "")
	if err != ErrEmojiRequired {
		t.Errorf("expected ErrEmojiRequired, got %v", err)
	}

	// Also test whitespace-only
	_, err = svc.ToggleReaction(ctx, uuid.New(), uuid.New(), "   ")
	if err != ErrEmojiRequired {
		t.Errorf("expected ErrEmojiRequired for whitespace emoji, got %v", err)
	}
}

func TestService_ToggleReaction_OversizedEmojiError(t *testing.T) {
	ctx := context.Background()
	repo := newMockReactionRepo()
	svc := NewService(repo)

	// Create a string larger than 32 bytes
	longEmoji := "abcdefghijklmnopqrstuvwxyz1234567" // 33 bytes
	_, err := svc.ToggleReaction(ctx, uuid.New(), uuid.New(), longEmoji)
	if err != ErrEmojiTooLong {
		t.Errorf("expected ErrEmojiTooLong, got %v", err)
	}
}

func TestService_ListReactions_OrderedResults(t *testing.T) {
	ctx := context.Background()
	repo := newMockReactionRepo()
	svc := NewService(repo)

	msgID := uuid.New()
	user1 := uuid.New()
	user2 := uuid.New()

	// Add reactions with slight time difference
	_ = repo.AddReaction(ctx, msgID, user1, "thumbsup")
	time.Sleep(time.Millisecond) // ensure ordering
	_ = repo.AddReaction(ctx, msgID, user2, "heart")

	reactions, err := svc.ListReactions(ctx, msgID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(reactions) != 2 {
		t.Fatalf("expected 2 reactions, got %d", len(reactions))
	}
	// First reaction should be chronologically first
	if reactions[0].UserID != user1 {
		t.Errorf("expected first reaction from user1, got %s", reactions[0].UserID)
	}
	if reactions[1].UserID != user2 {
		t.Errorf("expected second reaction from user2, got %s", reactions[1].UserID)
	}
}

func TestService_GetBatchReactionSummary_EmptySlice(t *testing.T) {
	ctx := context.Background()
	repo := newMockReactionRepo()
	svc := NewService(repo)

	result, err := svc.GetBatchReactionSummary(ctx, []uuid.UUID{}, uuid.New())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil map")
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %d entries", len(result))
	}
}

func TestService_GetBatchReactionSummary_MultipleMessages(t *testing.T) {
	ctx := context.Background()
	repo := newMockReactionRepo()
	svc := NewService(repo)

	msg1 := uuid.New()
	msg2 := uuid.New()
	msg3 := uuid.New() // no reactions
	user := uuid.New()
	otherUser := uuid.New()

	// Add reactions to msg1
	_ = repo.AddReaction(ctx, msg1, user, "thumbsup")
	_ = repo.AddReaction(ctx, msg1, otherUser, "thumbsup")

	// Add reaction to msg2
	_ = repo.AddReaction(ctx, msg2, user, "heart")

	result, err := svc.GetBatchReactionSummary(ctx, []uuid.UUID{msg1, msg2, msg3}, user)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// msg1 should have thumbsup with count 2
	summaries1, ok := result[msg1]
	if !ok {
		t.Fatal("expected summaries for msg1")
	}
	if len(summaries1) != 1 {
		t.Fatalf("expected 1 summary for msg1, got %d", len(summaries1))
	}
	if summaries1[0].Count != 2 {
		t.Errorf("expected count 2 for msg1 thumbsup, got %d", summaries1[0].Count)
	}
	if !summaries1[0].CurrentUserReacted {
		t.Error("expected current user reacted for msg1")
	}

	// msg2 should have heart with count 1
	summaries2, ok := result[msg2]
	if !ok {
		t.Fatal("expected summaries for msg2")
	}
	if len(summaries2) != 1 {
		t.Fatalf("expected 1 summary for msg2, got %d", len(summaries2))
	}
	if summaries2[0].Emoji != "heart" {
		t.Errorf("expected emoji 'heart', got %q", summaries2[0].Emoji)
	}

	// msg3 should have no entry (or empty)
	summaries3 := result[msg3]
	if len(summaries3) != 0 {
		t.Errorf("expected 0 summaries for msg3, got %d", len(summaries3))
	}
}

func TestService_GetReactionSummary_Direct(t *testing.T) {
	ctx := context.Background()
	repo := newMockReactionRepo()
	svc := NewService(repo)

	msgID := uuid.New()
	user := uuid.New()

	_ = repo.AddReaction(ctx, msgID, user, "fire")

	summaries, err := svc.GetReactionSummary(ctx, msgID, user)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}
	if summaries[0].Emoji != "fire" {
		t.Errorf("expected emoji 'fire', got %q", summaries[0].Emoji)
	}
	if summaries[0].Count != 1 {
		t.Errorf("expected count 1, got %d", summaries[0].Count)
	}
}

func TestValidateEmoji(t *testing.T) {
	tests := []struct {
		name    string
		emoji   string
		wantErr error
	}{
		{"valid single emoji", "thumbsup", nil},
		{"valid unicode emoji", "\U0001F600", nil},                           // grinning face
		{"valid flag emoji", "\U0001F1E8\U0001F1ED", nil},                   // CH flag (8 bytes)
		{"empty string", "", ErrEmojiRequired},
		{"whitespace only", "  \t ", ErrEmojiRequired},
		{"too long", "abcdefghijklmnopqrstuvwxyz1234567", ErrEmojiTooLong}, // 33 bytes
		{"exactly 32 bytes", "abcdefghijklmnopqrstuvwxyz123456", nil},       // 32 bytes
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEmoji(tt.emoji)
			if err != tt.wantErr {
				t.Errorf("validateEmoji(%q) = %v, want %v", tt.emoji, err, tt.wantErr)
			}
		})
	}
}
