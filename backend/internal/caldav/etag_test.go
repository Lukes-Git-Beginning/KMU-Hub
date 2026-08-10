package caldav

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestGenerateETag_Deterministic(t *testing.T) {
	id := uuid.New()
	updatedAt := time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC)

	first := GenerateETag(id, updatedAt)
	second := GenerateETag(id, updatedAt)

	assert.Equal(t, first, second, "same id and timestamp must produce the same ETag")
	assert.True(t, len(first) > 2 && first[0] == '"' && first[len(first)-1] == '"', "ETag must be quoted: %q", first)
}

func TestGenerateETag_DiffersOnUpdatedAt(t *testing.T) {
	id := uuid.New()
	t1 := time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC)
	t2 := t1.Add(time.Nanosecond)

	assert.NotEqual(t, GenerateETag(id, t1), GenerateETag(id, t2), "changing updatedAt by even 1ns must change the ETag")
}

func TestGenerateETag_DiffersOnID(t *testing.T) {
	updatedAt := time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC)

	assert.NotEqual(t, GenerateETag(uuid.New(), updatedAt), GenerateETag(uuid.New(), updatedAt))
}

func TestGenerateCTag_Format(t *testing.T) {
	assert.Equal(t, "ctag-42", GenerateCTag(42))
	assert.Equal(t, "ctag-0", GenerateCTag(0))
	assert.Equal(t, "ctag--1", GenerateCTag(-1))
}
