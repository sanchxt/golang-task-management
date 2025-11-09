package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"task-management/internal/domain"
)

func (m Model) renderProjectView() string {
	if m.projectTree == nil || len(m.projectTree.roots) == 0 {
		return m.renderEmptyProjectView()
	}

	width := m.width
	leftWidth := int(float64(width) * 0.6)
	rightWidth := width - leftWidth - 4

	tree := m.renderProjectTree(leftWidth)

	details := ""
	if m.selectedProject != nil {
		details = m.renderProjectDetails(rightWidth)
	} else {
		details = m.renderNoProjectSelected(rightWidth)
	}

	treeStyle := lipgloss.NewStyle().
		Width(leftWidth).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(m.theme.BorderColor)).
		Padding(0, 1)

	detailsStyle := lipgloss.NewStyle().
		Width(rightWidth).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(m.theme.BorderColor)).
		Padding(0, 1)

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		treeStyle.Render(tree),
		detailsStyle.Render(details),
	)
}

func (m Model) renderProjectTree(width int) string {
	var output strings.Builder

	title := m.styles.Title.Render("Projects")
	output.WriteString(title)
	output.WriteString("\n\n")

	visibleNodes := m.flattenTreeForDisplay()

	if len(visibleNodes) == 0 {
		output.WriteString(m.styles.Info.Render("No projects found."))
		return output.String()
	}

	for i, node := range visibleNodes {
		isSelected := m.selectedProject != nil && node.project.ID == m.selectedProject.ID
		isCursor := i == m.projectCursor

		line := m.renderProjectNode(node, isSelected, isCursor, width-4)
		output.WriteString(line)
		output.WriteString("\n")
	}

	output.WriteString("\n")
	totalProjects := len(m.projects)
	visibleCount := len(visibleNodes)
	footer := m.styles.Subtitle.Render(fmt.Sprintf("Showing %d of %d project(s)", visibleCount, totalProjects))
	output.WriteString(footer)

	return output.String()
}

func (m Model) flattenTreeForDisplay() []*ProjectTreeNode {
	var result []*ProjectTreeNode

	for _, root := range m.projectTree.roots {
		result = append(result, m.flattenNodeForDisplay(root)...)
	}

	return result
}

func (m Model) flattenNodeForDisplay(node *ProjectTreeNode) []*ProjectTreeNode {
	if node == nil {
		return []*ProjectTreeNode{}
	}

	result := []*ProjectTreeNode{node}

	if m.projectExpanded[node.project.ID] {
		for _, child := range node.children {
			result = append(result, m.flattenNodeForDisplay(child)...)
		}
	}

	return result
}

func (m Model) renderProjectNode(node *ProjectTreeNode, isSelected bool, isCursor bool, width int) string {
	prefix := m.buildTreePrefix(node)

	expandIndicator := ""
	if len(node.children) > 0 {
		if m.projectExpanded[node.project.ID] {
			expandIndicator = "▼ "
		} else {
			expandIndicator = "▶ "
		}
	} else {
		expandIndicator = "  "
	}

	icon := node.project.Icon
	if icon == "" {
		icon = "📦"
	}

	name := node.project.Name
	maxNameLen := width - len(prefix) - len(expandIndicator) - len(icon) - 10
	if len(name) > maxNameLen && maxNameLen > 3 {
		name = name[:maxNameLen-3] + "..."
	}

	statusIndicator := ""
	switch node.project.Status {
		case domain.ProjectStatusArchived:
			statusIndicator = m.styles.Info.Render(" [archived]")
		case domain.ProjectStatusCompleted:
			statusIndicator = m.styles.Success.Render(" [✓]")
	}

	favoriteIndicator := ""
	if node.project.IsFavorite {
		favoriteIndicator = m.styles.UrgentText.Render(" ★")
	}

	line := prefix + expandIndicator + icon + " " + name + statusIndicator + favoriteIndicator

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.SelectedFg)).
		Background(lipgloss.Color(m.theme.SelectedBg))

	if isCursor && isSelected {
		return selectedStyle.Bold(true).Render("▶ " + line)
	} else if isCursor {
		return selectedStyle.Render("▶ " + line)
	} else if isSelected {
		return selectedStyle.Bold(false).Render("  " + line)
	}

	return "  " + line
}

