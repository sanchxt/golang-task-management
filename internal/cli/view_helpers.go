package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"task-management/internal/repository"
)

func lookupViewID(ctx context.Context, repo repository.ViewRepository, viewStr string) (*int64, error) {
	if strings.TrimSpace(viewStr) == "" {
		return nil, nil
	}

	if id, err := strconv.ParseInt(viewStr, 10, 64); err == nil {
		view, err := repo.GetByID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("view with ID %d not found: %w", id, err)
		}
		return &view.ID, nil
	}

	view, err := repo.GetByName(ctx, viewStr)
	if err != nil {
		return nil, fmt.Errorf("view '%s' not found: %w", viewStr, err)
	}

	return &view.ID, nil
}
