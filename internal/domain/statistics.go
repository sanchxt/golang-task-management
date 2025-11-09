package domain

import (
	"fmt"
	"time"
)

type ProjectStats struct {
	ProjectID   int64     `json:"project_id"`
	ProjectName string    `json:"project_name"`
	ProjectPath string    `json:"project_path"`

	TotalTasks       int `json:"total_tasks"`
	PendingTasks     int `json:"pending_tasks"`
	InProgressTasks  int `json:"in_progress_tasks"`
	CompletedTasks   int `json:"completed_tasks"`
	CancelledTasks   int `json:"cancelled_tasks"`

	LowPriorityTasks    int `json:"low_priority_tasks"`
	MediumPriorityTasks int `json:"medium_priority_tasks"`
	HighPriorityTasks   int `json:"high_priority_tasks"`
	UrgentPriorityTasks int `json:"urgent_priority_tasks"`

	OverdueTasks     int       `json:"overdue_tasks"`
	DueSoonTasks     int       `json:"due_soon_tasks"` // within 7 days
	RecentTasks      int       `json:"recent_tasks"`   // in last 7 days
	RecentlyUpdated  int       `json:"recently_updated"` // in last 7 days

	CompletionRate   float64   `json:"completion_rate"`

	IncludeDescendants bool `json:"include_descendants"`
	DescendantCount    int  `json:"descendant_count,omitempty"`

	CalculatedAt time.Time `json:"calculated_at"`
}

type GlobalStats struct {
	TotalProjects      int `json:"total_projects"`
	ActiveProjects     int `json:"active_projects"`
	ArchivedProjects   int `json:"archived_projects"`
	CompletedProjects  int `json:"completed_projects"`
	FavoriteProjects   int `json:"favorite_projects"`

	TotalTasks         int `json:"total_tasks"`
	PendingTasks       int `json:"pending_tasks"`
	InProgressTasks    int `json:"in_progress_tasks"`
	CompletedTasks     int `json:"completed_tasks"`
	CancelledTasks     int `json:"cancelled_tasks"`

	LowPriorityTasks    int `json:"low_priority_tasks"`
	MediumPriorityTasks int `json:"medium_priority_tasks"`
	HighPriorityTasks   int `json:"high_priority_tasks"`
	UrgentPriorityTasks int `json:"urgent_priority_tasks"`

	OverdueTasks        int `json:"overdue_tasks"`
	DueSoonTasks        int `json:"due_soon_tasks"`
	RecentTasks         int `json:"recent_tasks"`

	OverallCompletionRate float64 `json:"overall_completion_rate"`

	TopProjectsByTaskCount []ProjectTaskCount `json:"top_projects_by_task_count,omitempty"`

	TotalViews         int `json:"total_views"`
	FavoriteViews      int `json:"favorite_views"`

	TotalTemplates     int `json:"total_templates"`

	CalculatedAt time.Time `json:"calculated_at"`
}

type ProjectTaskCount struct {
	ProjectID   int64  `json:"project_id"`
	ProjectName string `json:"project_name"`
	TaskCount   int    `json:"task_count"`
	Icon        string `json:"icon,omitempty"`
}

func (ps *ProjectStats) GetCompletionRate() float64 {
	if ps.TotalTasks == 0 {
		return 0.0
	}
	return (float64(ps.CompletedTasks) / float64(ps.TotalTasks)) * 100.0
}

func (ps *ProjectStats) GetActiveTasks() int {
	return ps.PendingTasks + ps.InProgressTasks
}

func (ps *ProjectStats) GetHealthScore() int {
	if ps.TotalTasks == 0 {
		return 100
	}

	score := 100.0

	if ps.OverdueTasks > 0 {
		overdueRatio := float64(ps.OverdueTasks) / float64(ps.TotalTasks)
		score -= overdueRatio * 50.0
	}

	score += (ps.CompletionRate * 0.3)

	if ps.CancelledTasks > 0 {
		cancelledRatio := float64(ps.CancelledTasks) / float64(ps.TotalTasks)
		score -= cancelledRatio * 20.0
	}

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return int(score)
}

func (ps *ProjectStats) GetHealthStatus() string {
	score := ps.GetHealthScore()

	switch {
	case score >= 80:
		return "Excellent"
	case score >= 60:
		return "Good"
	case score >= 40:
		return "Fair"
	case score >= 20:
		return "Needs Attention"
	default:
		return "Critical"
	}
}

func (ps *ProjectStats) GetPriorityDistribution() string {
	if ps.TotalTasks == 0 {
		return "No tasks"
	}

	return fmt.Sprintf("Low: %d, Medium: %d, High: %d, Urgent: %d",
		ps.LowPriorityTasks, ps.MediumPriorityTasks, ps.HighPriorityTasks, ps.UrgentPriorityTasks)
}

func (ps *ProjectStats) GetStatusDistribution() string {
	if ps.TotalTasks == 0 {
		return "No tasks"
	}

	return fmt.Sprintf("Pending: %d, In Progress: %d, Completed: %d, Cancelled: %d",
		ps.PendingTasks, ps.InProgressTasks, ps.CompletedTasks, ps.CancelledTasks)
}

func (gs *GlobalStats) GetCompletionRate() float64 {
	if gs.TotalTasks == 0 {
		return 0.0
	}
	return (float64(gs.CompletedTasks) / float64(gs.TotalTasks)) * 100.0
}

func (gs *GlobalStats) GetActiveTasks() int {
	return gs.PendingTasks + gs.InProgressTasks
}

func (gs *GlobalStats) GetAverageTasksPerProject() float64 {
	if gs.TotalProjects == 0 {
		return 0.0
	}
	return float64(gs.TotalTasks) / float64(gs.TotalProjects)
}

func (gs *GlobalStats) HasTasks() bool {
	return gs.TotalTasks > 0
}

func (gs *GlobalStats) HasProjects() bool {
	return gs.TotalProjects > 0
}

func NewProjectStats(projectID int64, projectName string) *ProjectStats {
	return &ProjectStats{
		ProjectID:          projectID,
		ProjectName:        projectName,
		CalculatedAt:       time.Now(),
		IncludeDescendants: false,
	}
}

func NewGlobalStats() *GlobalStats {
	return &GlobalStats{
		CalculatedAt:           time.Now(),
		TopProjectsByTaskCount: make([]ProjectTaskCount, 0),
	}
}
