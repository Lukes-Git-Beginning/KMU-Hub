package holiday

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

const defaultBaseURL = "https://date.nager.at"

// NagerHoliday represents a single holiday from the Nager.Date API response
type NagerHoliday struct {
	Date        string   `json:"date"`
	LocalName   string   `json:"localName"`
	Name        string   `json:"name"`
	CountryCode string   `json:"countryCode"`
	Fixed       bool     `json:"fixed"`
	Global      bool     `json:"global"`
	Counties    []string `json:"counties"` // ISO-3166-2 subdivision codes
	LaunchYear  *int     `json:"launchYear"`
	Types       []string `json:"types"` // "Public", "Bank", etc.
}

// NagerClient is an HTTP client for the Nager.Date public holidays API
type NagerClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewNagerClient creates a new Nager.Date API client
func NewNagerClient() *NagerClient {
	return &NagerClient{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		baseURL:    defaultBaseURL,
	}
}

// NewNagerClientWithURL creates a NagerClient with a custom base URL (for testing)
func NewNagerClientWithURL(baseURL string) *NagerClient {
	return &NagerClient{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		baseURL:    baseURL,
	}
}

// FetchHolidays retrieves public holidays for a given year and country code
func (c *NagerClient) FetchHolidays(ctx context.Context, year int, countryCode string) ([]NagerHoliday, error) {
	url := fmt.Sprintf("%s/api/v3/PublicHolidays/%d/%s", c.baseURL, year, countryCode)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "KMUHub/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch holidays: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("nager API returned status %d", resp.StatusCode)
	}

	var holidays []NagerHoliday
	if decodeErr := json.NewDecoder(resp.Body).Decode(&holidays); decodeErr != nil {
		return nil, fmt.Errorf("decode response: %w", decodeErr)
	}

	return holidays, nil
}

// MapToModels converts Nager API responses to internal models
func MapToModels(nagerHolidays []NagerHoliday) ([]models.PublicHoliday, error) {
	result := make([]models.PublicHoliday, 0, len(nagerHolidays))

	for _, nh := range nagerHolidays {
		date, err := time.Parse("2006-01-02", nh.Date)
		if err != nil {
			return nil, fmt.Errorf("parse date %q: %w", nh.Date, err)
		}

		holidayType := "public"
		if len(nh.Types) > 0 {
			holidayType = strings.ToLower(nh.Types[0])
		}

		h := models.PublicHoliday{
			ID:               uuid.New(),
			Date:             date,
			Name:             nh.Name,
			LocalName:        nh.LocalName,
			CountryCode:      nh.CountryCode,
			IsGlobal:         nh.Global,
			SubdivisionCodes: nh.Counties,
			HolidayType:      holidayType,
			Year:             date.Year(),
		}

		result = append(result, h)
	}

	return result, nil
}
