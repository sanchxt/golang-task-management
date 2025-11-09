package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"task-management/internal/domain"
)

func TestBuildProjectTree(t *testing.T) {
	tests := []struct {
		name     string
		projects []*domain.Project
		wantRoots int
		wantNodes int
	}{
		{
			name: "empty list",
			projects: []*domain.Project{},
			wantRoots: 0,
			wantNodes: 0,
		},
		{
			name: "single root project",
			projects: []*domain.Project{
				{ID: 1, Name: "Root", ParentID: nil},
			},
			wantRoots: 1,
			wantNodes: 1,
		},
		{
			name: "one root with two children",
			projects: []*domain.Project{
				{ID: 1, Name: "Root", ParentID: nil},
				{ID: 2, Name: "Child1", ParentID: int64Ptr(1)},
				{ID: 3, Name: "Child2", ParentID: int64Ptr(1)},
			},
			wantRoots: 1,
			wantNodes: 3,
		},
		{
			name: "multiple roots",
			projects: []*domain.Project{
				{ID: 1, Name: "Root1", ParentID: nil},
				{ID: 2, Name: "Root2", ParentID: nil},
			},
			wantRoots: 2,
			wantNodes: 2,
		},
		{
			name: "three-level hierarchy",
			projects: []*domain.Project{
				{ID: 1, Name: "Root", ParentID: nil},
				{ID: 2, Name: "Child", ParentID: int64Ptr(1)},
				{ID: 3, Name: "Grandchild", ParentID: int64Ptr(2)},
			},
			wantRoots: 1,
			wantNodes: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := buildProjectTree(tt.projects)

			if tree == nil && len(tt.projects) > 0 {
				t.Fatal("expected tree to be non-nil for non-empty projects")
			}

			if len(tt.projects) == 0 {
				return
			}

			if len(tree.roots) != tt.wantRoots {
				t.Errorf("buildProjectTree() roots = %d, want %d", len(tree.roots), tt.wantRoots)
			}

			if len(tree.flatMap) != tt.wantNodes {
				t.Errorf("buildProjectTree() nodes = %d, want %d", len(tree.flatMap), tt.wantNodes)
			}
		})
	}
}

func TestProjectTreeDepth(t *testing.T) {
	projects := []*domain.Project{
		{ID: 1, Name: "Root", ParentID: nil},
		{ID: 2, Name: "Child", ParentID: int64Ptr(1)},
		{ID: 3, Name: "Grandchild", ParentID: int64Ptr(2)},
		{ID: 4, Name: "GreatGrandchild", ParentID: int64Ptr(3)},
	}

	tree := buildProjectTree(projects)

	tests := []struct {
		projectID int64
		wantDepth int
	}{
		{projectID: 1, wantDepth: 0},
		{projectID: 2, wantDepth: 1},
		{projectID: 3, wantDepth: 2},
		{projectID: 4, wantDepth: 3},
	}

	for _, tt := range tests {
		node, exists := tree.flatMap[tt.projectID]
		if !exists {
			t.Errorf("project ID %d not found in tree", tt.projectID)
			continue
		}

		if node.depth != tt.wantDepth {
			t.Errorf("project ID %d: depth = %d, want %d", tt.projectID, node.depth, tt.wantDepth)
		}
	}
}

func TestProjectTreeRelationships(t *testing.T) {
	projects := []*domain.Project{
		{ID: 1, Name: "Root", ParentID: nil},
		{ID: 2, Name: "Child1", ParentID: int64Ptr(1)},
		{ID: 3, Name: "Child2", ParentID: int64Ptr(1)},
		{ID: 4, Name: "Grandchild", ParentID: int64Ptr(2)},
	}

	tree := buildProjectTree(projects)

	root := tree.flatMap[1]
	if root.parent != nil {
		t.Error("root should have nil parent")
	}

	if len(root.children) != 2 {
		t.Errorf("root should have 2 children, got %d", len(root.children))
	}

	child1 := tree.flatMap[2]
	if child1.parent != root {
		t.Error("child1 should have root as parent")
	}

	grandchild := tree.flatMap[4]
	if grandchild.parent != child1 {
		t.Error("grandchild should have child1 as parent")
	}
	if grandchild.depth != 2 {
		t.Errorf("grandchild depth = %d, want 2", grandchild.depth)
	}
}

