package server

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/chat/bookmark"
	"github.com/kmuhub/kmuhub/internal/chat/file"
	"github.com/kmuhub/kmuhub/internal/chat/message"
	chatsearch "github.com/kmuhub/kmuhub/internal/chat/search"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/work/reaction"
	chatv1 "github.com/kmuhub/kmuhub/proto/chat/v1"
)

// ============================================================================
// stubChatSearchRepo -- implements chatsearch.Repository (in-memory, no DB)
// ============================================================================

type stubChatSearchRepo struct {
	userChannels   map[uuid.UUID][]uuid.UUID
	messageResults []models.ChatSearchResult
	fileResults    []models.ChatSearchResult
	channelIDsErr  error
}

func newStubSearchRepo() *stubChatSearchRepo {
	return &stubChatSearchRepo{userChannels: make(map[uuid.UUID][]uuid.UUID)}
}

func (r *stubChatSearchRepo) SearchMessages(_ context.Context, _ string, _ string, _ []uuid.UUID, _, _ int) ([]models.ChatSearchResult, int, error) {
	return r.messageResults, len(r.messageResults), nil
}

func (r *stubChatSearchRepo) SearchFiles(_ context.Context, _ string, _ []uuid.UUID, _, _ int) ([]models.ChatSearchResult, int, error) {
	return r.fileResults, len(r.fileResults), nil
}

func (r *stubChatSearchRepo) GetUserChannelIDs(_ context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	if r.channelIDsErr != nil {
		return nil, r.channelIDsErr
	}
	return r.userChannels[userID], nil
}

type stubChatDetector struct{}

func (stubChatDetector) Detect(_ string) string { return "de" }

// ============================================================================
// stubReactionRepo -- implements reaction.Repository (in-memory, no DB)
// ============================================================================

type stubReactionRepo struct {
	byMessage map[uuid.UUID][]reaction.Reaction
}

func newStubReactionRepo() *stubReactionRepo {
	return &stubReactionRepo{byMessage: make(map[uuid.UUID][]reaction.Reaction)}
}

func (r *stubReactionRepo) AddReaction(_ context.Context, messageID, userID uuid.UUID, emoji string) error {
	r.byMessage[messageID] = append(r.byMessage[messageID], reaction.Reaction{
		MessageID: messageID,
		UserID:    userID,
		Emoji:     emoji,
		CreatedAt: time.Now(),
	})
	return nil
}

func (r *stubReactionRepo) RemoveReaction(_ context.Context, messageID, userID uuid.UUID, emoji string) error {
	list := r.byMessage[messageID]
	for i, rx := range list {
		if rx.UserID == userID && rx.Emoji == emoji {
			r.byMessage[messageID] = append(list[:i], list[i+1:]...)
			break
		}
	}
	return nil
}

func (r *stubReactionRepo) ReactionExists(_ context.Context, messageID, userID uuid.UUID, emoji string) (bool, error) {
	for _, rx := range r.byMessage[messageID] {
		if rx.UserID == userID && rx.Emoji == emoji {
			return true, nil
		}
	}
	return false, nil
}

func (r *stubReactionRepo) ListReactions(_ context.Context, messageID uuid.UUID) ([]reaction.Reaction, error) {
	return r.byMessage[messageID], nil
}

func (r *stubReactionRepo) GetReactionSummary(_ context.Context, messageID, currentUserID uuid.UUID) ([]reaction.ReactionSummary, error) {
	return summarizeReactions(r.byMessage[messageID], currentUserID), nil
}

func (r *stubReactionRepo) GetBatchReactionSummary(_ context.Context, messageIDs []uuid.UUID, currentUserID uuid.UUID) (map[uuid.UUID][]reaction.ReactionSummary, error) {
	result := make(map[uuid.UUID][]reaction.ReactionSummary)
	for _, mid := range messageIDs {
		result[mid] = summarizeReactions(r.byMessage[mid], currentUserID)
	}
	return result, nil
}

