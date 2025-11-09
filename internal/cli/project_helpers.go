package cli

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"task-management/internal/domain"
	"task-management/internal/fuzzy"
	"task-management/internal/repository"
)

func lookupProjectID(ctx context.Context, repo repository.ProjectRepository, projectStr string) (*int64, error) {
	if strings.TrimSpace(projectStr) == "" {
		return nil, nil
	}

	if id, err := strconv.ParseInt(projectStr, 10, 64); err == nil {
		project, err := repo.GetByID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("project with ID %d not found: %w", id, err)
		}
		return &project.ID, nil
	}

	project, err := repo.GetByName(ctx, projectStr)
	if err == nil {
		return &project.ID, nil
	}

	project, aliasErr := repo.GetByAlias(ctx, projectStr)
	if aliasErr == nil {
		return &project.ID, nil
	}

	return nil, fmt.Errorf("project '%s' not found (tried name and alias)", projectStr)
}

type projectWithScore struct {
	project *domain.Project
	score   int
}

func lookupProjectByFuzzyName(ctx context.Context, repo repository.ProjectRepository, searchName string, threshold int) (*int64, error) {
	if strings.TrimSpace(searchName) == "" {
		return nil, fmt.Errorf("search name cannot be empty")
	}

	filter := repository.ProjectFilter{
		ExcludeArchived: true,
	}

	projects, err := repo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	if len(projects) == 0 {
		return nil, fmt.Errorf("no matching project found for '%s'", searchName)
	}

	scoredProjects := make([]projectWithScore, 0, len(projects))
	for _, proj := range projects {
		score := fuzzy.Match(searchName, proj.Name)
		if score >= threshold {
			scoredProjects = append(scoredProjects, projectWithScore{
				project: proj,
				score:   score,
			})
		}
	}

	if len(scoredProjects) == 0 {
		return nil, fmt.Errorf("no matching project found for '%s' (threshold: %d)", searchName, threshold)
	}

	sort.Slice(scoredProjects, func(i, j int) bool {
		return scoredProjects[i].score > scoredProjects[j].score
	})

	bestMatch := scoredProjects[0].project
	return &bestMatch.ID, nil
}

func getProjectName(ctx context.Context, repo repository.ProjectRepository, projectID *int64) (string, error) {
	if projectID == nil {
		return "", nil
	}

	project, err := repo.GetByID(ctx, *projectID)
	if err != nil {
		return "", err
	}

	return project.Name, nil
}

func formatProjectDisplay(project *domain.Project) string {
	if project == nil {
		return "-"
	}

	display := project.Name
	if project.Icon != "" {
		display = project.Icon + " " + display
	}

	return display
}