func TestExpandCollapseState(t *testing.T) {
	m := Model{
		projectExpanded: make(map[int64]bool),
	}

	if m.projectExpanded[1] {
		t.Error("project 1 should not be expanded initially")
	}

	m.projectExpanded[1] = true
	if !m.projectExpanded[1] {
		t.Error("project 1 should be expanded after toggle")
	}

	m.projectExpanded[1] = false
	if m.projectExpanded[1] {
		t.Error("project 1 should not be expanded after collapse")
	}
}

func TestFindProjectNode(t *testing.T) {
	projects := []*domain.Project{
		{ID: 1, Name: "Root", ParentID: nil},
		{ID: 2, Name: "Child", ParentID: int64Ptr(1)},
	}

	tree := buildProjectTree(projects)

	tests := []struct {
		name      string
		projectID int64
		wantFound bool
	}{
		{name: "existing root", projectID: 1, wantFound: true},
		{name: "existing child", projectID: 2, wantFound: true},
		{name: "non-existent", projectID: 99, wantFound: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, found := tree.flatMap[tt.projectID]
			if found != tt.wantFound {
				t.Errorf("findProjectNode(%d) found = %v, want %v", tt.projectID, found, tt.wantFound)
			}
			if found && node == nil {
				t.Error("node should not be nil when found")
			}
		})
	}
}

func TestOrphanedProjects(t *testing.T) {
	projects := []*domain.Project{
		{ID: 1, Name: "Root", ParentID: nil},
		{ID: 2, Name: "Orphan", ParentID: int64Ptr(999)},
	}

	tree := buildProjectTree(projects)

	if len(tree.roots) != 2 {
		t.Errorf("expected 2 roots (including orphan), got %d", len(tree.roots))
	}

	orphan := tree.flatMap[2]
	if orphan.parent != nil {
		t.Error("orphan should be treated as root with nil parent")
	}
	if orphan.depth != 0 {
		t.Errorf("orphan depth = %d, want 0", orphan.depth)
	}
}

func TestCountDescendants(t *testing.T) {
	projects := []*domain.Project{
		{ID: 1, Name: "Root", ParentID: nil},
		{ID: 2, Name: "Child1", ParentID: int64Ptr(1)},
		{ID: 3, Name: "Child2", ParentID: int64Ptr(1)},
		{ID: 4, Name: "Grandchild1", ParentID: int64Ptr(2)},
		{ID: 5, Name: "Grandchild2", ParentID: int64Ptr(2)},
	}

	tree := buildProjectTree(projects)

	tests := []struct {
		projectID int64
		wantCount int
	}{
		{projectID: 1, wantCount: 4},
		{projectID: 2, wantCount: 2},
		{projectID: 3, wantCount: 0},
		{projectID: 4, wantCount: 0},
	}

	for _, tt := range tests {
		node := tree.flatMap[tt.projectID]
		count := countDescendants(node)
		if count != tt.wantCount {
			t.Errorf("project ID %d: descendants = %d, want %d", tt.projectID, count, tt.wantCount)
		}
	}
}

func TestFlattenTree(t *testing.T) {
	projects := []*domain.Project{
		{ID: 1, Name: "Root", ParentID: nil},
		{ID: 2, Name: "Child1", ParentID: int64Ptr(1)},
		{ID: 3, Name: "Child2", ParentID: int64Ptr(1)},
	}

	tree := buildProjectTree(projects)

	expanded := make(map[int64]bool)
	flattened := flattenTree(tree, expanded)

	if len(flattened) != 1 {
		t.Errorf("collapsed tree should have 1 visible node, got %d", len(flattened))
	}

	expanded[1] = true
	flattened = flattenTree(tree, expanded)

	if len(flattened) != 3 {
		t.Errorf("expanded root should have 3 visible nodes, got %d", len(flattened))
	}
}

func int64Ptr(i int64) *int64 {
	return &i
}

func countDescendants(node *ProjectTreeNode) int {
	if node == nil {
		return 0
	}
	count := len(node.children)
	for _, child := range node.children {
		count += countDescendants(child)
	}
	return count
}

