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

type BelegbilderUploader struct {
	oauthManager *OAuthManager
	httpClient   *http.Client
	baseURL      string
}

func NewBelegbilderUploader(oauthManager *OAuthManager, baseURL string) *BelegbilderUploader {
	return &BelegbilderUploader{
		oauthManager: oauthManager,
		httpClient:   &http.Client{Timeout: 60 * time.Second},
		baseURL:      baseURL,
	}
}

// UploadBeleg uploads a PDF document as a Belegbild.
func (bu *BelegbilderUploader) UploadBeleg(ctx context.Context, tenantID uuid.UUID, clientNumber string, pdfData []byte, filename string) error {
	token, err := bu.oauthManager.GetAccessToken(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("datev beleg upload: get token: %w", err)
	}

	uploadURL := fmt.Sprintf("%s/platform/v2/clients/%s/dms/incoming-documents", bu.baseURL, clientNumber)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, bytes.NewReader(pdfData))
	if err != nil {
		return fmt.Errorf("datev beleg upload: build request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/pdf")
	req.Header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	req.Header.Set("Accept", "application/json")

	resp, err := bu.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("datev beleg upload: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("datev beleg upload: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		slog.Error("datev beleg upload failed",
			"status", resp.StatusCode,
			"body", string(respBody),
			"tenant_id", tenantID,
			"filename", filename,
		)
		return fmt.Errorf("datev beleg upload: API returned status %d", resp.StatusCode)
	}

	slog.Info("datev belegbild uploaded",
		"tenant_id", tenantID,
		"client_number", clientNumber,
		"filename", filename,
		"size_bytes", len(pdfData),
	)
	return nil
}
