package account

import (
	"context"
	"errors"
	"sort"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// --- Mock VaultEncryptor ---

type mockEncryptor struct {
	encryptFn func(ctx context.Context, plaintext []byte) (string, error)
	decryptFn func(ctx context.Context, encrypted string) ([]byte, error)
}

func (m *mockEncryptor) Encrypt(ctx context.Context, plaintext []byte) (string, error) {
	if m.encryptFn != nil {
		return m.encryptFn(ctx, plaintext)
	}
	return "encrypted:" + string(plaintext), nil
}

func (m *mockEncryptor) Decrypt(ctx context.Context, encrypted string) ([]byte, error) {
	if m.decryptFn != nil {
		return m.decryptFn(ctx, encrypted)
	}
	// Strip "encrypted:" prefix for default mock
	if len(encrypted) > 10 && encrypted[:10] == "encrypted:" {
		return []byte(encrypted[10:]), nil
	}
	return []byte(encrypted), nil
}

// --- Mock Repository ---

type mockRepo struct {
	accounts map[uuid.UUID]*models.EmailAccount
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		accounts: make(map[uuid.UUID]*models.EmailAccount),
	}
}

func (r *mockRepo) Create(ctx context.Context, account *models.EmailAccount) error {
	// Store a copy to avoid the service modifying the stored value
	copy := *account
	r.accounts[account.ID] = &copy
	return nil
}

func (r *mockRepo) GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.EmailAccount, error) {
	a, ok := r.accounts[id]
	if !ok {
		return nil, ErrAccountNotFound
	}
	if a.TenantID != tenantID {
		return nil, ErrAccountNotFound
	}
	// Return a copy to avoid mutation
	copy := *a
	return &copy, nil
}

func (r *mockRepo) GetByUserIDAndTenant(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) (*models.EmailAccount, error) {
	for _, a := range r.accounts {
		if a.UserID == userID && a.TenantID == tenantID && a.IsDefault {
			copy := *a
			return &copy, nil
		}
	}
	return nil, ErrAccountNotFound
}

func (r *mockRepo) ListByUserAndTenant(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) ([]*models.EmailAccount, error) {
	var result []*models.EmailAccount
	for _, a := range r.accounts {
		if a.UserID == userID && a.TenantID == tenantID {
			copy := *a
			result = append(result, &copy)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.Before(result[j].CreatedAt) })
	return result, nil
}

func (r *mockRepo) Update(ctx context.Context, account *models.EmailAccount) error {
	if _, ok := r.accounts[account.ID]; !ok {
		return ErrAccountNotFound
	}
	copy := *account
	r.accounts[account.ID] = &copy
	return nil
}

func (r *mockRepo) Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	a, ok := r.accounts[id]
	if !ok {
		return ErrAccountNotFound
	}
	if a.TenantID != tenantID {
		return ErrAccountNotFound
	}
	delete(r.accounts, id)
	return nil
}

// SetDefault mirrors the atomic single-statement semantics of the real
// repository: id becomes the sole default for userID within tenantID.
func (r *mockRepo) SetDefault(ctx context.Context, id uuid.UUID, userID uuid.UUID, tenantID uuid.UUID) error {
	for _, a := range r.accounts {
		if a.UserID == userID && a.TenantID == tenantID {
			a.IsDefault = a.ID == id
		}
	}
	return nil
}

func (r *mockRepo) ListActive(ctx context.Context, tenantID uuid.UUID) ([]*models.EmailAccount, error) {
	var result []*models.EmailAccount
	for _, a := range r.accounts {
		if a.SyncEnabled && a.TenantID == tenantID {
			result = append(result, a)
		}
	}
	return result, nil
}

func (r *mockRepo) ListAllActive(ctx context.Context) ([]*models.EmailAccount, error) {
	var result []*models.EmailAccount
	for _, a := range r.accounts {
		if a.SyncEnabled {
			result = append(result, a)
		}
	}
	return result, nil
}

// --- Tests ---

