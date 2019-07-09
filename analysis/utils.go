package analysis

import (
	"strings"
	"time"

	"github.com/simon3z/jiracsv/jira"
)

func getIssueCommentStatus(issue *jira.Issue) (CheckResultStatus, *time.Time) {
	if issue.Fields.Comments == nil {
		return CheckStatusNone, nil
	}

	for j := len(issue.Comments) - 1; j >= 0; j-- {
		if strings.HasPrefix(issue.Comments[j].Body, "GREEN:") {
			return CheckStatusGreen, &issue.Comments[j].Updated
		}

		if strings.HasPrefix(issue.Comments[j].Body, "YELLOW:") {
			return CheckStatusYellow, &issue.Comments[j].Updated
		}

		if strings.HasPrefix(issue.Comments[j].Body, "RED:") {
			return CheckStatusRed, &issue.Comments[j].Updated
		}
	}

	return CheckStatusNone, nil
}
