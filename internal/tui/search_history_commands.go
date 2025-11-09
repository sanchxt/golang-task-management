package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"task-management/internal/domain"
	"task-management/internal/repository"
)

type (
	searchHistoryLoadedMsg struct {
		history []*domain.SearchHistory
		err     error
	}

	searchRecordedMsg struct {
		entry *domain.SearchHistory
		err   error
	}
)

func fetchSearchHistoryCmd(ctx context.Context, repo repository.SearchHistoryRepository, limit int) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return searchHistoryLoadedMsg{history: []*domain.SearchHistory{}}
		}

		history, err := repo.List(ctx, limit)
		if err != nil {
			return searchHistoryLoadedMsg{err: err}
		}

		return searchHistoryLoadedMsg{history: history}
	}
}

func recordSearchCmd(ctx context.Context, repo repository.SearchHistoryRepository, entry *domain.SearchHistory) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return searchRecordedMsg{}
		}

		err := repo.RecordSearch(ctx, entry)
		if err != nil {
			return searchRecordedMsg{err: err}
		}

		return searchRecordedMsg{entry: entry}
	}
}
