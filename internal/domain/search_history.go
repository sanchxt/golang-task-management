package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type SearchMode string

type QueryType string

const (
	SearchModeText  SearchMode = "text"
	SearchModeRegex SearchMode = "regex"
	SearchModeFuzzy SearchMode = "fuzzy"
)

const (
	QueryTypeSimple         QueryType = "simple"
	QueryTypeQueryLanguage  QueryType = "query_language"
	QueryTypeProjectMention QueryType = "project_mention"
)

type SearchHistory struct {
	ID             int64      `db:"id" json:"id"`
	QueryText      string     `db:"query_text" json:"query_text"`
	SearchMode     SearchMode `db:"search_mode" json:"search_mode"`
	FuzzyThreshold *int       `db:"fuzzy_threshold" json:"fuzzy_threshold,omitempty"`
	QueryType      QueryType  `db:"query_type" json:"query_type"`
	ProjectFilter  string     `db:"project_filter" json:"project_filter,omitempty"`
	ResultCount    int        `db:"result_count" json:"result_count"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at" json:"updated_at"`
}

func (s *SearchHistory) Validate() error {
	if strings.TrimSpace(s.QueryText) == "" {
		return errors.New("query text cannot be empty")
	}

	if !isValidSearchMode(s.SearchMode) {
		return errors.New("invalid search mode: must be text, regex, or fuzzy")
	}

	if !isValidQueryType(s.QueryType) {
		return errors.New("invalid query type: must be simple, query_language, or project_mention")
	}

	if s.FuzzyThreshold != nil && (*s.FuzzyThreshold < 0 || *s.FuzzyThreshold > 100) {
		return errors.New("fuzzy threshold must be between 0 and 100")
	}

	return nil
}

func (s *SearchHistory) GetModeIndicator() string {
	switch s.SearchMode {
	case SearchModeRegex:
		return "[RE]"
	case SearchModeFuzzy:
		if s.FuzzyThreshold != nil {
			return fmt.Sprintf("[F:%d]", *s.FuzzyThreshold)
		}
		return "[F]"
	default:
		return ""
	}
}

func (s *SearchHistory) GetRelativeTime() string {
	duration := time.Since(s.UpdatedAt)

	seconds := int(duration.Seconds())
	minutes := int(duration.Minutes())
	hours := int(duration.Hours())
	days := int(duration.Hours() / 24)

	switch {
	case seconds < 60:
		return "just now"
	case minutes == 1:
		return "1 minute ago"
	case minutes < 60:
		return fmt.Sprintf("%d minutes ago", minutes)
	case hours == 1:
		return "1 hour ago"
	case hours < 24:
		return fmt.Sprintf("%d hours ago", hours)
	case days == 1:
		return "yesterday"
	case days < 7:
		return fmt.Sprintf("%d days ago", days)
	case days < 30:
		weeks := days / 7
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	case days < 365:
		months := days / 30
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	default:
		years := days / 365
		if years == 1 {
			return "1 year ago"
		}
		return fmt.Sprintf("%d years ago", years)
	}
}

func (s *SearchHistory) GetDisplayText() string {
	var parts []string

	if indicator := s.GetModeIndicator(); indicator != "" {
		parts = append(parts, indicator)
	}

	parts = append(parts, s.QueryText)

	relTime := s.GetRelativeTime()
	if relTime != "" {
		return fmt.Sprintf("%s (%s)", strings.Join(parts, " "), relTime)
	}

	return strings.Join(parts, " ")
}

func NewSearchHistory(queryText string, searchMode SearchMode, queryType QueryType) *SearchHistory {
	now := time.Now()
	return &SearchHistory{
		QueryText:   queryText,
		SearchMode:  searchMode,
		QueryType:   queryType,
		ResultCount: 0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func isValidSearchMode(mode SearchMode) bool {
	switch mode {
	case SearchModeText, SearchModeRegex, SearchModeFuzzy:
		return true
	default:
		return false
	}
}

func isValidQueryType(qType QueryType) bool {
	switch qType {
	case QueryTypeSimple, QueryTypeQueryLanguage, QueryTypeProjectMention:
		return true
	default:
		return false
	}
}
