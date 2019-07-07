package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"

	jira "github.com/andygrunwald/go-jira"
)

const (
	// DeliveryOwnerRegExp is the Regular Expression used to collect the Epic Delivery Owner
	DeliveryOwnerRegExp = `\W*(Delivery Owner|DELIVERY OWNER)\W*:\W*\[~([a-zA-Z0-9]*)\]`
)

var (
	// ErrorAuthentication is returned when the authentication failed
	ErrorAuthentication = errors.New("Access Unauthorized: check basic authentication")
)

// JiraClient represents a Jira Client definition
type JiraClient struct {
	*jira.Client
	CustomFieldID struct {
		StoryPoints string
		AckFlags    string
	}
}

// JiraIssueApprovals represents a Jira Issue Approvals
type JiraIssueApprovals struct {
	Development   bool
	Product       bool
	Quality       bool
	Experience    bool
	Documentation bool
}

// JiraIssue represents a Jira Issue
type JiraIssue struct {
	jira.Issue
	Link         string
	LinkedIssues JiraIssueCollection
	StoryPoints  int
	Approvals    JiraIssueApprovals
}

// JiraIssueCollection is a collection of Jira Issues
type JiraIssueCollection []*JiraIssue

// NewJiraClient creates and returns a new Jira Client
func NewJiraClient(url string, username, password *string) (*JiraClient, error) {
	var httpClient *http.Client

	if username != nil && *username != "" {
		transport := jira.BasicAuthTransport{Username: *username, Password: *password}
		httpClient = transport.Client()
	}

	jiraClient, err := jira.NewClient(httpClient, url)

	if err != nil {
		return nil, err
	}

	fields, ret, err := jiraClient.Field.GetList()

	if err := jiraReturnError(ret, err); err != nil {
		return nil, err
	}

	client := &JiraClient{Client: jiraClient}

	for _, f := range fields {
		switch f.Name {
		case "Story Points":
			client.CustomFieldID.StoryPoints = f.ID
		case "5-Acks Check":
			client.CustomFieldID.AckFlags = f.ID
		}
	}

	return client, nil
}

func jiraReturnError(ret *jira.Response, err error) error {
	if err == nil {
		return nil
	}

	if ret.Response.StatusCode == http.StatusForbidden || ret.Response.StatusCode == http.StatusUnauthorized {
		return ErrorAuthentication
	}

	return err
}

// NewJiraIssueCollection creates and returns a new Jira Issue Collection
func NewJiraIssueCollection(size int) JiraIssueCollection {
	return make([]*JiraIssue, size)
}

// FindProjectComponents finds all the components in the specified project
func (c *JiraClient) FindProjectComponents(name string) ([]jira.ProjectComponent, error) {
	project, _, err := c.Project.Get(name)

	if err != nil {
		return nil, err
	}

	return project.Components, nil
}

// FindIssues finds all the Jira Issues returned by the JQL search
func (c *JiraClient) FindIssues(jql string) (JiraIssueCollection, error) {
	issues := JiraIssueCollection{}

	for {
		issuesPage, ret, err := c.Issue.Search(jql, &jira.SearchOptions{StartAt: len(issues), MaxResults: 50})

		if err := jiraReturnError(ret, err); err != nil {
			return nil, err
		}

		if len(issuesPage) == 0 {
			break
		}

		newIssues := NewJiraIssueCollection(len(issues) + len(issuesPage))

		if copy(newIssues, issues) != len(issues) {
			return nil, fmt.Errorf("cannot copy issues") // TODO
		}

		clientURL := c.GetBaseURL()

		for j, i := range issuesPage {
			storyPoints := 0

			if val := i.Fields.Unknowns[c.CustomFieldID.StoryPoints]; val != nil {
				storyPoints = int(val.(float64))
			}

			issueApprovals := JiraIssueApprovals{false, false, false, false, false}

			if val := i.Fields.Unknowns[c.CustomFieldID.AckFlags]; val != nil {
				for _, p := range val.([]interface{}) {
					switch p.(map[string]interface{})["value"].(string) {
					case "devel_ack":
						issueApprovals.Development = true
					case "pm_ack":
						issueApprovals.Product = true
					case "qa_ack":
						issueApprovals.Quality = true
					case "ux_ack":
						issueApprovals.Experience = true
					case "doc_ack":
						issueApprovals.Documentation = true
					}
				}
			}

			issueURL := url.URL{
				Scheme: clientURL.Scheme,
				Host:   clientURL.Host,
				Path:   clientURL.Path + "browse/" + i.Key,
			}

			newIssues[len(issues)+j] = &JiraIssue{
				i,
				issueURL.String(),
				JiraIssueCollection{},
				storyPoints,
				issueApprovals,
			}
		}

		issues = newIssues
	}

	return issues, nil
}

// FindEpics finds all the Jira Epics returned by the JQL search
func (c *JiraClient) FindEpics(jql string) (JiraIssueCollection, error) {
	issues, err := c.FindIssues(jql)

	if err != nil {
		return nil, err
	}

	ch := make(chan error)

	for _, i := range issues {
		go func(i *JiraIssue, ch chan<- error) {
			epics, err := c.FindIssues(fmt.Sprintf("\"Epic Link\" = \"%s\"", i.Key))

			if err == nil {
				i.LinkedIssues = epics
			}

			ch <- err
		}(i, ch)
	}

	for range issues {
		if err := <-ch; err != nil {
			return nil, err
		}
	}

	return issues, nil
}

// Approved returns true if all approvals are true
func (a *JiraIssueApprovals) Approved() bool {
	return a.Development == true && a.Product == true && a.Quality == true && a.Experience == true && a.Documentation == true
}

// DeliveryOwner returns the Jira Issue Delivery Owner
func (i *JiraIssue) DeliveryOwner() string {
	matches := regexp.MustCompile(DeliveryOwnerRegExp).FindStringSubmatch(i.Fields.Description)

	if len(matches) == 3 {
		return matches[2]
	}

	return i.Assignee()
}

// AcksStatusString returns the Jira Issue Acks Status
func (i *JiraIssue) AcksStatusString() string {
	if i.Approvals.Approved() {
		return "\u2713" // UTF-8 Mark
	}

	return ""
}

// Assignee returns the Jira Issue Assignee
func (i *JiraIssue) Assignee() string {
	if i.Fields.Assignee == nil {
		return ""
	}

	return i.Fields.Assignee.Key
}

// EpicsTotalStatusString returns the Jira Issues Collection Status
func (c JiraIssueCollection) EpicsTotalStatusString() string {
	totalIssues := len(c)
	completedIssues := 0

	for _, i := range c {
		if i.Fields.Resolution != nil && i.Fields.Resolution.Name == "Done" {
			completedIssues++
		}
	}

	if totalIssues == 0 && completedIssues == 0 {
		return ""
	}

	return fmt.Sprintf("%d/%d", completedIssues, totalIssues)
}

// EpicsTotalPointsString returns the Jira Issues Collection Status
func (c JiraIssueCollection) EpicsTotalPointsString() string {
	totalPoints := 0
	incompletePointsMark := ""
	completedPoints := 0

	for _, i := range c {
		if i.StoryPoints == 0 {
			incompletePointsMark = "!"
			continue
		}

		totalPoints += i.StoryPoints

		if i.Fields.Resolution != nil && i.Fields.Resolution.Name == "Done" {
			completedPoints += i.StoryPoints
		}
	}

	if totalPoints == 0 && completedPoints == 0 {
		return ""
	}

	return fmt.Sprintf("%d/%d%s", completedPoints, totalPoints, incompletePointsMark)
}
