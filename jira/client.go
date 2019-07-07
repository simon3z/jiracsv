package jira

import (
	"fmt"
	"net/http"
	"net/url"

	jira "github.com/andygrunwald/go-jira"
)

// Client represents a Jira Client definition
type Client struct {
	*jira.Client
	CustomFieldID struct {
		StoryPoints string
		AckFlags    string
	}
}

// NewClient creates and returns a new Jira Client
func NewClient(url string, username, password *string) (*Client, error) {
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

	client := &Client{Client: jiraClient}

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

// FindProjectComponents finds all the components in the specified project
func (c *Client) FindProjectComponents(project string) ([]jira.ProjectComponent, error) {
	p, _, err := c.Project.Get(project)

	if err != nil {
		return nil, err
	}

	return p.Components, nil
}

// FindIssues finds all the Jira Issues returned by the JQL search
func (c *Client) FindIssues(jql string) (IssueCollection, error) {
	issues := NewIssueCollection(0)

	for {
		issuesPage, ret, err := c.Issue.Search(jql, &jira.SearchOptions{StartAt: len(issues), MaxResults: 50})

		if err := jiraReturnError(ret, err); err != nil {
			return nil, err
		}

		if len(issuesPage) == 0 {
			break
		}

		newIssues := NewIssueCollection(len(issues) + len(issuesPage))

		if copy(newIssues, issues) != len(issues) {
			return nil, fmt.Errorf("cannot copy issues") // TODO
		}

		clientURL := c.GetBaseURL()

		for j, i := range issuesPage {
			storyPoints := 0

			if val := i.Fields.Unknowns[c.CustomFieldID.StoryPoints]; val != nil {
				storyPoints = int(val.(float64))
			}

			issueApprovals := IssueApprovals{false, false, false, false, false}

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

			newIssues[len(issues)+j] = &Issue{
				i,
				issueURL.String(),
				NewIssueCollection(0),
				storyPoints,
				issueApprovals,
			}
		}

		issues = newIssues
	}

	return issues, nil
}

// FindEpics finds all the Jira Epics returned by the JQL search
func (c *Client) FindEpics(jql string) (IssueCollection, error) {
	issues, err := c.FindIssues(jql)

	if err != nil {
		return nil, err
	}

	ch := make(chan error)

	for _, i := range issues {
		go func(i *Issue, ch chan<- error) {
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