func validInput() CreateInput {
	return CreateInput{
		UserID:       uuid.New(),
		EmailAddress: "test@example.com",
		DisplayName:  "Test User",
		IMAPHost:     "imap.example.com",
		IMAPPort:     993,
		SMTPHost:     "smtp.example.com",
		SMTPPort:     587,
		Username:     "test@example.com",
		Password:     "secret123",
		UseSSL:       true,
	}
}

// testTenantID is the canonical tenant used across test cases.
var testTenantID = uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")

func TestService_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := newMockRepo()
		enc := &mockEncryptor{}
		svc := NewService(repo, enc)

		input := validInput()
		account, err := svc.Create(context.Background(), testTenantID, input)

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, account.ID)
		assert.Equal(t, testTenantID, account.TenantID)
		assert.Equal(t, input.UserID, account.UserID)
		assert.Equal(t, input.EmailAddress, account.EmailAddress)
		assert.Equal(t, input.DisplayName, account.DisplayName)
		assert.Equal(t, input.IMAPHost, account.IMAPHost)
		assert.Equal(t, 993, account.IMAPPort)
		assert.True(t, account.SyncEnabled)
		// Password must not be returned
		assert.Empty(t, account.PasswordEncrypted)
	})

	t.Run("password encrypted on create", func(t *testing.T) {
		repo := newMockRepo()
		var encryptedValue string
		enc := &mockEncryptor{
			encryptFn: func(ctx context.Context, plaintext []byte) (string, error) {
				encryptedValue = "vault:" + string(plaintext)
				return encryptedValue, nil
			},
		}
		svc := NewService(repo, enc)

		input := validInput()
		_, err := svc.Create(context.Background(), testTenantID, input)

		require.NoError(t, err)
		// Verify the stored value in repo is encrypted
		stored := repo.accounts[getFirstKey(repo.accounts)]
		assert.Equal(t, encryptedValue, stored.PasswordEncrypted)
	})

	t.Run("first account becomes default, second does not", func(t *testing.T) {
		repo := newMockRepo()
		enc := &mockEncryptor{}
		svc := NewService(repo, enc)

		userID := uuid.New()
		input1 := validInput()
		input1.UserID = userID
		account1, err := svc.Create(context.Background(), testTenantID, input1)
		require.NoError(t, err)
		assert.True(t, account1.IsDefault)

		input2 := validInput()
		input2.UserID = userID
		input2.EmailAddress = "second@example.com"
		account2, err := svc.Create(context.Background(), testTenantID, input2)
		require.NoError(t, err)
		assert.False(t, account2.IsDefault)

		accounts, err := svc.ListByUserAndTenant(context.Background(), userID, testTenantID)
		require.NoError(t, err)
		assert.Len(t, accounts, 2)
	})

	t.Run("invalid email rejected", func(t *testing.T) {
		repo := newMockRepo()
		enc := &mockEncryptor{}
		svc := NewService(repo, enc)

		input := validInput()
		input.EmailAddress = "not-an-email"
		_, err := svc.Create(context.Background(), testTenantID, input)
		assert.Error(t, err)
	})

	t.Run("empty password rejected", func(t *testing.T) {
		repo := newMockRepo()
		enc := &mockEncryptor{}
		svc := NewService(repo, enc)

		input := validInput()
		input.Password = ""
		_, err := svc.Create(context.Background(), testTenantID, input)
		assert.Error(t, err)
	})

	t.Run("encryption failure propagated", func(t *testing.T) {
		repo := newMockRepo()
		enc := &mockEncryptor{
			encryptFn: func(ctx context.Context, plaintext []byte) (string, error) {
				return "", errors.New("vault unavailable")
			},
		}
		svc := NewService(repo, enc)

		input := validInput()
		_, err := svc.Create(context.Background(), testTenantID, input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "encrypt")
	})

	t.Run("default ports when zero", func(t *testing.T) {
		repo := newMockRepo()
		enc := &mockEncryptor{}
		svc := NewService(repo, enc)

		input := validInput()
		input.IMAPPort = 0
		input.SMTPPort = 0
		account, err := svc.Create(context.Background(), testTenantID, input)

		require.NoError(t, err)
		stored := repo.accounts[account.ID]
		assert.Equal(t, 993, stored.IMAPPort)
		assert.Equal(t, 587, stored.SMTPPort)
	})
}

