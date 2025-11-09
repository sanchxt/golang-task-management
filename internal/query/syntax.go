package query

import (
	"fmt"
	"regexp"
	"strings"
)

type ProjectMention struct {
	Name  string
	Fuzzy bool   // true if fuzzy matching (@~), false for exact (@)
}

func (pm ProjectMention) String() string {
	if pm.Fuzzy {
		return "@~" + pm.Name
	}
	return "@" + pm.Name
}

type ProjectMentionQuery struct {
	BaseQuery       string           // query without @mentions
	ProjectMentions []ProjectMention
}

func (pq ProjectMentionQuery) HasProjectFilter() bool {
	return len(pq.ProjectMentions) > 0
}

func (pq ProjectMentionQuery) GetProjectNames() []string {
	names := make([]string, len(pq.ProjectMentions))
	for i, mention := range pq.ProjectMentions {
		names[i] = mention.Name
	}
	return names
}

func (pq ProjectMentionQuery) HasFuzzyProjectFilter() bool {
	for _, mention := range pq.ProjectMentions {
		if mention.Fuzzy {
			return true
		}
	}
	return false
}

var (
	projectMentionRegex = regexp.MustCompile(`@(~)?([a-zA-Z0-9_-]+)`)
)

func ParseProjectMentions(query string) (*ProjectMentionQuery, error) {
	matches := projectMentionRegex.FindAllStringSubmatch(query, -1)

	mentions := make([]ProjectMention, 0, len(matches))
	for _, match := range matches {
		// match[0] -> full match (@~backend or @backend)
		// match[1] -> fuzzy indicator (~) if present
		// match[2] -> project name
		fuzzy := match[1] == "~"
		name := match[2]

		mentions = append(mentions, ProjectMention{
			Name:  name,
			Fuzzy: fuzzy,
		})
	}

	baseQuery := projectMentionRegex.ReplaceAllString(query, "")

	baseQuery = strings.TrimSpace(baseQuery)
	baseQuery = normalizeWhitespace(baseQuery)

	return &ProjectMentionQuery{
		BaseQuery:       baseQuery,
		ProjectMentions: mentions,
	}, nil
}

func normalizeWhitespace(s string) string {
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}

func FormatProjectMentions(mentions []ProjectMention) string {
	if len(mentions) == 0 {
		return ""
	}

	parts := make([]string, len(mentions))
	for i, mention := range mentions {
		parts[i] = mention.String()
	}
	return strings.Join(parts, " ")
}

func ReconstructQuery(pq ProjectMentionQuery) string {
	parts := make([]string, 0, len(pq.ProjectMentions)+1)

	for _, mention := range pq.ProjectMentions {
		parts = append(parts, mention.String())
	}

	if pq.BaseQuery != "" {
		parts = append(parts, pq.BaseQuery)
	}

	return strings.Join(parts, " ")
}

func ValidateProjectName(name string) error {
	if name == "" {
		return fmt.Errorf("project name cannot be empty")
	}

	validPattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validPattern.MatchString(name) {
		return fmt.Errorf("project name '%s' contains invalid characters (only alphanumeric, hyphens, and underscores allowed)", name)
	}

	return nil
}
