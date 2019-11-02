package analysis

import (
	"strings"
	"time"

	"io.bytenix.com/jiracsv/jira"
)

// IssueAnalysis represents the assessment for an epic
type IssueAnalysis struct {
	Issue        *jira.Issue
	Component    *string
	LinkedIssues struct {
		Total     int
		Completed int
	}
	StoryPoints struct {
		Total     int
		Completed int
		Complete  bool
	}
	NoActivities     bool
	Impediment       bool
	MixedTypes       bool
	IssueNoComponent bool
	CommentStatus    CheckResultStatus
	CommentDate      *time.Time
}

// AnalyzeIssue analyzes a Jira Epic
func AnalyzeIssue(issue *jira.Issue, component *string) *IssueAnalysis {
	assessment := &IssueAnalysis{
		Issue:            issue,
		Component:        component,
		Impediment:       false,
		MixedTypes:       false,
		IssueNoComponent: false,
		CommentStatus:    CheckStatusNone,
	}

	allLinkedIssues := issue.LinkedIssues.FilterByFunction(
		func(i *jira.Issue) bool {
			return !i.InStatus(jira.IssueStatusObsolete)
		},
	)

	var linkedIssues jira.IssueCollection

	if component != nil && *component != "" {
		linkedIssues = allLinkedIssues.FilterByFunction(
			func(i *jira.Issue) bool {
				return i.HasComponent(*component) && !i.IsType(jira.IssueTypeEpic)

			},
		)
	} else {
		linkedIssues = allLinkedIssues
	}

	linkedIssuesDone := linkedIssues.FilterByFunction(
		func(i *jira.Issue) bool {
			return i.InStatus(jira.IssueStatusDone)
		},
	)

	linkedActivities := linkedIssues.FilterByFunction(
		func(i *jira.Issue) bool {
			return i.IsType(jira.IssueTypeStory) || i.IsType(jira.IssueTypeTask)
		},
	)

	linkedActivitiesDone := linkedActivities.FilterByFunction(
		func(i *jira.Issue) bool {
			return i.InStatus(jira.IssueStatusDone)
		},
	)

	assessment.NoActivities = linkedActivities.Len() == 0

	assessment.LinkedIssues.Total = linkedIssues.Len()
	assessment.LinkedIssues.Completed = linkedIssuesDone.Len()

	assessment.StoryPoints.Total = linkedActivities.StoryPoints()
	assessment.StoryPoints.Completed = linkedActivitiesDone.StoryPoints()

	assessment.StoryPoints.Complete = true

	for _, i := range allLinkedIssues {
		if len(i.Fields.Components) == 0 {
			assessment.IssueNoComponent = true
		}
	}

	for _, i := range linkedIssues {
		if i.IsType(jira.IssueTypeStory) {
			if !i.HasStoryPoints() {
				assessment.StoryPoints.Complete = false
			}
		} else {
			assessment.MixedTypes = true
		}

		if i.Impediment {
			assessment.Impediment = true
		}
	}

	for _, i := range append(linkedIssues, assessment.Issue) {
		if commentStatus, commentDate := getIssueCommentStatus(i); commentStatus > assessment.CommentStatus {
			assessment.CommentStatus = commentStatus
			assessment.CommentDate = commentDate
		}
	}

	return assessment
}

// CheckStatus executes the checks for a specific ReleasePhase
func (a *IssueAnalysis) CheckStatus() *CheckResult {
	result := NewCheckResult(true, CheckStatusNone)

	if a.NoActivities {
		result.SetReady(false).AddMessage("NOSTORIES")
	}

	if a.Issue.Fields.Description == "" {
		result.SetReady(false).AddMessage("NODESCRIPTION")
	}

	if !a.Issue.Approvals.Approved() {
		result.SetReady(false).AddMessage("NOACKS")
	}

	if a.Issue.Owner == "" {
		result.SetReady(false).SetStatus(CheckStatusRed).AddMessage("NODELIVERYOWNER")
	}

	if a.Issue.QAContact == "" {
		result.SetReady(false).SetStatus(CheckStatusRed).AddMessage("NOQACONTACT")
	}

	if a.Issue.Acceptance == "" {
		result.SetReady(false).SetStatus(CheckStatusRed).AddMessage("NOCRITERIA")
	}

	if !a.Issue.IsPrioritized() {
		result.SetReady(false).SetStatus(CheckStatusRed).AddMessage("NOPRIORITY")
	}

	if !a.Issue.IsActive() && !a.Issue.InStatus(jira.IssueStatusDone) {
		result.SetStatus(CheckStatusYellow).AddMessage("NOTSTARTED")
	}

	if !a.StoryPoints.Complete {
		result.AddMessage("NOSTORYPOINTS")
	}

	if a.Impediment {
		result.SetStatus(CheckStatusRed).AddMessage("IMPEDIMENT")
	}

	if a.Issue.ParentLink == "" {
		result.SetReady(false).AddMessage("NOINITIATIVE")
	}

	if a.IssueNoComponent {
		result.SetReady(false).AddMessage("ISSUENOCOMPONENT")
	}

	for _, label := range a.Issue.Issue.Fields.Labels {
		if label == "grooming" {
			result.SetReady(false).AddMessage("GROOMING")
		}
	}

	/*
		for _, i := range linkedIssues {
			if i.Owner == "" && i.Fields.Sprint.Name != "" {
				result.AppendCheckResult(CheckStatusNone, CheckStatusYellow, "ISSUENOASSIGNEE")
			}
		}
	*/

	if a.Component != nil {
		missing := true

		for _, c := range a.Issue.Fields.Components {
			if c.Name == *a.Component {
				missing = false
				break
			}
		}

		if missing {
			result.SetReady(false).SetStatus(CheckStatusYellow).AddMessage("NOCOMPONENT")
		}
	}

	if a.Issue.InStatus(jira.IssueStatusDone) {
		if a.LinkedIssues.Completed != a.LinkedIssues.Total ||
			a.StoryPoints.Completed != a.StoryPoints.Total {
			result.SetStatus(CheckStatusRed).AddMessage("NOTDONE")
		} else {
			result.SetStatus(CheckStatusGreen)
		}
	}

	if a.CommentStatus > CheckStatusNone {
		result.SetStatus(a.CommentStatus).AddMessage("STATUSCOMMENT")
	}

	return result
}

func getIssueCommentStatus(issue *jira.Issue) (CheckResultStatus, *time.Time) {
	if issue.Fields.Comments == nil {
		return CheckStatusNone, nil
	}

	for j := len(issue.Comments) - 1; j >= 0; j-- {
		if strings.HasPrefix(issue.Comments[j].Body, "GREEN: ") {
			return CheckStatusGreen, &issue.Comments[j].Updated
		}

		if strings.HasPrefix(issue.Comments[j].Body, "YELLOW: ") {
			return CheckStatusYellow, &issue.Comments[j].Updated
		}

		if strings.HasPrefix(issue.Comments[j].Body, "RED: ") {
			return CheckStatusRed, &issue.Comments[j].Updated
		}
	}

	return CheckStatusNone, nil
}