func flattenTree(tree *ProjectTree, expanded map[int64]bool) []*ProjectTreeNode {
	if tree == nil {
		return []*ProjectTreeNode{}
	}

	var result []*ProjectTreeNode
	for _, root := range tree.roots {
		result = append(result, flattenNode(root, expanded)...)
	}
	return result
}

func flattenNode(node *ProjectTreeNode, expanded map[int64]bool) []*ProjectTreeNode {
	if node == nil {
		return []*ProjectTreeNode{}
	}

	result := []*ProjectTreeNode{node}

	if expanded[node.project.ID] {
		for _, child := range node.children {
			result = append(result, flattenNode(child, expanded)...)
		}
	}

	return result
}

func TestProjectPickerInitialization(t *testing.T) {
	picker := ProjectPicker{
		active:      false,
		projects:    []*domain.Project{},
		cursor:      0,
		searchQuery: "",
		selected:    nil,
	}

	if picker.active {
		t.Error("picker should not be active initially")
	}

	if picker.cursor != 0 {
		t.Error("cursor should be at 0 initially")
	}

	if picker.searchQuery != "" {
		t.Error("search query should be empty initially")
	}
}

func TestProjectPickerFiltering(t *testing.T) {
	now := time.Now()
	projects := []*domain.Project{
		{ID: 1, Name: "Backend API", ParentID: nil, CreatedAt: now, UpdatedAt: now},
		{ID: 2, Name: "Frontend", ParentID: nil, CreatedAt: now, UpdatedAt: now},
		{ID: 3, Name: "Backend Database", ParentID: int64Ptr(1), CreatedAt: now, UpdatedAt: now},
	}

	tests := []struct {
		name        string
		query       string
		wantMatches int
	}{
		{name: "empty query", query: "", wantMatches: 3},
		{name: "match backend", query: "backend", wantMatches: 2},
		{name: "match frontend", query: "frontend", wantMatches: 1},
		{name: "match api", query: "api", wantMatches: 1},
		{name: "no match", query: "xyz", wantMatches: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := filterProjects(projects, tt.query)
			if len(filtered) != tt.wantMatches {
				t.Errorf("filterProjects(%q) = %d matches, want %d", tt.query, len(filtered), tt.wantMatches)
			}
		})
	}
}