func TestService_TenantIsolation(t *testing.T) {
	t.Run("tenant A account not visible to tenant B", func(t *testing.T) {
		repo := newMockRepo()
		enc := &mockEncryptor{}
		svc := NewService(repo, enc)

		tenantA := uuid.MustParse("11111111-1111-1111-1111-111111111111")
		tenantB := uuid.MustParse("22222222-2222-2222-2222-222222222222")

		input := validInput()
		accountA, err := svc.Create(context.Background(), tenantA, input)
		require.NoError(t, err)

		// Tenant B tries to fetch tenant A's account by ID
		_, err = svc.GetByID(context.Background(), accountA.ID, tenantB)
		assert.ErrorIs(t, err, ErrAccountNotFound)

		// Tenant B tries to fetch tenant A's account by user ID
		_, err = svc.GetByUserIDAndTenant(context.Background(), input.UserID, tenantB)
		assert.ErrorIs(t, err, ErrAccountNotFound)
	})
}

func TestService_GetDecryptedCredentials(t *testing.T) {
	t.Run("success round-trip", func(t *testing.T) {
		repo := newMockRepo()
		enc := &mockEncryptor{}
		svc := NewService(repo, enc)

		input := validInput()
		account, err := svc.Create(context.Background(), testTenantID, input)
		require.NoError(t, err)

		creds, err := svc.GetDecryptedCredentials(context.Background(), account.ID, testTenantID)
		require.NoError(t, err)

		assert.Equal(t, input.IMAPHost, creds.IMAPHost)
		assert.Equal(t, input.IMAPPort, creds.IMAPPort)
		assert.Equal(t, input.SMTPHost, creds.SMTPHost)
		assert.Equal(t, input.SMTPPort, creds.SMTPPort)
		assert.Equal(t, input.Username, creds.Username)
		assert.Equal(t, input.Password, creds.Password)
		assert.Equal(t, input.UseSSL, creds.UseSSL)
	})

	t.Run("not found", func(t *testing.T) {
		repo := newMockRepo()
		enc := &mockEncryptor{}
		svc := NewService(repo, enc)

		_, err := svc.GetDecryptedCredentials(context.Background(), uuid.New(), testTenantID)
		assert.ErrorIs(t, err, ErrAccountNotFound)
	})

	t.Run("decryption failure", func(t *testing.T) {
		repo := newMockRepo()
		enc := &mockEncryptor{
			decryptFn: func(ctx context.Context, encrypted string) ([]byte, error) {
				return nil, errors.New("decryption failed")
			},
		}
		svc := NewService(repo, enc)

		input := validInput()
		account, err := svc.Create(context.Background(), testTenantID, input)
		require.NoError(t, err)

		_, err = svc.GetDecryptedCredentials(context.Background(), account.ID, testTenantID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "decrypt")
	})
}

func TestService_Update(t *testing.T) {
	t.Run("update email address", func(t *testing.T) {
		repo := newMockRepo()
		enc := &mockEncryptor{}
		svc := NewService(repo, enc)

		input := validInput()
		account, err := svc.Create(context.Background(), testTenantID, input)
		require.NoError(t, err)

		newEmail := "new@example.com"
		updated, err := svc.Update(context.Background(), account.ID, testTenantID, UpdateInput{
			EmailAddress: &newEmail,
		})

		require.NoError(t, err)
		assert.Equal(t, "new@example.com", updated.EmailAddress)
		assert.Empty(t, updated.PasswordEncrypted)
	})

	t.Run("update password re-encrypts", func(t *testing.T) {
		repo := newMockRepo()
		encryptCalls := 0
		enc := &mockEncryptor{
			encryptFn: func(ctx context.Context, plaintext []byte) (string, error) {
				encryptCalls++
				return "enc:" + string(plaintext), nil
			},
		}
		svc := NewService(repo, enc)

		input := validInput()
		account, err := svc.Create(context.Background(), testTenantID, input)
		require.NoError(t, err)
		assert.Equal(t, 1, encryptCalls)

		newPass := "newpassword"
		_, err = svc.Update(context.Background(), account.ID, testTenantID, UpdateInput{
			Password: &newPass,
		})

		require.NoError(t, err)
		assert.Equal(t, 2, encryptCalls)
		stored := repo.accounts[account.ID]
		assert.Equal(t, "enc:newpassword", stored.PasswordEncrypted)
	})

	t.Run("not found", func(t *testing.T) {
		repo := newMockRepo()
		enc := &mockEncryptor{}
		svc := NewService(repo, enc)

		newEmail := "test@example.com"
		_, err := svc.Update(context.Background(), uuid.New(), testTenantID, UpdateInput{
			EmailAddress: &newEmail,
		})
		assert.ErrorIs(t, err, ErrAccountNotFound)
	})
}

