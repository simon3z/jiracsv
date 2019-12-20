package jira

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"time"

	jira "github.com/andygrunwald/go-jira"
)

// Client represents a Jira Client definition
type Client struct {
	*jira.Client
	CustomFieldID struct {
		ParentLink  string
		EpicLink    string
		StoryPoints string
		AckFlags    string
		QAContact   string
		Acceptance  string
		Flagged     string
	}
}

const (
	// JiraTimeLayout represents the layout used to parse the Jira time
	JiraTimeLayout = "2006-01-02T15:04:05.999-0700"

	// NoStoryPoints is a special value used when no story points were set
	NoStoryPoints int = -1
)

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
		case "Parent Link":
			client.CustomFieldID.ParentLink = f.ID
		case "Epic Link":
			client.CustomFieldID.EpicLink = f.ID
		case "Story Points":
			client.CustomFieldID.StoryPoints = f.ID
		case "5-Acks Check":
			client.CustomFieldID.AckFlags = f.ID
		case "QA Contact":
			client.CustomFieldID.QAContact = f.ID
		case "Acceptance Criteria":
			client.CustomFieldID.Acceptance = f.ID
		case "Flagged":
			client.CustomFieldID.Flagged = f.ID
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
		issuesPage, ret, err := c.Issue.Search(jql, &jira.SearchOptions{
			StartAt:       len(issues),
			MaxResults:    50,
			ValidateQuery: "strict",
			Fields:        []string{"*all"},
		})

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
			storyPoints := NoStoryPoints

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

			parentLink := ""

			if val := i.Fields.Unknowns[c.CustomFieldID.ParentLink]; val != nil {
				parentLink = val.(string)
			}

			if val := i.Fields.Unknowns[c.CustomFieldID.EpicLink]; i.Fields.Epic == nil && val != nil {
				i.Fields.Epic = &jira.Epic{Key: val.(string)}
			}

			qaContact := ""

			if val := i.Fields.Unknowns[c.CustomFieldID.QAContact]; val != nil {
				qaContact = (val.(map[string]interface{})["key"]).(string)
			}

			acceptanceCriteria := ""

			if val := i.Fields.Unknowns[c.CustomFieldID.Acceptance]; val != nil {
				acceptanceCriteria = val.(string)
			}

			deliveryOwner := ""
			deliveryOwnerMatches := regexp.MustCompile(DeliveryOwnerRegExp).FindStringSubmatch(i.Fields.Description)

			if len(deliveryOwnerMatches) == 3 {
				deliveryOwner = deliveryOwnerMatches[2]
			} else if i.Fields.Assignee != nil {
				deliveryOwner = i.Fields.Assignee.Name
			}

			impediment := false

			if val := i.Fields.Unknowns[c.CustomFieldID.Flagged]; val != nil {
				for _, f := range val.([]interface{}) {
					switch f.(map[string]interface{})["value"].(string) {
					case "Impediment":
						impediment = true
					}
				}
			}

			issueURL := url.URL{
				Scheme: clientURL.Scheme,
				Host:   clientURL.Host,
				Path:   clientURL.Path + "browse/" + i.Key,
			}

			issueComments := []*Comment{}

			for _, c := range i.Fields.Comments.Comments {
				commentCreateTime, err := time.Parse(JiraTimeLayout, c.Created)

				if err != nil {
					return nil, err
				}

				commentUpdateTime, err := time.Parse(JiraTimeLayout, c.Updated)

				if err != nil {
					return nil, err
				}

				issueComments = append(issueComments, &Comment{
					Comment: c,
					Created: commentCreateTime,
					Updated: commentUpdateTime,
				})
			}

			newIssues[len(issues)+j] = &Issue{
				i,
				issueURL.String(),
				parentLink,
				NewIssueCollection(0),
				storyPoints,
				issueApprovals,
				qaContact,
				acceptanceCriteria,
				deliveryOwner,
				impediment,
				issueComments,
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
			epics, err := c.FindIssues(fmt.Sprintf("issueFunction in issuesInEpics(\"Key = %s\") ORDER BY Key ASC", i.Key))

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