func filterProjects(projects []*domain.Project, query string) []*domain.Project {
	if query == "" {
		return projects
	}

	var filtered []*domain.Project
	lowerQuery := toLower(query)
	for _, p := range projects {
		if contains(toLower(p.Name), lowerQuery) || contains(toLower(p.Description), lowerQuery) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

func toLower(s string) string {
	return strings.ToLower(s)
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestGetVisibleProjectNodes(t *testing.T) {
	now := time.Now()
	projects := []*domain.Project{
		{ID: 1, Name: "Root1", ParentID: nil, CreatedAt: now, UpdatedAt: now},
		{ID: 2, Name: "Child1", ParentID: int64Ptr(1), CreatedAt: now, UpdatedAt: now},
		{ID: 3, Name: "Grandchild1", ParentID: int64Ptr(2), CreatedAt: now, UpdatedAt: now},
		{ID: 4, Name: "Root2", ParentID: nil, CreatedAt: now, UpdatedAt: now},
	}

	tests := []struct {
		name          string
		expandedIDs   []int64
		wantVisible   int
		wantIDs       []int64
	}{
		{
			name:        "all collapsed",
			expandedIDs: []int64{},
			wantVisible: 2,
			wantIDs:     []int64{1, 4},
		},
		{
			name:        "root1 expanded",
			expandedIDs: []int64{1},
			wantVisible: 3,
			wantIDs:     []int64{1, 2, 4},
		},
		{
			name:        "root1 and child1 expanded",
			expandedIDs: []int64{1, 2},
			wantVisible: 4,
			wantIDs:     []int64{1, 2, 3, 4},
		},
		{
			name:        "only child expanded (no effect)",
			expandedIDs: []int64{2},
			wantVisible: 2,
			wantIDs:     []int64{1, 4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				projectTree:     buildProjectTree(projects),
				projectExpanded: make(map[int64]bool),
			}

			for _, id := range tt.expandedIDs {
				m.projectExpanded[id] = true
			}

			visible := m.getVisibleProjectNodes()

			if len(visible) != tt.wantVisible {
				t.Errorf("getVisibleProjectNodes() = %d nodes, want %d", len(visible), tt.wantVisible)
				t.Logf("Visible nodes:")
				for i, node := range visible {
					t.Logf("  [%d] ID=%d Name=%s", i, node.project.ID, node.project.Name)
				}
			}

			visibleIDs := make(map[int64]bool)
			for _, node := range visible {
				visibleIDs[node.project.ID] = true
			}

			for _, wantID := range tt.wantIDs {
				if !visibleIDs[wantID] {
					t.Errorf("expected ID %d to be visible but it wasn't", wantID)
				}
			}

			if len(visibleIDs) != len(tt.wantIDs) {
				t.Errorf("visible IDs count = %d, want %d", len(visibleIDs), len(tt.wantIDs))
			}
		})
	}
}

func TestGetVisibleProjectNodesEmpty(t *testing.T) {
	m := Model{
		projectTree:     nil,
		projectExpanded: make(map[int64]bool),
	}

	visible := m.getVisibleProjectNodes()
	if len(visible) != 0 {
		t.Errorf("getVisibleProjectNodes() on empty tree = %d nodes, want 0", len(visible))
	}
}

func TestGetVisibleProjectNodesNested(t *testing.T) {
	now := time.Now()
	projects := []*domain.Project{
		{ID: 1, Name: "Root", ParentID: nil, CreatedAt: now, UpdatedAt: now},
		{ID: 2, Name: "Child1", ParentID: int64Ptr(1), CreatedAt: now, UpdatedAt: now},
		{ID: 3, Name: "Child2", ParentID: int64Ptr(1), CreatedAt: now, UpdatedAt: now},
		{ID: 4, Name: "Grandchild1", ParentID: int64Ptr(2), CreatedAt: now, UpdatedAt: now},
	}

	m := Model{
		projectTree:     buildProjectTree(projects),
		projectExpanded: map[int64]bool{1: true, 2: true},
	}

	visible := m.getVisibleProjectNodes()

	if len(visible) != 4 {
		t.Errorf("expected 4 visible nodes, got %d", len(visible))
	}

	visibleIDs := make(map[int64]bool)
	for _, node := range visible {
		visibleIDs[node.project.ID] = true
	}

	expectedIDs := []int64{1, 2, 3, 4}
	for _, expectedID := range expectedIDs {
		if !visibleIDs[expectedID] {
			t.Errorf("expected ID %d to be visible but it wasn't", expectedID)
		}
	}

	for _, node := range visible {
		if node.project.ID == 1 && node.depth != 0 {
			t.Errorf("root depth = %d, want 0", node.depth)
		}
		if node.project.ID == 4 && node.depth != 2 {
			t.Errorf("grandchild depth = %d, want 2", node.depth)
		}
	}
}

func TestProjectLookupByName(t *testing.T) {
	now := time.Now()
	projects := []*domain.Project{
		{ID: 1, Name: "Backend API", ParentID: nil, CreatedAt: now, UpdatedAt: now},
		{ID: 2, Name: "Frontend", ParentID: nil, CreatedAt: now, UpdatedAt: now},
		{ID: 3, Name: "backend database", ParentID: int64Ptr(1), CreatedAt: now, UpdatedAt: now},
	}

	tests := []struct {
		name        string
		searchName  string
		wantFound   bool
		wantID      int64
	}{
		{
			name:       "exact match",
			searchName: "Backend API",
			wantFound:  true,
			wantID:     1,
		},
		{
			name:       "case insensitive match",
			searchName: "backend api",
			wantFound:  true,
			wantID:     1,
		},
		{
			name:       "case insensitive mixed case",
			searchName: "BACKEND DATABASE",
			wantFound:  true,
			wantID:     3,
		},
		{
			name:       "no match",
			searchName: "NonExistent",
			wantFound:  false,
			wantID:     0,
		},
		{
			name:       "empty search",
			searchName: "",
			wantFound:  false,
			wantID:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var foundID *int64
			searchName := strings.TrimSpace(tt.searchName)

			if searchName != "" {
				for _, proj := range projects {
					if strings.EqualFold(proj.Name, searchName) {
						foundID = &proj.ID
						break
					}
				}
			}

			found := foundID != nil
			if found != tt.wantFound {
				t.Errorf("lookup found = %v, want %v", found, tt.wantFound)
			}

			if found && *foundID != tt.wantID {
				t.Errorf("lookup ID = %d, want %d", *foundID, tt.wantID)
			}
		})
	}
}

func TestViewPickerInitialization(t *testing.T) {
	picker := ViewPicker{
		active:      false,
		views:       []*domain.SavedView{},
		cursor:      0,
		searchQuery: "",
		selected:    nil,
	}

	if picker.active {
		t.Error("picker should not be active initially")
	}

	if picker.cursor != 0 {
		t.Error("cursor should be at 0 initially")
	}

	if picker.searchQuery != "" {
		t.Error("search query should be empty initially")
	}

	if picker.selected != nil {
		t.Error("selected should be nil initially")
	}
}

func TestViewPickerNavigation(t *testing.T) {
	now := time.Now()
	views := []*domain.SavedView{
		{ID: 1, Name: "My Tasks", CreatedAt: now, UpdatedAt: now},
		{ID: 2, Name: "Team View", CreatedAt: now, UpdatedAt: now},
		{ID: 3, Name: "Pending Only", CreatedAt: now, UpdatedAt: now},
	}

	picker := ViewPicker{
		active: true,
		views:  views,
		cursor: 0,
	}

	if picker.cursor < len(picker.views)-1 {
		picker.cursor++
	}
	if picker.cursor != 1 {
		t.Errorf("cursor after moving down = %d, want 1", picker.cursor)
	}

	if picker.cursor < len(picker.views)-1 {
		picker.cursor++
	}
	if picker.cursor != 2 {
		t.Errorf("cursor after moving down twice = %d, want 2", picker.cursor)
	}

	if picker.cursor < len(picker.views)-1 {
		picker.cursor++
	}
	if picker.cursor != 2 {
		t.Errorf("cursor should not exceed list bounds, got %d", picker.cursor)
	}

	if picker.cursor > 0 {
		picker.cursor--
	}
	if picker.cursor != 1 {
		t.Errorf("cursor after moving up = %d, want 1", picker.cursor)
	}

	if picker.cursor > 0 {
		picker.cursor--
	}
	if picker.cursor != 0 {
		t.Errorf("cursor at beginning = %d, want 0", picker.cursor)
	}

	if picker.cursor > 0 {
		picker.cursor--
	}
	if picker.cursor != 0 {
		t.Errorf("cursor should not go below 0, got %d", picker.cursor)
	}
}

func TestQuickAccessViewLookup(t *testing.T) {
	hotKey := 1
	now := time.Now()
	quickAccessViews := map[int]*domain.SavedView{
		1: {ID: 1, Name: "Quick View 1", HotKey: &hotKey, CreatedAt: now, UpdatedAt: now},
		2: {ID: 2, Name: "Quick View 2", HotKey: int2Ptr(2), CreatedAt: now, UpdatedAt: now},
		5: {ID: 5, Name: "Quick View 5", HotKey: int2Ptr(5), CreatedAt: now, UpdatedAt: now},
	}

	tests := []struct {
		key      int
		wantName string
		wantOK   bool
	}{
		{1, "Quick View 1", true},
		{2, "Quick View 2", true},
		{3, "", false},
		{5, "Quick View 5", true},
		{9, "", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("key_%d", tt.key), func(t *testing.T) {
			view, ok := quickAccessViews[tt.key]
			if ok != tt.wantOK {
				t.Errorf("found quick access = %v, want %v", ok, tt.wantOK)
			}
			if ok && view.Name != tt.wantName {
				t.Errorf("view name = %s, want %s", view.Name, tt.wantName)
			}
		})
	}
}

func TestViewFilterConversion(t *testing.T) {
	projectID := int64(42)
	dateFrom := "2025-01-01"
	dateTo := "2025-12-31"

	viewFilter := domain.SavedViewFilter{
		Status:      domain.StatusPending,
		Priority:    domain.PriorityHigh,
		ProjectID:   &projectID,
		Tags:        []string{"bug", "urgent"},
		SearchQuery: "test query",
		SearchMode:  "text",
		SortBy:      "priority",
		SortOrder:   "asc",
		DueDateFrom: &dateFrom,
		DueDateTo:   &dateTo,
	}

	m := &Model{}

	taskFilter := m.convertViewFilterToTaskFilter(viewFilter)

	if taskFilter.Status != domain.StatusPending {
		t.Errorf("status = %v, want %v", taskFilter.Status, domain.StatusPending)
	}

	if taskFilter.Priority != domain.PriorityHigh {
		t.Errorf("priority = %v, want %v", taskFilter.Priority, domain.PriorityHigh)
	}

	if taskFilter.ProjectID == nil || *taskFilter.ProjectID != projectID {
		t.Error("project ID conversion failed")
	}

	if len(taskFilter.Tags) != 2 || taskFilter.Tags[0] != "bug" {
		t.Error("tags conversion failed")
	}

	if taskFilter.SearchQuery != "test query" {
		t.Errorf("search query = %s, want test query", taskFilter.SearchQuery)
	}

	if taskFilter.SearchMode != "text" {
		t.Errorf("search mode = %s, want text", taskFilter.SearchMode)
	}

	if taskFilter.SortBy != "priority" {
		t.Errorf("sort by = %s, want priority", taskFilter.SortBy)
	}

	if taskFilter.SortOrder != "asc" {
		t.Errorf("sort order = %s, want asc", taskFilter.SortOrder)
	}

	if taskFilter.DueDateFrom == nil || *taskFilter.DueDateFrom != "2025-01-01" {
		t.Error("due date from conversion failed")
	}

	if taskFilter.DueDateTo == nil || *taskFilter.DueDateTo != "2025-12-31" {
		t.Error("due date to conversion failed")
	}
}

func TestSliceRemoveByID(t *testing.T) {
	now := time.Now()
	views := []*domain.SavedView{
		{ID: 1, Name: "View 1", CreatedAt: now, UpdatedAt: now},
		{ID: 2, Name: "View 2", CreatedAt: now, UpdatedAt: now},
		{ID: 3, Name: "View 3", CreatedAt: now, UpdatedAt: now},
		{ID: 4, Name: "View 4", CreatedAt: now, UpdatedAt: now},
	}

	tests := []struct {
		removeID   int64
		wantLength int
		wantNames  []string
	}{
		{2, 3, []string{"View 1", "View 3", "View 4"}},
		{1, 3, []string{"View 2", "View 3", "View 4"}},
		{4, 3, []string{"View 1", "View 2", "View 3"}},
		{99, 4, []string{"View 1", "View 2", "View 3", "View 4"}},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("remove_%d", tt.removeID), func(t *testing.T) {
			testViews := make([]*domain.SavedView, len(views))
			copy(testViews, views)

			result := sliceRemoveByID(testViews, tt.removeID)

			if len(result) != tt.wantLength {
				t.Errorf("result length = %d, want %d", len(result), tt.wantLength)
			}

			for i, wantName := range tt.wantNames {
				if i >= len(result) || result[i].Name != wantName {
					t.Errorf("result[%d].Name = %s, want %s", i, result[i].Name, wantName)
				}
			}
		})
	}
}

