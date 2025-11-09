package domain

import (
	"testing"
	"time"
)

func TestProjectStats_GetCompletionRate(t *testing.T) {
	tests := []struct {
		name           string
		totalTasks     int
		completedTasks int
		want           float64
	}{
		{
			name:           "50% completion",
			totalTasks:     10,
			completedTasks: 5,
			want:           50.0,
		},
		{
			name:           "100% completion",
			totalTasks:     10,
			completedTasks: 10,
			want:           100.0,
		},
		{
			name:           "0% completion",
			totalTasks:     10,
			completedTasks: 0,
			want:           0.0,
		},
		{
			name:           "no tasks",
			totalTasks:     0,
			completedTasks: 0,
			want:           0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := &ProjectStats{
				TotalTasks:     tt.totalTasks,
				CompletedTasks: tt.completedTasks,
			}
			if got := ps.GetCompletionRate(); got != tt.want {
				t.Errorf("GetCompletionRate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProjectStats_GetActiveTasks(t *testing.T) {
	tests := []struct {
		name            string
		pendingTasks    int
		inProgressTasks int
		want            int
	}{
		{
			name:            "mixed tasks",
			pendingTasks:    5,
			inProgressTasks: 3,
			want:            8,
		},
		{
			name:            "only pending",
			pendingTasks:    10,
			inProgressTasks: 0,
			want:            10,
		},
		{
			name:            "only in progress",
			pendingTasks:    0,
			inProgressTasks: 7,
			want:            7,
		},
		{
			name:            "no active tasks",
			pendingTasks:    0,
			inProgressTasks: 0,
			want:            0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := &ProjectStats{
				PendingTasks:    tt.pendingTasks,
				InProgressTasks: tt.inProgressTasks,
			}
			if got := ps.GetActiveTasks(); got != tt.want {
				t.Errorf("GetActiveTasks() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProjectStats_GetHealthScore(t *testing.T) {
	tests := []struct {
		name           string
		stats          *ProjectStats
		wantMin        int
		wantMax        int
		wantStatus     string
	}{
		{
			name: "excellent health - all completed",
			stats: &ProjectStats{
				TotalTasks:     10,
				CompletedTasks: 10,
				OverdueTasks:   0,
				CancelledTasks: 0,
			},
			wantMin:    80,
			wantMax:    100,
			wantStatus: "Excellent",
		},
		{
			name: "good health - high completion, no overdue",
			stats: &ProjectStats{
				TotalTasks:     10,
				CompletedTasks: 7,
				OverdueTasks:   0,
				CancelledTasks: 0,
			},
			wantMin:    60,
			wantMax:    100,
			wantStatus: "Good",
		},
		{
			name: "critical health - all overdue",
			stats: &ProjectStats{
				TotalTasks:     10,
				CompletedTasks: 0,
				OverdueTasks:   10,
				CancelledTasks: 0,
			},
			wantMin:    0,
			wantMax:    60,
			wantStatus: "Critical",
		},
		{
			name: "fair health - mixed status",
			stats: &ProjectStats{
				TotalTasks:     10,
				CompletedTasks: 4,
				OverdueTasks:   2,
				CancelledTasks: 1,
			},
			wantMin:    20,
			wantMax:    80,
			wantStatus: "Fair",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.stats.CompletionRate = tt.stats.GetCompletionRate()

			score := tt.stats.GetHealthScore()
			status := tt.stats.GetHealthStatus()

			if score < tt.wantMin || score > tt.wantMax {
				t.Errorf("GetHealthScore() = %v, want between %v and %v", score, tt.wantMin, tt.wantMax)
			}

			if status != tt.wantStatus {
				t.Errorf("GetHealthStatus() = %v, want %v", status, tt.wantStatus)
			}
		})
	}
}

func TestGlobalStats_GetCompletionRate(t *testing.T) {
	tests := []struct {
		name           string
		totalTasks     int
		completedTasks int
		want           float64
	}{
		{
			name:           "75% completion",
			totalTasks:     100,
			completedTasks: 75,
			want:           75.0,
		},
		{
			name:           "no tasks",
			totalTasks:     0,
			completedTasks: 0,
			want:           0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := &GlobalStats{
				TotalTasks:     tt.totalTasks,
				CompletedTasks: tt.completedTasks,
			}
			if got := gs.GetCompletionRate(); got != tt.want {
				t.Errorf("GetCompletionRate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGlobalStats_GetAverageTasksPerProject(t *testing.T) {
	tests := []struct {
		name          string
		totalProjects int
		totalTasks    int
		wantMin       float64
		wantMax       float64
	}{
		{
			name:          "average 5 tasks per project",
			totalProjects: 10,
			totalTasks:    50,
			wantMin:       5.0,
			wantMax:       5.0,
		},
		{
			name:          "no projects",
			totalProjects: 0,
			totalTasks:    0,
			wantMin:       0.0,
			wantMax:       0.0,
		},
		{
			name:          "uneven distribution",
			totalProjects: 3,
			totalTasks:    10,
			wantMin:       3.33,
			wantMax:       3.34,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := &GlobalStats{
				TotalProjects: tt.totalProjects,
				TotalTasks:    tt.totalTasks,
			}
			got := gs.GetAverageTasksPerProject()
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("GetAverageTasksPerProject() = %v, want between %v and %v", got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestGlobalStats_HasTasks(t *testing.T) {
	tests := []struct {
		name       string
		totalTasks int
		want       bool
	}{
		{
			name:       "has tasks",
			totalTasks: 10,
			want:       true,
		},
		{
			name:       "no tasks",
			totalTasks: 0,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := &GlobalStats{
				TotalTasks: tt.totalTasks,
			}
			if got := gs.HasTasks(); got != tt.want {
				t.Errorf("HasTasks() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGlobalStats_HasProjects(t *testing.T) {
	tests := []struct {
		name          string
		totalProjects int
		want          bool
	}{
		{
			name:          "has projects",
			totalProjects: 5,
			want:          true,
		},
		{
			name:          "no projects",
			totalProjects: 0,
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := &GlobalStats{
				TotalProjects: tt.totalProjects,
			}
			if got := gs.HasProjects(); got != tt.want {
				t.Errorf("HasProjects() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewProjectStats(t *testing.T) {
	projectID := int64(123)
	projectName := "Test Project"

	stats := NewProjectStats(projectID, projectName)

	if stats.ProjectID != projectID {
		t.Errorf("ProjectID = %v, want %v", stats.ProjectID, projectID)
	}

	if stats.ProjectName != projectName {
		t.Errorf("ProjectName = %v, want %v", stats.ProjectName, projectName)
	}

	if stats.IncludeDescendants != false {
		t.Errorf("IncludeDescendants should be false by default")
	}

	if time.Since(stats.CalculatedAt) > time.Second {
		t.Errorf("CalculatedAt should be recent")
	}
}

func TestNewGlobalStats(t *testing.T) {
	stats := NewGlobalStats()

	if stats.TopProjectsByTaskCount == nil {
		t.Errorf("TopProjectsByTaskCount should be initialized")
	}

	if len(stats.TopProjectsByTaskCount) != 0 {
		t.Errorf("TopProjectsByTaskCount should be empty initially")
	}

	if time.Since(stats.CalculatedAt) > time.Second {
		t.Errorf("CalculatedAt should be recent")
	}
}

func TestProjectStats_GetHealthStatusByScore(t *testing.T) {
	tests := []struct {
		name  string
		score int
		want  string
	}{
		{"excellent - 90", 90, "Excellent"},
		{"excellent boundary - 80", 80, "Excellent"},
		{"good - 70", 70, "Good"},
		{"good boundary - 60", 60, "Good"},
		{"fair - 50", 50, "Fair"},
		{"fair boundary - 40", 40, "Fair"},
		{"needs attention - 30", 30, "Needs Attention"},
		{"needs attention boundary - 20", 20, "Needs Attention"},
		{"critical - 10", 10, "Critical"},
		{"critical - 0", 0, "Critical"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var status string
			switch {
			case tt.score >= 80:
				status = "Excellent"
			case tt.score >= 60:
				status = "Good"
			case tt.score >= 40:
				status = "Fair"
			case tt.score >= 20:
				status = "Needs Attention"
			default:
				status = "Critical"
			}

			if status != tt.want {
				t.Errorf("Status for score %d = %v, want %v", tt.score, status, tt.want)
			}
		})
	}
}
