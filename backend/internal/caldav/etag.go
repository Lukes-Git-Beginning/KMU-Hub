package caldav

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// GenerateETag produces a deterministic ETag from a UUID and updated_at timestamp.
// The ETag is the first 16 bytes of the SHA-256 hash of the UUID bytes concatenated
// with the updatedAt unix nanoseconds, hex-encoded and wrapped in quotes.
func GenerateETag(id uuid.UUID, updatedAt time.Time) string {
	h := sha256.New()
	h.Write(id[:])
	ts := make([]byte, 8)
	binary.BigEndian.PutUint64(ts, uint64(updatedAt.UnixNano()))
	h.Write(ts)
	sum := h.Sum(nil)
	return fmt.Sprintf(`"%s"`, hex.EncodeToString(sum[:16]))
}

// GenerateCTag produces a CTag string from a sync version number.
// Format: "ctag-{syncVersion}" for calendar/address book collection versioning.
func GenerateCTag(syncVersion int64) string {
	return fmt.Sprintf("ctag-%d", syncVersion)
}