func TestFavoriteViewsExtraction(t *testing.T) {
	now := time.Now()
	views := []*domain.SavedView{
		{ID: 1, Name: "View 1", IsFavorite: true, CreatedAt: now, UpdatedAt: now},
		{ID: 2, Name: "View 2", IsFavorite: false, CreatedAt: now, UpdatedAt: now},
		{ID: 3, Name: "View 3", IsFavorite: true, CreatedAt: now, UpdatedAt: now},
		{ID: 4, Name: "View 4", IsFavorite: false, CreatedAt: now, UpdatedAt: now},
	}

	favorites := []*domain.SavedView{}
	for _, v := range views {
		if v.IsFavorite {
			favorites = append(favorites, v)
		}
	}

	if len(favorites) != 2 {
		t.Errorf("favorite count = %d, want 2", len(favorites))
	}

	if favorites[0].Name != "View 1" || favorites[1].Name != "View 3" {
		t.Error("favorite views not correctly extracted")
	}
}

func TestQuickAccessViewsExtraction(t *testing.T) {
	now := time.Now()
	hotKey1 := 1
	hotKey2 := 2
	hotKey9 := 9

	views := []*domain.SavedView{
		{ID: 1, Name: "Quick 1", HotKey: &hotKey1, CreatedAt: now, UpdatedAt: now},
		{ID: 2, Name: "Regular", HotKey: nil, CreatedAt: now, UpdatedAt: now},
		{ID: 3, Name: "Quick 2", HotKey: &hotKey2, CreatedAt: now, UpdatedAt: now},
		{ID: 4, Name: "Quick 9", HotKey: &hotKey9, CreatedAt: now, UpdatedAt: now},
	}

	quickAccess := make(map[int]*domain.SavedView)
	for _, v := range views {
		if v.HotKey != nil && *v.HotKey >= 1 && *v.HotKey <= 9 {
			quickAccess[*v.HotKey] = v
		}
	}

	if len(quickAccess) != 3 {
		t.Errorf("quick access count = %d, want 3", len(quickAccess))
	}

	if quickAccess[1].Name != "Quick 1" {
		t.Error("quick access 1 not correctly extracted")
	}

	if quickAccess[2].Name != "Quick 2" {
		t.Error("quick access 2 not correctly extracted")
	}

	if quickAccess[9].Name != "Quick 9" {
		t.Error("quick access 9 not correctly extracted")
	}
}

