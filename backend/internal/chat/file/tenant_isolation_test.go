package file

// Cross-tenant isolation tests for chat_files (Migration 000115).
//
// chat_files.tenant_id is backfilled from channels.tenant_id via channel_id.
// The mock repository filters by channel_id; since each channel belongs to
// exactly one tenant, listing files per channel is already tenant-scoped.
// These tests verify that ListChannelFiles for channel A never returns files
// that belong to channel B (i.e., a different tenant's channel).

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// TestChatFiles_ChannelIsolation verifies that listing files for channel A does
// not return files uploaded to channel B (which may belong to a different tenant).
func TestChatFiles_ChannelIsolation(t *testing.T) {
	repo := NewMockRepository()

	channelA := uuid.New() // tenant A's channel
	channelB := uuid.New() // tenant B's channel
	uploader := uuid.New()

	// Register the uploader in the mock.
	repo.users[uploader] = struct{ firstName, lastName string }{"Alice", "Tenant"}

	now := time.Now().UTC()

	// Seed a file in channel A.
	fileA := &models.ChatFile{
		ID:         uuid.New(),
		ChannelID:  channelA,
		Filename:   "report-tenant-a.pdf",
		MimeType:   "application/pdf",
		FileSize:   1024,
		StorageKey: "a/report.pdf",
		UploadedBy: uploader,
		IsDeleted:  false,
		CreatedAt:  now,
	}

	// Seed a file in channel B.
	fileB := &models.ChatFile{
		ID:         uuid.New(),
		ChannelID:  channelB,
		Filename:   "secret-tenant-b.pdf",
		MimeType:   "application/pdf",
		FileSize:   2048,
		StorageKey: "b/secret.pdf",
		UploadedBy: uploader,
		IsDeleted:  false,
		CreatedAt:  now,
	}

	ctx := context.Background()
	require.NoError(t, repo.CreateFile(ctx, fileA))
	require.NoError(t, repo.CreateFile(ctx, fileB))

	// List files for channel A — must only return fileA.
	filesA, total, err := repo.ListChannelFiles(ctx, channelA, 50, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, filesA, 1)
	assert.Equal(t, channelA, filesA[0].ChannelID)
	assert.Equal(t, "report-tenant-a.pdf", filesA[0].Filename)

	// List files for channel B — must only return fileB.
	filesB, totalB, err := repo.ListChannelFiles(ctx, channelB, 50, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, totalB)
	require.Len(t, filesB, 1)
	assert.Equal(t, channelB, filesB[0].ChannelID)
	assert.Equal(t, "secret-tenant-b.pdf", filesB[0].Filename)

	// Verify no cross-contamination.
	for _, f := range filesB {
		assert.NotEqual(t, channelA, f.ChannelID, "channel A file leaked into channel B listing")
		assert.NotEqual(t, "report-tenant-a.pdf", f.Filename, "tenant A filename leaked into tenant B listing")
	}
}

// TestChatFiles_GetFileByID_ReturnsCorrectOwner verifies that GetFileByID
// returns the file with the correct channel assignment.
func TestChatFiles_GetFileByID_ReturnsCorrectOwner(t *testing.T) {
	repo := NewMockRepository()

	channelA := uuid.New()
	channelB := uuid.New()
	uploader := uuid.New()
	repo.users[uploader] = struct{ firstName, lastName string }{"Bob", "Tester"}

	now := time.Now().UTC()
	fileA := &models.ChatFile{
		ID:         uuid.New(),
		ChannelID:  channelA,
		Filename:   "tenantA.txt",
		MimeType:   "text/plain",
		FileSize:   100,
		StorageKey: "a/file.txt",
		UploadedBy: uploader,
		IsDeleted:  false,
		CreatedAt:  now,
	}
	fileB := &models.ChatFile{
		ID:         uuid.New(),
		ChannelID:  channelB,
		Filename:   "tenantB.txt",
		MimeType:   "text/plain",
		FileSize:   200,
		StorageKey: "b/file.txt",
		UploadedBy: uploader,
		IsDeleted:  false,
		CreatedAt:  now,
	}

	ctx := context.Background()
	require.NoError(t, repo.CreateFile(ctx, fileA))
	require.NoError(t, repo.CreateFile(ctx, fileB))

	// Fetching fileA by ID returns fileA's channel.
	gotA, err := repo.GetFileByID(ctx, fileA.ID)
	require.NoError(t, err)
	assert.Equal(t, channelA, gotA.ChannelID)

	// Fetching fileB by ID returns fileB's channel — not fileA's.
	gotB, err := repo.GetFileByID(ctx, fileB.ID)
	require.NoError(t, err)
	assert.Equal(t, channelB, gotB.ChannelID)
	assert.NotEqual(t, channelA, gotB.ChannelID, "channel A identity leaked into file B")
}