func summarizeReactions(reactions []reaction.Reaction, currentUserID uuid.UUID) []reaction.ReactionSummary {
	byEmoji := make(map[string]*reaction.ReactionSummary)
	var order []string
	for _, rx := range reactions {
		s, ok := byEmoji[rx.Emoji]
		if !ok {
			s = &reaction.ReactionSummary{MessageID: rx.MessageID, Emoji: rx.Emoji}
			byEmoji[rx.Emoji] = s
			order = append(order, rx.Emoji)
		}
		s.Count++
		s.UserIDs = append(s.UserIDs, rx.UserID)
		if rx.UserID == currentUserID {
			s.CurrentUserReacted = true
		}
	}
	result := make([]reaction.ReactionSummary, 0, len(order))
	for _, emoji := range order {
		result = append(result, *byEmoji[emoji])
	}
	return result
}

// ============================================================================
// stubMessageReader -- implements bookmark.MessageReader (in-memory, no DB)
// ============================================================================

type stubMessageReader struct {
	messages map[uuid.UUID]*models.MessageWithSender
	// notFound/notMember let a test simulate a bookmark whose message has
	// become inaccessible (deleted or the user left the channel) without
	// removing it from the underlying repo -- see bookmark.Service.List's
	// skip behavior.
	notFound  map[uuid.UUID]bool
	notMember map[uuid.UUID]bool
}

func newStubMessageReader() *stubMessageReader {
	return &stubMessageReader{
		messages:  make(map[uuid.UUID]*models.MessageWithSender),
		notFound:  make(map[uuid.UUID]bool),
		notMember: make(map[uuid.UUID]bool),
	}
}

func (r *stubMessageReader) GetByID(_ context.Context, id, _, _ uuid.UUID) (*models.MessageWithSender, error) {
	if r.notFound[id] {
		return nil, message.ErrMessageNotFound
	}
	if r.notMember[id] {
		return nil, message.ErrNotChannelMember
	}
	m, ok := r.messages[id]
	if !ok {
		return nil, message.ErrMessageNotFound
	}
	return m, nil
}

// ============================================================================
// stubBookmarkRepo -- implements bookmark.Repository (in-memory, no DB)
// ============================================================================

type stubBookmarkRepo struct {
	// order holds message IDs per tenant+user, most-recently-bookmarked first.
	order map[string][]uuid.UUID
}

func newStubBookmarkRepo() *stubBookmarkRepo {
	return &stubBookmarkRepo{order: make(map[string][]uuid.UUID)}
}

func bookmarkKey(tenantID, userID uuid.UUID) string {
	return tenantID.String() + ":" + userID.String()
}

func (r *stubBookmarkRepo) Exists(_ context.Context, tenantID, userID, messageID uuid.UUID) (bool, error) {
	for _, id := range r.order[bookmarkKey(tenantID, userID)] {
		if id == messageID {
			return true, nil
		}
	}
	return false, nil
}

func (r *stubBookmarkRepo) Add(_ context.Context, tenantID, userID, messageID uuid.UUID) error {
	key := bookmarkKey(tenantID, userID)
	for _, id := range r.order[key] {
		if id == messageID {
			return nil // idempotent
		}
	}
	r.order[key] = append([]uuid.UUID{messageID}, r.order[key]...)
	return nil
}

func (r *stubBookmarkRepo) Remove(_ context.Context, tenantID, userID, messageID uuid.UUID) error {
	key := bookmarkKey(tenantID, userID)
	list := r.order[key]
	for i, id := range list {
		if id == messageID {
			r.order[key] = append(list[:i], list[i+1:]...)
			break
		}
	}
	return nil
}

func (r *stubBookmarkRepo) ListMessageIDs(_ context.Context, tenantID, userID uuid.UUID) ([]uuid.UUID, error) {
	return r.order[bookmarkKey(tenantID, userID)], nil
}