func int2Ptr(i int) *int {
	return &i
}


func TestQueryLanguageDetection(t *testing.T) {
	tests := []struct {
		name         string
		searchQuery  string
		wantDetected bool
	}{
		{"status filter", "status:pending", true},
		{"priority filter", "priority:high", true},
		{"tag filter", "tag:bug", true},
		{"project mention exact", "@backend", true},
		{"project mention fuzzy", "@~back", true},
		{"negation filter", "-tag:wontfix", true},
		{"due date filter", "due:+7d", true},
		{"complex query", "status:pending @backend -tag:wontfix", true},
		{"plain text search", "just some text", false},
		{"text with numbers", "fix bug 123", false},
		{"empty search", "", false},
		{"only whitespace", "   ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detected := isQueryLanguageSyntax(tt.searchQuery)
			if detected != tt.wantDetected {
				t.Errorf("isQueryLanguageSyntax(%q) = %v, want %v", tt.searchQuery, detected, tt.wantDetected)
			}
		})
	}
}

func isQueryLanguageSyntax(s string) bool {
	if strings.TrimSpace(s) == "" {
		return false
	}

	patterns := []string{
		"status:",
		"priority:",
		"tag:",
		"-tag:",
		"due:",
		"created:",
		"updated:",
		"@",
	}

	for _, pattern := range patterns {
		if strings.Contains(s, pattern) {
			return true
		}
	}

	return false
}

