package models

// EntityType represents the type of entity for search
type SearchEntityType string

const (
	SearchEntityContact  SearchEntityType = "contact"
	SearchEntityCompany  SearchEntityType = "company"
	SearchEntityDeal     SearchEntityType = "deal"
	SearchEntityActivity SearchEntityType = "activity"
)

// ValidSearchEntityTypes for validation
var ValidSearchEntityTypes = map[SearchEntityType]bool{
	SearchEntityContact:  true,
	SearchEntityCompany:  true,
	SearchEntityDeal:     true,
	SearchEntityActivity: true,
}

// IsValid checks if the search entity type is valid
func (et SearchEntityType) IsValid() bool {
	return ValidSearchEntityTypes[et]
}

// SearchResult represents a single search result
type SearchResult struct {
	ID         string  `json:"id"`
	EntityType string  `json:"entity_type"` // contact, company, deal, activity
	Title      string  `json:"title"`       // Primary display text
	Subtitle   string  `json:"subtitle"`    // Secondary info (e.g., email, company name)
	Score      float64 `json:"score"`       // Relevance score from full-text search
}