// ============================================================================
// Fixture
// ============================================================================

// chatFilesSearchReactionsFixture wires real file.Service/search.Service/
// reaction.Service/bookmark.Service instances against independent in-memory
// repos, mirroring how NewChatGRPCServer is wired in cmd/server.
// channelService/messageService stay nil -- this unit's scope is files,
// search, reactions and bookmarks only, none of which call into them.
type chatFilesSearchReactionsFixture struct {
	srv        *ChatGRPCServer
	fileRepo   *fileMockRepo
	fileStore  *fileMockStore
	searchRepo *stubChatSearchRepo
	reactRepo  *stubReactionRepo
	bmRepo     *stubBookmarkRepo
	msgReader  *stubMessageReader
}

func newChatFilesSearchReactionsFixture() *chatFilesSearchReactionsFixture {
	fileRepo := newFileMockRepo()
	fileStore := &fileMockStore{}
	fileSvc := file.NewService(fileRepo, fileStore, &fileMockScanner{}, &fileMockThumbGen{}, 50)

	searchRepo := newStubSearchRepo()
	searchSvc := chatsearch.NewService(searchRepo, stubChatDetector{})

	reactRepo := newStubReactionRepo()
	reactSvc := reaction.NewService(reactRepo)

	msgReader := newStubMessageReader()
	bmRepo := newStubBookmarkRepo()
	bmSvc := bookmark.NewService(bmRepo, msgReader)

	srv := NewChatGRPCServer(nil, nil, fileSvc, searchSvc, reactSvc, bmSvc)
	return &chatFilesSearchReactionsFixture{
		srv:        srv,
		fileRepo:   fileRepo,
		fileStore:  fileStore,
		searchRepo: searchRepo,
		reactRepo:  reactRepo,
		bmRepo:     bmRepo,
		msgReader:  msgReader,
	}
}

func fsrCtx(tenantID uuid.UUID) context.Context {
	return context.WithValue(context.Background(), middleware.TenantIDKey, tenantID.String())
}

// ============================================================================
// Files
// ============================================================================