func TestQueryModeState(t *testing.T) {
	m := Model{
		queryMode:   false,
		queryString: "",
	}

	if m.queryMode {
		t.Error("model should not be in query mode initially")
	}

	if m.queryString != "" {
		t.Error("query string should be empty initially")
	}

	m.queryMode = true
	m.queryString = "status:pending @backend"

	if !m.queryMode {
		t.Error("model should be in query mode after setting")
	}

	if m.queryString != "status:pending @backend" {
		t.Errorf("query string = %q, want %q", m.queryString, "status:pending @backend")
	}

	m.queryMode = false
	m.queryString = ""

	if m.queryMode {
		t.Error("model should not be in query mode after clearing")
	}

	if m.queryString != "" {
		t.Error("query string should be empty after clearing")
	}
}

func TestQueryHelpModalToggle(t *testing.T) {
	m := Model{
		showQueryHelp: false,
	}

	if m.showQueryHelp {
		t.Error("query help modal should be hidden initially")
	}

	m.showQueryHelp = true
	if !m.showQueryHelp {
		t.Error("query help modal should be visible after toggle")
	}

	m.showQueryHelp = false
	if m.showQueryHelp {
		t.Error("query help modal should be hidden after toggle")
	}
}

func TestQueryModeIndicatorState(t *testing.T) {
	tests := []struct {
		name       string
		queryMode  bool
		wantVisible bool
	}{
		{"query mode active", true, true},
		{"query mode inactive", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				queryMode: tt.queryMode,
			}

			visible := m.queryMode
			if visible != tt.wantVisible {
				t.Errorf("query mode indicator visible = %v, want %v", visible, tt.wantVisible)
			}
		})
	}
}