func TestService_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := newMockRepo()
		enc := &mockEncryptor{}
		svc := NewService(repo, enc)

		input := validInput()
		account, err := svc.Create(context.Background(), testTenantID, input)
		require.NoError(t, err)

		err = svc.Delete(context.Background(), account.ID, testTenantID)
		require.NoError(t, err)

		_, err = svc.GetByID(context.Background(), account.ID, testTenantID)
		assert.ErrorIs(t, err, ErrAccountNotFound)
	})

	t.Run("not found", func(t *testing.T) {
		repo := newMockRepo()
		enc := &mockEncryptor{}
		svc := NewService(repo, enc)

		err := svc.Delete(context.Background(), uuid.New(), testTenantID)
		assert.ErrorIs(t, err, ErrAccountNotFound)
	})

	t.Run("deleting the default promotes the oldest remaining account", func(t *testing.T) {
		repo := newMockRepo()
		enc := &mockEncryptor{}
		svc := NewService(repo, enc)

		userID := uuid.New()
		input1 := validInput()
		input1.UserID = userID
		accountA, err := svc.Create(context.Background(), testTenantID, input1)
		require.NoError(t, err)
		assert.True(t, accountA.IsDefault)

		input2 := validInput()
		input2.UserID = userID
		input2.EmailAddress = "second@example.com"
		accountB, err := svc.Create(context.Background(), testTenantID, input2)
		require.NoError(t, err)
		assert.False(t, accountB.IsDefault)

		require.NoError(t, svc.Delete(context.Background(), accountA.ID, testTenantID))

		promoted, err := svc.GetByID(context.Background(), accountB.ID, testTenantID)
		require.NoError(t, err)
		assert.True(t, promoted.IsDefault)

		got, err := svc.GetByUserIDAndTenant(context.Background(), userID, testTenantID)
		require.NoError(t, err)
		assert.Equal(t, accountB.ID, got.ID)
	})

	t.Run("deleting the last account leaves no default", func(t *testing.T) {
		repo := newMockRepo()
		enc := &mockEncryptor{}
		svc := NewService(repo, enc)

		input := validInput()
		account, err := svc.Create(context.Background(), testTenantID, input)
		require.NoError(t, err)

		require.NoError(t, svc.Delete(context.Background(), account.ID, testTenantID))

		accounts, err := svc.ListByUserAndTenant(context.Background(), input.UserID, testTenantID)
		require.NoError(t, err)
		assert.Empty(t, accounts)
	})

	t.Run("deleting a non-default account does not touch the default", func(t *testing.T) {
		repo := newMockRepo()
		enc := &mockEncryptor{}
		svc := NewService(repo, enc)

		userID := uuid.New()
		input1 := validInput()
		input1.UserID = userID
		accountA, err := svc.Create(context.Background(), testTenantID, input1)
		require.NoError(t, err)

		input2 := validInput()
		input2.UserID = userID
		input2.EmailAddress = "second@example.com"
		accountB, err := svc.Create(context.Background(), testTenantID, input2)
		require.NoError(t, err)

		require.NoError(t, svc.Delete(context.Background(), accountB.ID, testTenantID))

		got, err := svc.GetByID(context.Background(), accountA.ID, testTenantID)
		require.NoError(t, err)
		assert.True(t, got.IsDefault)
	})
}