func (m Model) buildTreePrefix(node *ProjectTreeNode) string {
	if node.depth == 0 {
		return ""
	}

	prefix := ""
	current := node

	var ancestors []*ProjectTreeNode
	for current.parent != nil {
		ancestors = append([]*ProjectTreeNode{current.parent}, ancestors...)
		current = current.parent
	}

	for _, ancestor := range ancestors {
		hasNextSibling := m.hasNextSibling(ancestor)
		if hasNextSibling {
			prefix += "│   "
		} else {
			prefix += "    "
		}
	}

	hasNextSibling := m.hasNextSibling(node)
	if hasNextSibling {
		prefix += "├── "
	} else {
		prefix += "└── "
	}

	return prefix
}

func (m Model) hasNextSibling(node *ProjectTreeNode) bool {
	if node.parent == nil {
		for i, root := range m.projectTree.roots {
			if root == node && i < len(m.projectTree.roots)-1 {
				return true
			}
		}
		return false
	}

	for i, child := range node.parent.children {
		if child == node && i < len(node.parent.children)-1 {
			return true
		}
	}
	return false
}

func (m Model) renderProjectDetails(width int) string {
	if m.selectedProject == nil {
		return m.renderNoProjectSelected(width)
	}

	project := m.selectedProject
	var output strings.Builder

	icon := project.Icon
	if icon == "" {
		icon = "📦"
	}
	title := m.styles.Title.Render(fmt.Sprintf("%s %s", icon, project.Name))
	output.WriteString(title)
	output.WriteString("\n\n")

	output.WriteString(m.styles.DetailLabel.Render("ID: "))
	output.WriteString(m.styles.DetailValue.Render(fmt.Sprintf("%d", project.ID)))
	output.WriteString("\n")

	output.WriteString(m.styles.DetailLabel.Render("Status: "))
	statusStyle := m.styles.DetailValue
	switch project.Status {
		case domain.ProjectStatusActive:
			statusStyle = m.styles.Success
		case domain.ProjectStatusArchived:
			statusStyle = m.styles.Info
		case domain.ProjectStatusCompleted:
			statusStyle = m.styles.Success
	}
	output.WriteString(statusStyle.Render(string(project.Status)))
	output.WriteString("\n")

	if project.ParentID != nil {
		output.WriteString(m.styles.DetailLabel.Render("Path: "))
		path := m.buildProjectPath(project)
		output.WriteString(m.styles.DetailValue.Render(path))
		output.WriteString("\n")
	} else {
		output.WriteString(m.styles.DetailLabel.Render("Type: "))
		output.WriteString(m.styles.DetailValue.Render("Root Project"))
		output.WriteString("\n")
	}

	if project.Description != "" {
		output.WriteString("\n")
		output.WriteString(m.styles.DetailLabel.Render("Description:"))
		output.WriteString("\n")
		desc := wrapText(project.Description, width-2)
		output.WriteString(m.styles.DetailValue.Render(desc))
		output.WriteString("\n")
	}

	if project.Color != "" {
		output.WriteString("\n")
		output.WriteString(m.styles.DetailLabel.Render("Color: "))
		output.WriteString(m.styles.DetailValue.Render(project.Color))
		output.WriteString("\n")
	}

	if len(project.Aliases) > 0 {
		output.WriteString("\n")
		output.WriteString(m.styles.DetailLabel.Render("Aliases: "))
		output.WriteString(m.styles.DetailValue.Render(project.FormatAliases()))
		output.WriteString("\n")
	}

	output.WriteString(m.styles.DetailLabel.Render("Favorite: "))
	if project.IsFavorite {
		output.WriteString(m.styles.UrgentText.Render("★ Yes"))
	} else {
		output.WriteString(m.styles.DetailValue.Render("No"))
	}
	output.WriteString("\n")

	if project.HasNotes() {
		output.WriteString("\n")
		output.WriteString(m.styles.DetailLabel.Render("Notes:"))
		output.WriteString("\n")

		notesPreview := strings.TrimSpace(project.Notes)
		lines := strings.Split(notesPreview, "\n")

		previewLines := 3
		if len(lines) > previewLines {
			for i := 0; i < previewLines; i++ {
				wrapped := wrapText(lines[i], width-4)
				output.WriteString(m.styles.DetailValue.Render("  " + wrapped))
				output.WriteString("\n")
			}
			output.WriteString(m.styles.Info.Render(fmt.Sprintf("  ... (%d more lines, press 'M' to view full notes)", len(lines)-previewLines)))
			output.WriteString("\n")
		} else {
			for _, line := range lines {
				wrapped := wrapText(line, width-4)
				output.WriteString(m.styles.DetailValue.Render("  " + wrapped))
				output.WriteString("\n")
			}
			output.WriteString(m.styles.Info.Render("  (Press 'M' to view notes)"))
			output.WriteString("\n")
		}
	}

	output.WriteString("\n")
	output.WriteString(m.styles.DetailLabel.Render("Created: "))
	output.WriteString(m.styles.DetailValue.Render(project.CreatedAt.Format("2006-01-02 15:04")))
	output.WriteString("\n")
	output.WriteString(m.styles.DetailLabel.Render("Updated: "))
	output.WriteString(m.styles.DetailValue.Render(project.UpdatedAt.Format("2006-01-02 15:04")))
	output.WriteString("\n")

	node := m.projectTree.flatMap[project.ID]
	if node != nil && len(node.children) > 0 {
		output.WriteString("\n")
		output.WriteString(m.styles.Subtitle.Render(fmt.Sprintf("Child Projects (%d):", len(node.children))))
		output.WriteString("\n")
		for _, child := range node.children {
			childIcon := child.project.Icon
			if childIcon == "" {
				childIcon = "📦"
			}
			output.WriteString(m.styles.Info.Render(fmt.Sprintf("  %s %s", childIcon, child.project.Name)))
			output.WriteString("\n")
		}
	}

	if stats, ok := m.projectStats[project.ID]; ok {
		output.WriteString("\n")
		output.WriteString(m.styles.Subtitle.Render("Task Statistics:"))
		output.WriteString("\n")

		output.WriteString(m.styles.DetailLabel.Render("  Total Tasks: "))
		output.WriteString(m.styles.DetailValue.Render(fmt.Sprintf("%d", stats.taskCount)))
		output.WriteString("\n")

		if len(stats.stats) > 0 {
			output.WriteString("\n")
			output.WriteString(m.styles.DetailLabel.Render("  By Status:"))
			output.WriteString("\n")

			statusOrder := []domain.Status{
				domain.StatusPending,
				domain.StatusInProgress,
				domain.StatusCompleted,
				domain.StatusCancelled,
			}

			for _, status := range statusOrder {
				if count, exists := stats.stats[status]; exists && count > 0 {
					statusStyle := m.styles.GetStatusStyle(status)
					statusText := statusStyle.Render(string(status))
					output.WriteString(fmt.Sprintf("    %s: %s\n", statusText, m.styles.DetailValue.Render(fmt.Sprintf("%d", count))))
				}
			}
		}
	}

	return output.String()
}

