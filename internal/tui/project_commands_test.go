package tui

import (
	"context"
	"errors"
	"testing"
	"time"

	"task-management/internal/domain"
	"task-management/internal/repository"
)

type mockProjectRepository struct {
	projects           []*domain.Project
	createErr          error
	updateErr          error
	deleteErr          error
	listErr            error
	getTaskCountErr    error
	getStatsByStatusErr error
	taskCount          int
	statsByStatus      map[domain.Status]int
}

func (m *mockProjectRepository) Create(ctx context.Context, project *domain.Project) error {
	if m.createErr != nil {
		return m.createErr
	}
	project.ID = int64(len(m.projects) + 1)
	m.projects = append(m.projects, project)
	return nil
}

func (m *mockProjectRepository) GetByID(ctx context.Context, id int64) (*domain.Project, error) {
	for _, p := range m.projects {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, errors.New("project not found")
}

func (m *mockProjectRepository) GetByName(ctx context.Context, name string) (*domain.Project, error) {
	for _, p := range m.projects {
		if p.Name == name {
			return p, nil
		}
	}
	return nil, errors.New("project not found")
}

func (m *mockProjectRepository) Update(ctx context.Context, project *domain.Project) error {
	return m.updateErr
}

func (m *mockProjectRepository) Delete(ctx context.Context, id int64) error {
	return m.deleteErr
}

func (m *mockProjectRepository) List(ctx context.Context, filter repository.ProjectFilter) ([]*domain.Project, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.projects, nil
}

func (m *mockProjectRepository) ListWithHierarchy(ctx context.Context, filter repository.ProjectFilter) ([]*domain.Project, error) {
	return m.List(ctx, filter)
}

func (m *mockProjectRepository) GetChildren(ctx context.Context, parentID int64) ([]*domain.Project, error) {
	var children []*domain.Project
	for _, p := range m.projects {
		if p.ParentID != nil && *p.ParentID == parentID {
			children = append(children, p)
		}
	}
	return children, nil
}

func (m *mockProjectRepository) GetDescendants(ctx context.Context, parentID int64) ([]*domain.Project, error) {
	return nil, nil
}

func (m *mockProjectRepository) GetPath(ctx context.Context, projectID int64) ([]*domain.Project, error) {
	return nil, nil
}

func (m *mockProjectRepository) GetRoots(ctx context.Context) ([]*domain.Project, error) {
	var roots []*domain.Project
	for _, p := range m.projects {
		if p.ParentID == nil {
			roots = append(roots, p)
		}
	}
	return roots, nil
}

func (m *mockProjectRepository) Archive(ctx context.Context, id int64) error {
	return nil
}

func (m *mockProjectRepository) Unarchive(ctx context.Context, id int64) error {
	return nil
}

func (m *mockProjectRepository) SetFavorite(ctx context.Context, id int64, isFavorite bool) error {
	return nil
}

func (m *mockProjectRepository) GetFavorites(ctx context.Context) ([]*domain.Project, error) {
	var favorites []*domain.Project
	for _, p := range m.projects {
		if p.IsFavorite {
			favorites = append(favorites, p)
		}
	}
	return favorites, nil
}

func (m *mockProjectRepository) GetTaskCount(ctx context.Context, projectID int64) (int, error) {
	if m.getTaskCountErr != nil {
		return 0, m.getTaskCountErr
	}
	return m.taskCount, nil
}

func (m *mockProjectRepository) GetTaskCountByStatus(ctx context.Context, projectID int64) (map[domain.Status]int, error) {
	if m.getStatsByStatusErr != nil {
		return nil, m.getStatsByStatusErr
	}
	return m.statsByStatus, nil
}

func (m *mockProjectRepository) Count(ctx context.Context, filter repository.ProjectFilter) (int64, error) {
	return int64(len(m.projects)), nil
}

func (m *mockProjectRepository) ValidateHierarchy(ctx context.Context, projectID int64, parentID int64) error {
	return nil
}

func (m *mockProjectRepository) Search(ctx context.Context, query string, limit int) ([]*domain.Project, error) {
	return m.projects, nil
}

func (m *mockProjectRepository) GetByAlias(ctx context.Context, alias string) (*domain.Project, error) {
	return nil, errors.New("project not found")
}

func (m *mockProjectRepository) ValidateAliasUniqueness(ctx context.Context, alias string, excludeProjectID *int64) error {
	return nil
}

func TestFetchProjectsCmd_Success(t *testing.T) {
	now := time.Now()
	mockRepo := &mockProjectRepository{
		projects: []*domain.Project{
			{ID: 1, Name: "Project 1", CreatedAt: now, UpdatedAt: now},
			{ID: 2, Name: "Project 2", CreatedAt: now, UpdatedAt: now},
		},
	}

	ctx := context.Background()
	filter := repository.ProjectFilter{}

	cmd := fetchProjectsCmd(ctx, mockRepo, filter)
	msg := cmd()

	loadedMsg, ok := msg.(projectsLoadedMsg)
	if !ok {
		t.Fatal("expected projectsLoadedMsg")
	}

	if loadedMsg.err != nil {
		t.Errorf("expected no error, got %v", loadedMsg.err)
	}

	if len(loadedMsg.projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(loadedMsg.projects))
	}
}

