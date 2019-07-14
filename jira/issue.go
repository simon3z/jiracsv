package jira

import (
	"errors"
	"net/http"
	"time"

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
	ParentLink   string
	LinkedIssues IssueCollection
	StoryPoints  int
	Approvals    IssueApprovals
	QAContact    string
	Acceptance   string
	Owner        string
	Impediment   bool
	Comments     []*Comment
}

// Comment represents Jira Issue Comment
type Comment struct {
	*jira.Comment
	Created time.Time
	Updated time.Time
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
	// IssueStatusDone represents the Issue Status Done
	IssueStatusDone IssueStatus = "Done"

	// IssueStatusObsolete represents the Issue Status Done
	IssueStatusObsolete IssueStatus = "Obsolete"

	// IssueStatusInProgress represents the Issue Status Done
	IssueStatusInProgress IssueStatus = "In Progress"

	// IssueStatusFeatureComplete represents the Issue Status Done
	IssueStatusFeatureComplete IssueStatus = "Feature Complete"

	// IssueStatusCodeReview represents the Issue Status Done
	IssueStatusCodeReview IssueStatus = "Code Review"

	// IssueStatusQEReview represents the Issue Status Done
	IssueStatusQEReview IssueStatus = "QE Review"
)

// IssueResolution represent an Issue Resolution
type IssueResolution string

const (
	// IssueResolutionDone represents the Issue Resolution Done
	IssueResolutionDone IssueResolution = "Done"
)

// IssuePriority represent an Issue Resolution
type IssuePriority string

const (
	// IssuePriorityUnprioritized represents the Issue Priority Unprioritized
	IssuePriorityUnprioritized IssuePriority = "Unprioritized"
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

// IsActive returns true if the issue is currently worked on
func (i *Issue) IsActive() bool {
	switch IssueStatus(i.Fields.Status.Name) {
	case IssueStatusInProgress:
		return true
	case IssueStatusFeatureComplete:
		return true
	case IssueStatusCodeReview:
		return true
	case IssueStatusQEReview:
		return true
	}

	return false
}

// IsDone returns true if the issue Status is Done
func (i *Issue) IsDone() bool {
	if i.Fields.Status != nil && IssueStatus(i.Fields.Status.Name) == IssueStatusDone {
		return true
	}

	return false
}

// IsResolved returns true if the issue Resolution is Done
func (i *Issue) IsResolved() bool {
	if i.Fields.Resolution != nil && IssueResolution(i.Fields.Resolution.Name) == IssueResolutionDone {
		return true
	}

	return false
}

// IsObsolete returns true if the issue Status is Obsolete
func (i *Issue) IsObsolete() bool {
	if i.Fields.Status != nil && IssueStatus(i.Fields.Status.Name) == IssueStatusObsolete {
		return true
	}

	return false
}

// IsPrioritized returns true if the issue Priority has been set
func (i *Issue) IsPrioritized() bool {
	if i.Fields.Priority != nil {
		switch IssuePriority(i.Fields.Priority.Name) {
		case IssuePriorityUnprioritized:
			return false
		case "":
			return false
		}
	}

	return true
}

// HasStoryPoints returns true if the issue has story points defined
func (i *Issue) HasStoryPoints() bool {
	if i.StoryPoints > NoStoryPoints {
		return true
	}

	return false
}

// FilterByFunction returns jira issues from collection that satisfy the provided function
func (c IssueCollection) FilterByFunction(fn func(*Issue) bool) IssueCollection {
	r := NewIssueCollection(0)

	for _, i := range c {
		if fn(i) {
			r = append(r, i)
		}
	}

	return r
}

// FilterByComponent returns jira issues from collection that belongs to a component
func (c IssueCollection) FilterByComponent(component string) IssueCollection {
	return c.FilterByFunction(func(i *Issue) bool {
		for _, t := range i.Fields.Components {
			if t.Name == component {
				return true
			}
		}
		return false
	})
}

// FilterNotObsolete returns jira issues from collection that are not obsolete
func (c IssueCollection) FilterNotObsolete() IssueCollection {
	return c.FilterByFunction(func(i *Issue) bool {
		return !i.IsObsolete()
	})
}

// FilterDone returns jira issues from collection that are done
func (c IssueCollection) FilterDone() IssueCollection {
	return c.FilterByFunction(func(i *Issue) bool {
		return i.IsDone()
	})
}

// Len returns the number of issues in the collection
func (c IssueCollection) Len() int {
	return len(c)
}

// StoryPoints returns the total number of story points for the issues in the collection
func (c IssueCollection) StoryPoints() int {
	points := 0

	for _, i := range c {
		if i.HasStoryPoints() {
			points += i.StoryPoints
		}
	}

	return points
}
