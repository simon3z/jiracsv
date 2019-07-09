package analysis

import (
	"time"

	"github.com/simon3z/jiracsv/jira"
)

// IssueAnalysis represents the assessment for an epic
type IssueAnalysis struct {
	Issue            *jira.Issue
	Component        *string
	IssuesCompletion jira.Progress
	PointsCompletion jira.Progress
	NumActivities    int
	IssueNoComponent bool
	CommentStatus    CheckResultStatus
	CommentDate      *time.Time
}

// NewIssueAnalysis returns an newòy initialized IssueAnalysis
func NewIssueAnalysis(issue *jira.Issue, component *string) *IssueAnalysis {
	analysis := &IssueAnalysis{
		Issue:     issue,
		Component: component,
	}

	analysis.analyzeIssueNoComponent()
	analysis.analyzeIssuesCompletion()
	analysis.analyzeNumActivities()
	analysis.analyzeStoryPoints()
	analysis.analyzeCommentStatus()

	return analysis
}

func (a *IssueAnalysis) allLinkedIssues() jira.IssueCollection {
	return a.Issue.LinkedIssues.FilterByFunction(
		func(i *jira.Issue) bool {
			return !i.InStatus(jira.IssueStatusObsolete)
		},
	)
}

func (a *IssueAnalysis) componentLinkedIssues() jira.IssueCollection {
	if a.Component != nil && *a.Component != "" {
		return a.allLinkedIssues().FilterByFunction(
			func(i *jira.Issue) bool {
				return i.HasComponent(*a.Component)
			},
		)
	}

	return a.allLinkedIssues()
}

func (a *IssueAnalysis) analyzeIssueNoComponent() {
	issueNoComponent := false

	for _, i := range a.allLinkedIssues() {
		if len(i.Fields.Components) > 0 {
			continue
		}

		if i.Fields.Project.Key != a.Issue.Fields.Project.Key {
			continue
		}

		issueNoComponent = true
		break
	}

	a.IssueNoComponent = issueNoComponent
}

func (a *IssueAnalysis) analyzeIssuesCompletion() {
	a.IssuesCompletion = a.componentLinkedIssues().Progress()
}

func (a *IssueAnalysis) analyzeNumActivities() {
	activities := a.componentLinkedIssues().FilterByFunction(
		func(i *jira.Issue) bool {
			return i.IsType(jira.IssueTypeStory) || i.IsType(jira.IssueTypeTask) || i.IsType(jira.IssueTypeBug)
		},
	)

	a.NumActivities = len(activities)
}

func (a *IssueAnalysis) analyzeStoryPoints() {
	a.PointsCompletion = a.componentLinkedIssues().StoryPointsProgress()
}

// Analyze analyzes a Jira Epic
func (a *IssueAnalysis) analyzeCommentStatus() {
	for _, i := range append(a.componentLinkedIssues(), a.Issue) {
		if commentStatus, commentDate := getIssueCommentStatus(i); commentStatus > a.CommentStatus {
			a.CommentStatus = commentStatus
			a.CommentDate = commentDate
		}
	}
}
