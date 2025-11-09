package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"task-management/internal/domain"
	"task-management/internal/repository"
)

type (
	viewsLoadedMsg struct {
		views []*domain.SavedView
		err   error
	}

	favoritesLoadedMsg struct {
		favorites []*domain.SavedView
		err       error
	}

	quickAccessLoadedMsg struct {
		views map[int]*domain.SavedView
		err   error
	}

	viewCreatedMsg struct {
		view *domain.SavedView
		err  error
	}

	viewUpdatedMsg struct {
		view *domain.SavedView
		err  error
	}

	viewDeletedMsg struct {
		viewID int64
		err    error
	}

	viewAppliedMsg struct {
		view *domain.SavedView
		err  error
	}
)

func fetchViewsCmd(ctx context.Context, repo repository.ViewRepository) tea.Cmd {
	return func() tea.Msg {
		views, err := repo.List(ctx, repository.ViewFilter{})
		if err != nil {
			return viewsLoadedMsg{err: err}
		}

		quickAccess := make(map[int]*domain.SavedView)
		favorites := []*domain.SavedView{}

		for _, v := range views {
			if v.HotKey != nil && *v.HotKey >= 1 && *v.HotKey <= 9 {
				quickAccess[*v.HotKey] = v
			}
			if v.IsFavorite {
				favorites = append(favorites, v)
			}
		}

		return viewsLoadedMsg{views: views}
	}
}

func applyViewCmd(ctx context.Context, repo repository.ViewRepository, viewID int64) tea.Cmd {
	return func() tea.Msg {
		view, err := repo.GetByID(ctx, viewID)
		if err != nil {
			return viewAppliedMsg{err: err}
		}

		_ = repo.RecordViewAccess(ctx, viewID)

		return viewAppliedMsg{view: view}
	}
}