func TestChatGetFileDownloadURL(t *testing.T) {
	t.Run("invalid file id", func(t *testing.T) {
		f := newChatFilesSearchReactionsFixture()
		_, err := f.srv.GetFileDownloadURL(context.Background(), &chatv1.GetFileDownloadURLRequest{
			FileId: "not-a-uuid", UserId: uuid.NewString(),
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid user id", func(t *testing.T) {
		f := newChatFilesSearchReactionsFixture()
		_, err := f.srv.GetFileDownloadURL(context.Background(), &chatv1.GetFileDownloadURLRequest{
			FileId: uuid.NewString(), UserId: "not-a-uuid",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("file not found", func(t *testing.T) {
		f := newChatFilesSearchReactionsFixture()
		_, err := f.srv.GetFileDownloadURL(context.Background(), &chatv1.GetFileDownloadURLRequest{
			FileId: uuid.NewString(), UserId: uuid.NewString(),
		})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("not a channel member", func(t *testing.T) {
		f := newChatFilesSearchReactionsFixture()
		channelID, userID, fileID := uuid.New(), uuid.New(), uuid.New()
		f.fileRepo.files[fileID] = &models.ChatFile{ID: fileID, ChannelID: channelID, StorageKey: "k"}

		_, err := f.srv.GetFileDownloadURL(context.Background(), &chatv1.GetFileDownloadURLRequest{
			FileId: fileID.String(), UserId: userID.String(),
		})
		requireGRPCCode(t, err, codes.PermissionDenied)
	})

	t.Run("success", func(t *testing.T) {
		f := newChatFilesSearchReactionsFixture()
		channelID, userID, fileID := uuid.New(), uuid.New(), uuid.New()
		f.fileRepo.files[fileID] = &models.ChatFile{ID: fileID, ChannelID: channelID, StorageKey: "k"}
		f.fileRepo.members[f.fileRepo.memberKey(channelID, userID)] = true

		resp, err := f.srv.GetFileDownloadURL(context.Background(), &chatv1.GetFileDownloadURLRequest{
			FileId: fileID.String(), UserId: userID.String(),
		})
		require.NoError(t, err)
		require.Equal(t, "https://example.com/presigned", resp.Url)
	})
}

func TestChatGetFileThumbnailURL(t *testing.T) {
	t.Run("invalid ids", func(t *testing.T) {
		f := newChatFilesSearchReactionsFixture()
		_, err := f.srv.GetFileThumbnailURL(context.Background(), &chatv1.GetFileThumbnailURLRequest{
			FileId: "bad", UserId: uuid.NewString(),
		})
		requireGRPCCode(t, err, codes.InvalidArgument)

		_, err = f.srv.GetFileThumbnailURL(context.Background(), &chatv1.GetFileThumbnailURLRequest{
			FileId: uuid.NewString(), UserId: "bad",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("no thumbnail generated yet", func(t *testing.T) {
		f := newChatFilesSearchReactionsFixture()
		channelID, userID, fileID := uuid.New(), uuid.New(), uuid.New()
		f.fileRepo.files[fileID] = &models.ChatFile{ID: fileID, ChannelID: channelID, StorageKey: "k", ThumbnailKey: nil}
		f.fileRepo.members[f.fileRepo.memberKey(channelID, userID)] = true

		_, err := f.srv.GetFileThumbnailURL(context.Background(), &chatv1.GetFileThumbnailURLRequest{
			FileId: fileID.String(), UserId: userID.String(),
		})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("success", func(t *testing.T) {
		f := newChatFilesSearchReactionsFixture()
		channelID, userID, fileID := uuid.New(), uuid.New(), uuid.New()
		thumbKey := "k.thumb.jpg"
		f.fileRepo.files[fileID] = &models.ChatFile{ID: fileID, ChannelID: channelID, StorageKey: "k", ThumbnailKey: &thumbKey}
		f.fileRepo.members[f.fileRepo.memberKey(channelID, userID)] = true

		resp, err := f.srv.GetFileThumbnailURL(context.Background(), &chatv1.GetFileThumbnailURLRequest{
			FileId: fileID.String(), UserId: userID.String(),
		})
		require.NoError(t, err)
		require.Equal(t, "https://example.com/presigned", resp.Url)
	})
}

func TestChatListChannelFiles(t *testing.T) {
	t.Run("invalid ids", func(t *testing.T) {
		f := newChatFilesSearchReactionsFixture()
		_, err := f.srv.ListChannelFiles(context.Background(), &chatv1.ListChannelFilesRequest{
			ChannelId: "bad", UserId: uuid.NewString(),
		})
		requireGRPCCode(t, err, codes.InvalidArgument)

		_, err = f.srv.ListChannelFiles(context.Background(), &chatv1.ListChannelFilesRequest{
			ChannelId: uuid.NewString(), UserId: "bad",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("not a channel member", func(t *testing.T) {
		f := newChatFilesSearchReactionsFixture()
		_, err := f.srv.ListChannelFiles(context.Background(), &chatv1.ListChannelFilesRequest{
			ChannelId: uuid.NewString(), UserId: uuid.NewString(),
		})
		requireGRPCCode(t, err, codes.PermissionDenied)
	})

	t.Run("empty list is not null", func(t *testing.T) {
		f := newChatFilesSearchReactionsFixture()
		channelID, userID := uuid.New(), uuid.New()
		f.fileRepo.members[f.fileRepo.memberKey(channelID, userID)] = true

		resp, err := f.srv.ListChannelFiles(context.Background(), &chatv1.ListChannelFilesRequest{
			ChannelId: channelID.String(), UserId: userID.String(), Page: 1, PageSize: 20,
		})
		require.NoError(t, err)
		require.Equal(t, int32(0), resp.Total)
		require.Empty(t, resp.Files)
	})

	t.Run("success returns uploader info", func(t *testing.T) {
		f := newChatFilesSearchReactionsFixture()
		channelID, userID, fileID := uuid.New(), uuid.New(), uuid.New()
		f.fileRepo.members[f.fileRepo.memberKey(channelID, userID)] = true
		f.fileRepo.files[fileID] = &models.ChatFile{
			ID: fileID, ChannelID: channelID, Filename: "report.pdf",
			MimeType: "application/pdf", UploadedBy: userID, CreatedAt: time.Now(),
		}
		f.fileRepo.userInfo[userID] = [2]string{"Anna", "Admin"}

		resp, err := f.srv.ListChannelFiles(context.Background(), &chatv1.ListChannelFilesRequest{
			ChannelId: channelID.String(), UserId: userID.String(), Page: 1, PageSize: 20,
		})
		require.NoError(t, err)
		require.Equal(t, int32(1), resp.Total)
		require.Len(t, resp.Files, 1)
		require.Equal(t, "report.pdf", resp.Files[0].Filename)
		require.Equal(t, "Anna", resp.Files[0].UploaderFirstName)
	})
}

func TestChatDeleteFile(t *testing.T) {
	t.Run("invalid ids", func(t *testing.T) {
		f := newChatFilesSearchReactionsFixture()
		_, err := f.srv.DeleteFile(context.Background(), &chatv1.DeleteFileRequest{
			FileId: "bad", UserId: uuid.NewString(),
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("uploader can delete own file", func(t *testing.T) {
		f := newChatFilesSearchReactionsFixture()
		channelID, userID, fileID := uuid.New(), uuid.New(), uuid.New()
		f.fileRepo.files[fileID] = &models.ChatFile{ID: fileID, ChannelID: channelID, UploadedBy: userID}

		_, err := f.srv.DeleteFile(context.Background(), &chatv1.DeleteFileRequest{
			FileId: fileID.String(), UserId: userID.String(),
		})
		require.NoError(t, err)
		require.True(t, f.fileRepo.files[fileID].IsDeleted)
	})

	t.Run("non-uploader without moderate role is denied", func(t *testing.T) {
		f := newChatFilesSearchReactionsFixture()
		channelID, uploaderID, otherID, fileID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
		f.fileRepo.files[fileID] = &models.ChatFile{ID: fileID, ChannelID: channelID, UploadedBy: uploaderID}
		f.fileRepo.roles[channelID.String()+":"+otherID.String()] = models.ChannelRoleMember

		_, err := f.srv.DeleteFile(context.Background(), &chatv1.DeleteFileRequest{
			FileId: fileID.String(), UserId: otherID.String(),
		})
		requireGRPCCode(t, err, codes.PermissionDenied)
		require.False(t, f.fileRepo.files[fileID].IsDeleted)
	})

	t.Run("channel admin can delete another member's file", func(t *testing.T) {
		f := newChatFilesSearchReactionsFixture()
		channelID, uploaderID, adminID, fileID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
		f.fileRepo.files[fileID] = &models.ChatFile{ID: fileID, ChannelID: channelID, UploadedBy: uploaderID}
		f.fileRepo.roles[channelID.String()+":"+adminID.String()] = models.ChannelRoleAdmin

		_, err := f.srv.DeleteFile(context.Background(), &chatv1.DeleteFileRequest{
			FileId: fileID.String(), UserId: adminID.String(),
		})
		require.NoError(t, err)
		require.True(t, f.fileRepo.files[fileID].IsDeleted)
	})
}

// ============================================================================
// Search
// ============================================================================

func TestChatSearchChat(t *testing.T) {
	t.Run("invalid user id", func(t *testing.T) {
		f := newChatFilesSearchReactionsFixture()
		_, err := f.srv.SearchChat(context.Background(), &chatv1.SearchChatRequest{
			UserId: "bad", Query: "invoice",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid channel id", func(t *testing.T) {
		f := newChatFilesSearchReactionsFixture()
		badChannel := "bad"
		_, err := f.srv.SearchChat(context.Background(), &chatv1.SearchChatRequest{
			UserId: uuid.NewString(), Query: "invoice", ChannelId: &badChannel,
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("query too short maps to InvalidArgument", func(t *testing.T) {
		f := newChatFilesSearchReactionsFixture()
		_, err := f.srv.SearchChat(context.Background(), &chatv1.SearchChatRequest{
			UserId: uuid.NewString(), Query: "a",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("success", func(t *testing.T) {
		f := newChatFilesSearchReactionsFixture()
		userID, channelID, msgID := uuid.New(), uuid.New(), uuid.New()
		f.searchRepo.userChannels[userID] = []uuid.UUID{channelID}
		f.searchRepo.messageResults = []models.ChatSearchResult{
			{Type: models.ChatSearchResultMessage, ID: msgID, ChannelID: channelID, ChannelName: "general", Score: 1, MessageID: &msgID},
		}

		resp, err := f.srv.SearchChat(context.Background(), &chatv1.SearchChatRequest{
			UserId: userID.String(), Query: "invoice", Page: 1, PageSize: 20,
		})
		require.NoError(t, err)
		require.Equal(t, int32(1), resp.Total)
		require.Len(t, resp.Results, 1)
		require.Equal(t, msgID.String(), *resp.Results[0].MessageId)
	})
}

func TestToChatSearchResultProto(t *testing.T) {
	t.Run("fully populated", func(t *testing.T) {
		now := time.Now()
		msgID, channelID, parentID, fileID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
		filename, mimeType := "invoice.pdf", "application/pdf"
		var fileSize int64 = 1024

		proto := toChatSearchResultProto(&models.ChatSearchResult{
			Type: models.ChatSearchResultFile, ID: fileID, ChannelID: channelID, ChannelName: "general",
			Score: 0.9, Snippet: "...invoice...", CreatedAt: now, FirstName: "Anna", LastName: "Admin",
			MessageID: &msgID, ParentMessageID: &parentID, FileID: &fileID,
			Filename: &filename, MimeType: &mimeType, FileSize: &fileSize,
		})

		require.Equal(t, "file", proto.Type)
		require.Equal(t, fileID.String(), proto.Id)
		require.Equal(t, msgID.String(), *proto.MessageId)
		require.Equal(t, parentID.String(), *proto.ParentMessageId)
		require.Equal(t, fileID.String(), *proto.FileId)
		require.Equal(t, filename, *proto.Filename)
		require.Equal(t, mimeType, *proto.MimeType)
		require.Equal(t, fileSize, *proto.FileSize)
	})

	t.Run("message result leaves file-specific fields unset", func(t *testing.T) {
		channelID, resultID := uuid.New(), uuid.New()
		proto := toChatSearchResultProto(&models.ChatSearchResult{
			Type: models.ChatSearchResultMessage, ID: resultID, ChannelID: channelID, ChannelName: "general",
			Score: 0.5, Snippet: "hello",
		})

		require.Equal(t, "message", proto.Type)
		require.Nil(t, proto.MessageId)
		require.Nil(t, proto.ParentMessageId)
		require.Nil(t, proto.FileId)
		require.Nil(t, proto.Filename)
		require.Nil(t, proto.MimeType)
		require.Nil(t, proto.FileSize)
	})
}

// ============================================================================
// Reactions
// ============================================================================

func TestChatToggleReaction(t *testing.T) {
	t.Run("invalid ids", func(t *testing.T) {
		f := newChatFilesSearchReactionsFixture()
		_, err := f.srv.ToggleReaction(context.Background(), &chatv1.ToggleReactionRequest{
			MessageId: "bad", UserId: uuid.NewString(), Emoji: "👍",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("empty emoji maps to InvalidArgument", func(t *testing.T) {
		f := newChatFilesSearchReactionsFixture()
		_, err := f.srv.ToggleReaction(context.Background(), &chatv1.ToggleReactionRequest{
			MessageId: uuid.NewString(), UserId: uuid.NewString(), Emoji: "  ",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("adds then removes on repeat toggle", func(t *testing.T) {
		f := newChatFilesSearchReactionsFixture()
		msgID, userID := uuid.New(), uuid.New()

		resp, err := f.srv.ToggleReaction(context.Background(), &chatv1.ToggleReactionRequest{
			MessageId: msgID.String(), UserId: userID.String(), Emoji: "👍",
		})
		require.NoError(t, err)
		require.True(t, resp.Added)
		require.Len(t, resp.Reactions, 1)
		require.Equal(t, userID.String(), resp.Reactions[0].UserId)

		resp, err = f.srv.ToggleReaction(context.Background(), &chatv1.ToggleReactionRequest{
			MessageId: msgID.String(), UserId: userID.String(), Emoji: "👍",
		})
		require.NoError(t, err)
		require.False(t, resp.Added)
		require.Empty(t, resp.Reactions)
	})
}

func TestChatListReactions(t *testing.T) {
	t.Run("invalid message id", func(t *testing.T) {
		f := newChatFilesSearchReactionsFixture()
		_, err := f.srv.ListReactions(context.Background(), &chatv1.ListReactionsRequest{MessageId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("success", func(t *testing.T) {
		f := newChatFilesSearchReactionsFixture()
		msgID, userA, userB := uuid.New(), uuid.New(), uuid.New()
		require.NoError(t, f.reactRepo.AddReaction(context.Background(), msgID, userA, "👍"))
		require.NoError(t, f.reactRepo.AddReaction(context.Background(), msgID, userB, "🎉"))

		resp, err := f.srv.ListReactions(context.Background(), &chatv1.ListReactionsRequest{MessageId: msgID.String()})
		require.NoError(t, err)
		require.Len(t, resp.Reactions, 2)
	})
}

func TestChatGetReactionSummary(t *testing.T) {
	t.Run("invalid user id", func(t *testing.T) {
		f := newChatFilesSearchReactionsFixture()
		_, err := f.srv.GetReactionSummary(context.Background(), &chatv1.GetReactionSummaryRequest{
			UserId: "bad", MessageIds: []string{uuid.NewString()},
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid message id in list", func(t *testing.T) {
		f := newChatFilesSearchReactionsFixture()
		_, err := f.srv.GetReactionSummary(context.Background(), &chatv1.GetReactionSummaryRequest{
			UserId: uuid.NewString(), MessageIds: []string{"bad"},
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("success reflects current user's reaction", func(t *testing.T) {
		f := newChatFilesSearchReactionsFixture()
		msgID, userID := uuid.New(), uuid.New()
		require.NoError(t, f.reactRepo.AddReaction(context.Background(), msgID, userID, "👍"))

		resp, err := f.srv.GetReactionSummary(context.Background(), &chatv1.GetReactionSummaryRequest{
			UserId: userID.String(), MessageIds: []string{msgID.String()},
		})
		require.NoError(t, err)
		require.Len(t, resp.Summaries, 1)
		require.Equal(t, "👍", resp.Summaries[0].Emoji)
		require.Equal(t, int32(1), resp.Summaries[0].Count)
		require.True(t, resp.Summaries[0].CurrentUserReacted)
	})
}

// ============================================================================
// Bookmarks
// ============================================================================

func TestChatToggleBookmark(t *testing.T) {
	t.Run("missing tenant", func(t *testing.T) {
		f := newChatFilesSearchReactionsFixture()
		_, err := f.srv.ToggleBookmark(context.Background(), &chatv1.ToggleBookmarkRequest{
			MessageId: uuid.NewString(), UserId: uuid.NewString(),
		})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("invalid ids", func(t *testing.T) {
		f := newChatFilesSearchReactionsFixture()
		_, err := f.srv.ToggleBookmark(fsrCtx(uuid.New()), &chatv1.ToggleBookmarkRequest{
			MessageId: "bad", UserId: uuid.NewString(),
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("message not visible to caller maps to NotFound", func(t *testing.T) {
		f := newChatFilesSearchReactionsFixture()
		msgID := uuid.New()
		f.msgReader.notFound[msgID] = true

		_, err := f.srv.ToggleBookmark(fsrCtx(uuid.New()), &chatv1.ToggleBookmarkRequest{
			MessageId: msgID.String(), UserId: uuid.NewString(),
		})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("adds then removes on repeat toggle", func(t *testing.T) {
		f := newChatFilesSearchReactionsFixture()
		tenantID, userID, msgID := uuid.New(), uuid.New(), uuid.New()
		f.msgReader.messages[msgID] = &models.MessageWithSender{Message: models.Message{ID: msgID, ChannelID: uuid.New(), CreatedAt: time.Now()}}

		resp, err := f.srv.ToggleBookmark(fsrCtx(tenantID), &chatv1.ToggleBookmarkRequest{
			MessageId: msgID.String(), UserId: userID.String(),
		})
		require.NoError(t, err)
		require.True(t, resp.Bookmarked)

		resp, err = f.srv.ToggleBookmark(fsrCtx(tenantID), &chatv1.ToggleBookmarkRequest{
			MessageId: msgID.String(), UserId: userID.String(),
		})
		require.NoError(t, err)
		require.False(t, resp.Bookmarked)
	})
}

func TestChatListBookmarks(t *testing.T) {
	t.Run("missing tenant", func(t *testing.T) {
		f := newChatFilesSearchReactionsFixture()
		_, err := f.srv.ListBookmarks(context.Background(), &chatv1.ListBookmarksRequest{UserId: uuid.NewString()})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("invalid user id", func(t *testing.T) {
		f := newChatFilesSearchReactionsFixture()
		_, err := f.srv.ListBookmarks(fsrCtx(uuid.New()), &chatv1.ListBookmarksRequest{UserId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("skips bookmarks whose message is no longer visible", func(t *testing.T) {
		f := newChatFilesSearchReactionsFixture()
		tenantID, userID := uuid.New(), uuid.New()
		visibleID, revokedID := uuid.New(), uuid.New()
		f.msgReader.messages[visibleID] = &models.MessageWithSender{Message: models.Message{ID: visibleID, ChannelID: uuid.New(), CreatedAt: time.Now()}}
		f.msgReader.notMember[revokedID] = true
		f.bmRepo.order[bookmarkKey(tenantID, userID)] = []uuid.UUID{revokedID, visibleID}

		resp, err := f.srv.ListBookmarks(fsrCtx(tenantID), &chatv1.ListBookmarksRequest{UserId: userID.String()})
		require.NoError(t, err)
		require.Len(t, resp.Messages, 1)
		require.Equal(t, visibleID.String(), resp.Messages[0].Id)
	})
}

// ============================================================================
// Wire-shape helpers
// ============================================================================

func TestToProtoMentionType(t *testing.T) {
	tests := []struct {
		name string
		in   models.MentionType
		want chatv1.MentionType
	}{
		{"user", models.MentionTypeUser, chatv1.MentionType_MENTION_TYPE_USER},
		{"channel", models.MentionTypeChannel, chatv1.MentionType_MENTION_TYPE_CHANNEL},
		{"everyone", models.MentionTypeEveryone, chatv1.MentionType_MENTION_TYPE_EVERYONE},
		{"unknown falls back to user", models.MentionType("bogus"), chatv1.MentionType_MENTION_TYPE_USER},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, toProtoMentionType(tt.in))
		})
	}
}
