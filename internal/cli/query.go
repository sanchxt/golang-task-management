package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query language help and utilities",
	Long:  `Display query language syntax help and provide query-related utilities.`,
}

var queryHelpCmd = &cobra.Command{
	Use:   "help",
	Short: "Display query language syntax reference",
	Long:  `Display comprehensive query language syntax reference with examples.`,
	Run: func(cmd *cobra.Command, args []string) {
		printQueryHelp()
	},
}

func init() {
	rootCmd.AddCommand(queryCmd)
	queryCmd.AddCommand(queryHelpCmd)
}

func printQueryHelp() {
	help := `
Query Language Syntax Reference

The query language allows you to build complex filters using a simple syntax.
Use it with: taskflow list --query "your query here"

FIELD FILTERS:
  status:<value>       Filter by status (pending, in_progress, completed, cancelled)
  priority:<value>     Filter by priority (low, medium, high, urgent)
  tag:<value>          Filter by tag
  project:<name>       Filter by project name

NEGATION:
  -tag:<value>         Exclude tasks with tag
  -status:<value>      Exclude tasks with status
  -priority:<value>    Exclude tasks with priority

PROJECT MENTIONS:
  @<name>              Exact project name match
  @~<name>             Fuzzy project name match (typo-tolerant)

DATE FILTERS:
  due:<date>           Due on specific date (YYYY-MM-DD)
  due:+<N>d            Due in next N days
  due:-<N>d            Due in last N days (overdue)
  due:today            Due today
  due:tomorrow         Due tomorrow
  due:none             No due date

COMBINING FILTERS:
  Use spaces to combine multiple filters
  All filters are combined with AND logic
  Example: status:pending priority:high @backend -tag:wontfix

EXAMPLES:
  taskflow list --query "status:pending @frontend"
    → Show pending tasks in frontend project

  taskflow list --query "priority:high due:+7d"
    → Show high priority tasks due in next 7 days

  taskflow list --query "@~back tag:bug -status:completed"
    → Show bug tasks in projects matching "back", excluding completed

  taskflow list --query "status:pending -tag:blocked -tag:waiting"
    → Show pending tasks excluding blocked and waiting tags

  taskflow list --query "due:-7d status:pending"
    → Show overdue pending tasks (due in last 7 days)

TIPS:
  - Filters are case-insensitive
  - Use @~ for typo-tolerant project matching
  - Combine multiple negations to exclude multiple values
  - Date filters use relative (+7d) or absolute (2025-01-15) formats
`
	fmt.Println(help)
}