func TestFetchProjectsCmd_Error(t *testing.T) {
	mockRepo := &mockProjectRepository{
		listErr: errors.New("database error"),
	}

	ctx := context.Background()
	filter := repository.ProjectFilter{}

	cmd := fetchProjectsCmd(ctx, mockRepo, filter)
	msg := cmd()

	loadedMsg, ok := msg.(projectsLoadedMsg)
	if !ok {
		t.Fatal("expected projectsLoadedMsg")
	}

	if loadedMsg.err == nil {
		t.Error("expected error, got nil")
	}
}

func TestCreateProjectCmd_Success(t *testing.T) {
	now := time.Now()
	mockRepo := &mockProjectRepository{
		projects: []*domain.Project{},
	}

	ctx := context.Background()
	project := &domain.Project{
		Name:      "New Project",
		CreatedAt: now,
		UpdatedAt: now,
	}

	cmd := createProjectCmd(ctx, mockRepo, project)
	msg := cmd()

	createdMsg, ok := msg.(projectCreatedMsg)
	if !ok {
		t.Fatal("expected projectCreatedMsg")
	}

	if createdMsg.err != nil {
		t.Errorf("expected no error, got %v", createdMsg.err)
	}

	if createdMsg.project.ID == 0 {
		t.Error("expected project ID to be set")
	}
}

func TestCreateProjectCmd_Error(t *testing.T) {
	now := time.Now()
	mockRepo := &mockProjectRepository{
		createErr: errors.New("create failed"),
	}

	ctx := context.Background()
	project := &domain.Project{
		Name:      "New Project",
		CreatedAt: now,
		UpdatedAt: now,
	}

	cmd := createProjectCmd(ctx, mockRepo, project)
	msg := cmd()

	createdMsg, ok := msg.(projectCreatedMsg)
	if !ok {
		t.Fatal("expected projectCreatedMsg")
	}

	if createdMsg.err == nil {
		t.Error("expected error, got nil")
	}
}

func TestUpdateProjectCmd_Success(t *testing.T) {
	now := time.Now()
	mockRepo := &mockProjectRepository{}

	ctx := context.Background()
	project := &domain.Project{
		ID:        1,
		Name:      "Updated Project",
		CreatedAt: now,
		UpdatedAt: now,
	}

	cmd := updateProjectCmd(ctx, mockRepo, project)
	msg := cmd()

	updatedMsg, ok := msg.(projectUpdatedMsg)
	if !ok {
		t.Fatal("expected projectUpdatedMsg")
	}

	if updatedMsg.err != nil {
		t.Errorf("expected no error, got %v", updatedMsg.err)
	}
}

func TestDeleteProjectCmd_Success(t *testing.T) {
	mockRepo := &mockProjectRepository{}

	ctx := context.Background()
	projectID := int64(1)

	cmd := deleteProjectCmd(ctx, mockRepo, projectID)
	msg := cmd()

	deletedMsg, ok := msg.(projectDeletedMsg)
	if !ok {
		t.Fatal("expected projectDeletedMsg")
	}

	if deletedMsg.err != nil {
		t.Errorf("expected no error, got %v", deletedMsg.err)
	}

	if deletedMsg.projectID != projectID {
		t.Errorf("expected projectID %d, got %d", projectID, deletedMsg.projectID)
	}
}

func TestFetchProjectStatsCmd_Success(t *testing.T) {
	mockRepo := &mockProjectRepository{
		taskCount: 10,
		statsByStatus: map[domain.Status]int{
			domain.StatusPending:    3,
			domain.StatusInProgress: 2,
			domain.StatusCompleted:  5,
		},
	}

	ctx := context.Background()
	projectID := int64(1)

	cmd := fetchProjectStatsCmd(ctx, mockRepo, projectID)
	msg := cmd()

	statsMsg, ok := msg.(projectStatsMsg)
	if !ok {
		t.Fatal("expected projectStatsMsg")
	}

	if statsMsg.err != nil {
		t.Errorf("expected no error, got %v", statsMsg.err)
	}

	if statsMsg.taskCount != 10 {
		t.Errorf("expected taskCount 10, got %d", statsMsg.taskCount)
	}

	if len(statsMsg.stats) != 3 {
		t.Errorf("expected 3 stats entries, got %d", len(statsMsg.stats))
	}
}

func TestFetchProjectStatsCmd_TaskCountError(t *testing.T) {
	mockRepo := &mockProjectRepository{
		getTaskCountErr: errors.New("task count failed"),
	}

	ctx := context.Background()
	projectID := int64(1)

	cmd := fetchProjectStatsCmd(ctx, mockRepo, projectID)
	msg := cmd()

	statsMsg, ok := msg.(projectStatsMsg)
	if !ok {
		t.Fatal("expected projectStatsMsg")
	}

	if statsMsg.err == nil {
		t.Error("expected error, got nil")
	}
}

func TestFetchProjectStatsCmd_StatsByStatusError(t *testing.T) {
	mockRepo := &mockProjectRepository{
		taskCount:           10,
		getStatsByStatusErr: errors.New("stats failed"),
	}

	ctx := context.Background()
	projectID := int64(1)

	cmd := fetchProjectStatsCmd(ctx, mockRepo, projectID)
	msg := cmd()

	statsMsg, ok := msg.(projectStatsMsg)
	if !ok {
		t.Fatal("expected projectStatsMsg")
	}

	if statsMsg.err == nil {
		t.Error("expected error, got nil")
	}
}
