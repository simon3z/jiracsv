package main

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"regexp"

	jira "github.com/andygrunwald/go-jira"
)

// DeliveryOwnerRegExp is the Regular Expression used to collect the Epic Delivery Owner
var DeliveryOwnerRegExp = `\W*(Delivery Owner|DELIVERY OWNER)\W*:\W*\[~([a-zA-Z0-9]*)\]`

// JiraClient represents a Jira Client definition
type JiraClient struct {
	*jira.Client
}

// JiraIssue represents a Jira Issue
type JiraIssue struct {
	jira.Issue
	Link         string
	LinkedIssues JiraIssueCollection
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

	return &JiraClient{jiraClient}, nil
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

		if err != nil {
			if ret.Response.StatusCode == http.StatusForbidden || ret.Response.StatusCode == http.StatusUnauthorized {
				return nil, fmt.Errorf("Access Unauthorized: check basic authentication using a browser and retry")
			}
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
			issueURL := url.URL{
				Scheme: clientURL.Scheme,
				Host:   clientURL.Host,
				Path:   clientURL.Path + "browse/" + i.Key,
			}

			newIssues[len(issues)+j] = &JiraIssue{
				issuesPage[j],
				issueURL.String(),
				JiraIssueCollection{},
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

// DeliveryOwner returns the Jira Issue Delivery Owner
func (i *JiraIssue) DeliveryOwner() string {
	matches := regexp.MustCompile(DeliveryOwnerRegExp).FindStringSubmatch(i.Fields.Description)

	if len(matches) == 3 {
		return matches[2]
	}

	return i.Assignee()
}

func (i *JiraIssue) iterateUnknownMaps(f func(m map[string]interface{})) {
	for _, v := range i.Fields.Unknowns {
		if l, isSlice := v.([]interface{}); isSlice {
			for _, j := range l {
				f(j.(map[string]interface{}))
			}
		}
	}
}

// AcksStatus returns the Jira Issue Acks Status
func (i *JiraIssue) AcksStatus() string {
	acks := struct {
		DevelAck    bool `value:"devel_ack"`
		PMAck       bool `value:"pm_ack"`
		QEAck       bool `value:"qa_ack"`
		UXAck       bool `value:"ux_ack"`
		DocAck      bool `value:"doc_ack"`
		StoryPoints int  `value:""`
	}{false, false, false, false, false, 0}

	acksType := reflect.TypeOf(acks)
	acksValue := reflect.ValueOf(&acks)

	i.iterateUnknownMaps(func(m map[string]interface{}) {
		for j := 0; j < acksType.NumField(); j++ {
			if m["value"] == acksType.Field(j).Tag.Get("value") {
				acksValue.Elem().Field(j).SetBool(true)
			}
		}
	})

	if acks.DevelAck && acks.PMAck && acks.QEAck && acks.UXAck && acks.DocAck {
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