func TestService_SetDefault(t *testing.T) {
	t.Run("switches default and clears the prior one", func(t *testing.T) {
		repo := newMockRepo()
		enc := &mockEncryptor{}
		svc := NewService(repo, enc)

		userID := uuid.New()
		input1 := validInput()
		input1.UserID = userID
		accountA, err := svc.Create(context.Background(), testTenantID, input1)
		require.NoError(t, err)

		input2 := validInput()
		input2.UserID = userID
		input2.EmailAddress = "second@example.com"
		accountB, err := svc.Create(context.Background(), testTenantID, input2)
		require.NoError(t, err)

		require.NoError(t, svc.SetDefault(context.Background(), accountB.ID, testTenantID))

		gotB, err := svc.GetByID(context.Background(), accountB.ID, testTenantID)
		require.NoError(t, err)
		assert.True(t, gotB.IsDefault)

		gotA, err := svc.GetByID(context.Background(), accountA.ID, testTenantID)
		require.NoError(t, err)
		assert.False(t, gotA.IsDefault)

		got, err := svc.GetByUserIDAndTenant(context.Background(), userID, testTenantID)
		require.NoError(t, err)
		assert.Equal(t, accountB.ID, got.ID)
	})

	t.Run("not found", func(t *testing.T) {
		repo := newMockRepo()
		enc := &mockEncryptor{}
		svc := NewService(repo, enc)

		err := svc.SetDefault(context.Background(), uuid.New(), testTenantID)
		assert.ErrorIs(t, err, ErrAccountNotFound)
	})

	t.Run("wrong tenant rejected", func(t *testing.T) {
		repo := newMockRepo()
		enc := &mockEncryptor{}
		svc := NewService(repo, enc)

		input := validInput()
		account, err := svc.Create(context.Background(), testTenantID, input)
		require.NoError(t, err)

		otherTenant := uuid.New()
		err = svc.SetDefault(context.Background(), account.ID, otherTenant)
		assert.ErrorIs(t, err, ErrAccountNotFound)
	})
}

func TestService_ListActive(t *testing.T) {
	repo := newMockRepo()
	enc := &mockEncryptor{}
	svc := NewService(repo, enc)

	// Create active account for testTenantID
	input1 := validInput()
	_, err := svc.Create(context.Background(), testTenantID, input1)
	require.NoError(t, err)

	// Create another user for testTenantID, disable sync
	input2 := validInput()
	input2.UserID = uuid.New()
	input2.EmailAddress = "other@example.com"
	account2, err := svc.Create(context.Background(), testTenantID, input2)
	require.NoError(t, err)

	disabled := false
	_, err = svc.Update(context.Background(), account2.ID, testTenantID, UpdateInput{SyncEnabled: &disabled})
	require.NoError(t, err)

	active, err := svc.ListActive(context.Background(), testTenantID)
	require.NoError(t, err)
	assert.Len(t, active, 1)
}

func TestService_GetByID(t *testing.T) {
	repo := newMockRepo()
	enc := &mockEncryptor{}
	svc := NewService(repo, enc)

	input := validInput()
	created, err := svc.Create(context.Background(), testTenantID, input)
	require.NoError(t, err)

	got, err := svc.GetByID(context.Background(), created.ID, testTenantID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
	assert.Empty(t, got.PasswordEncrypted)
}

func TestService_GetByUserIDAndTenant(t *testing.T) {
	repo := newMockRepo()
	enc := &mockEncryptor{}
	svc := NewService(repo, enc)

	input := validInput()
	_, err := svc.Create(context.Background(), testTenantID, input)
	require.NoError(t, err)

	got, err := svc.GetByUserIDAndTenant(context.Background(), input.UserID, testTenantID)
	require.NoError(t, err)
	assert.Equal(t, input.EmailAddress, got.EmailAddress)
	assert.Empty(t, got.PasswordEncrypted)
}

// getFirstKey returns the first key from a map (helper for tests).
func getFirstKey(m map[uuid.UUID]*models.EmailAccount) uuid.UUID {
	for k := range m {
		return k
	}
	return uuid.Nil
}