func (m Model) buildProjectPath(project *domain.Project) string {
	if project.ParentID == nil {
		return project.Name
	}

	var parts []string
	current := project

	for current != nil {
		parts = append([]string{current.Name}, parts...)
		if current.ParentID == nil {
			break
		}
		node := m.projectTree.flatMap[*current.ParentID]
		if node == nil {
			break
		}
		current = node.project
	}

	return strings.Join(parts, " > ")
}

func (m Model) renderNoProjectSelected(width int) string {
	var output strings.Builder

	output.WriteString(m.styles.Title.Render("Project Details"))
	output.WriteString("\n\n")
	output.WriteString(m.styles.Info.Render("No project selected."))
	output.WriteString("\n\n")
	output.WriteString(m.styles.Subtitle.Render("Navigation:"))
	output.WriteString("\n")
	output.WriteString("  ↑/↓ or j/k  - Navigate projects\n")
	output.WriteString("  →/l         - Expand project\n")
	output.WriteString("  ←/h         - Collapse project\n")
	output.WriteString("  Enter       - Select project\n")
	output.WriteString("  P           - Back to tasks\n")

	return output.String()
}

func (m Model) renderEmptyProjectView() string {
	var output strings.Builder

	output.WriteString("\n\n")
	output.WriteString(m.styles.Title.Render("Projects"))
	output.WriteString("\n\n")
	output.WriteString(m.styles.Info.Render("No projects found."))
	output.WriteString("\n\n")
	output.WriteString(m.styles.Subtitle.Render("Get started:"))
	output.WriteString("\n")
	output.WriteString("  Use 'taskflow project add \"Project Name\"' to create a project\n")
	output.WriteString("\n\n")
	output.WriteString(m.styles.Info.Render("Press 'P' to return to tasks view"))

	return output.String()
}
