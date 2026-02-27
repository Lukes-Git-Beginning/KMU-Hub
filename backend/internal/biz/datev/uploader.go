package datev

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type Uploader struct {
	oauthManager *OAuthManager
	httpClient   *http.Client
	baseURL      string
}

func NewUploader(oauthManager *OAuthManager, baseURL string) *Uploader {
	return &Uploader{
		oauthManager: oauthManager,
		httpClient:   &http.Client{Timeout: 60 * time.Second},
		baseURL:      baseURL,
	}
}

// UploadBuchungsstapel uploads a DATEV CSV Buchungsstapel to the API.
func (u *Uploader) UploadBuchungsstapel(ctx context.Context, tenantID uuid.UUID, clientNumber string, csvData []byte) error {
	token, err := u.oauthManager.GetAccessToken(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("datev upload: get token: %w", err)
	}

	uploadURL := fmt.Sprintf("%s/platform/v2/clients/%s/accounting-documents", u.baseURL, clientNumber)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, bytes.NewReader(csvData))
	if err != nil {
		return fmt.Errorf("datev upload: build request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "text/csv; charset=utf-8")
	req.Header.Set("Accept", "application/json")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("datev upload: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("datev upload: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		slog.Error("datev upload failed",
			"status", resp.StatusCode,
			"body", string(respBody),
			"tenant_id", tenantID,
			"client_number", clientNumber,
		)
		return fmt.Errorf("datev upload: API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	slog.Info("datev buchungsstapel uploaded",
		"tenant_id", tenantID,
		"client_number", clientNumber,
		"size_bytes", len(csvData),
	)
	return nil
}

// ListClients retrieves the list of DATEV clients (Mandanten).
func (u *Uploader) ListClients(ctx context.Context, tenantID uuid.UUID) ([]byte, error) {
	token, err := u.oauthManager.GetAccessToken(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("datev list clients: get token: %w", err)
	}

	clientsURL := fmt.Sprintf("%s/platform/v2/clients", u.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, clientsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("datev list clients: build request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("datev list clients: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("datev list clients: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("datev list clients: API returned status %d", resp.StatusCode)
	}

	return body, nil
}
