package jira

import (
	"errors"
	"fmt"
	"net/http"

	jira "github.com/andygrunwald/go-jira"
)

// IssueApprovals represents a Jira Issue Approvals
type IssueApprovals struct {
	Development   bool
	Product       bool
	Quality       bool
	Experience    bool
	Documentation bool
}

// Issue represents a Jira Issue
type Issue struct {
	jira.Issue
	Link         string
	LinkedIssues IssueCollection
	StoryPoints  int
	Approvals    IssueApprovals
	QAContact    string
	Acceptance   string
	Owner        string
	Impediment   bool
}

// IssueCollection is a collection of Jira Issues
type IssueCollection []*Issue

const (
	// DeliveryOwnerRegExp is the Regular Expression used to collect the Epic Delivery Owner
	DeliveryOwnerRegExp = `\W*(Delivery Owner|DELIVERY OWNER)\W*:\W*\[~([a-zA-Z0-9]*)\]`
)

var (
	// ErrorAuthentication is returned when the authentication failed
	ErrorAuthentication = errors.New("Access Unauthorized: check basic authentication")
)

// IssueStatus represent an Issue Status
type IssueStatus string

const (
	// IssueStatusObsolete represents the Issue Status Done
	IssueStatusObsolete IssueStatus = "Obsolete"
)

// IssueResolution represent an Issue Resolution
type IssueResolution string

const (
	// IssueResolutionDone represents the Issue Resolution Done
	IssueResolutionDone IssueResolution = "Done"
)

func jiraReturnError(ret *jira.Response, err error) error {
	if err == nil {
		return nil
	}

	if ret.Response.StatusCode == http.StatusForbidden || ret.Response.StatusCode == http.StatusUnauthorized {
		return ErrorAuthentication
	}

	return err
}

// NewIssueCollection creates and returns a new Jira Issue Collection
func NewIssueCollection(size int) IssueCollection {
	return make([]*Issue, size)
}

// Approved returns true if all approvals are true
func (a *IssueApprovals) Approved() bool {
	return a.Development == true && a.Product == true && a.Quality == true && a.Experience == true && a.Documentation == true
}

// AcksStatusString returns the Jira Issue Acks Status
func (i *Issue) AcksStatusString() string {
	if i.Approvals.Approved() {
		return "\u2713" // UTF-8 Mark
	}

	return ""
}

// FilterByComponent returns jira issues from collection that belongs to a component
func (c IssueCollection) FilterByComponent(component string) IssueCollection {
	r := NewIssueCollection(0)

	for _, i := range c {
		for _, t := range i.Fields.Components {
			if t.Name == component {
				r = append(r, i)
				break
			}
		}
	}

	return r
}

// FilterNotObsolete returns jira issues from collection that are not obsolete
func (c IssueCollection) FilterNotObsolete() IssueCollection {
	r := NewIssueCollection(0)

	for _, i := range c {
		if IssueStatus(i.Fields.Status.Name) != IssueStatusObsolete {
			r = append(r, i)
		}
	}

	return r
}

// EpicsTotalStatusString returns the Jira Issues Collection Status
func (c IssueCollection) EpicsTotalStatusString() string {
	totalIssues := len(c)
	completedIssues := 0

	for _, i := range c {
		if i.Fields.Resolution != nil && IssueResolution(i.Fields.Resolution.Name) == IssueResolutionDone {
			completedIssues++
		}
	}

	if totalIssues == 0 && completedIssues == 0 {
		return ""
	}

	return fmt.Sprintf("%d/%d", completedIssues, totalIssues)
}

// EpicsTotalPointsString returns the Jira Issues Collection Status
func (c IssueCollection) EpicsTotalPointsString() string {
	totalPoints := 0
	incompletePointsMark := ""
	completedPoints := 0

	for _, i := range c {
		if i.StoryPoints == 0 {
			incompletePointsMark = "!"
			continue
		}

		totalPoints += i.StoryPoints

		if i.Fields.Resolution != nil && IssueResolution(i.Fields.Resolution.Name) == IssueResolutionDone {
			completedPoints += i.StoryPoints
		}
	}

	if totalPoints == 0 && completedPoints == 0 {
		return ""
	}

	return fmt.Sprintf("%d/%d%s", completedPoints, totalPoints, incompletePointsMark)
}
