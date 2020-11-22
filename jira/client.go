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
		QEAssignee  string
		Acceptance  string
		Flagged     string
		Planning    string
		Readiness   string
		Design      string
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
		case "QE Assignee":
			client.CustomFieldID.QEAssignee = f.ID
		case "Acceptance Criteria":
			client.CustomFieldID.Acceptance = f.ID
		case "Flagged":
			client.CustomFieldID.Flagged = f.ID
		case "OpenShift Planning":
			client.CustomFieldID.Planning = f.ID
		case "Ready-Ready":
			client.CustomFieldID.Readiness = f.ID
		case "Design Doc":
			client.CustomFieldID.Design = f.ID
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

			issueReadiness := IssueReadiness{false, false, false, false, false, false}

			if i.Fields.FixVersions != nil && len(i.Fields.FixVersions) > 0 {
				issueReadiness.Development = true
				issueReadiness.Product = true
			}

			if val := i.Fields.Unknowns[c.CustomFieldID.Readiness]; val != nil {
				for _, r := range val.([]interface{}) {
					switch r.(map[string]interface{})["value"].(string) {
					case "dev-ready":
						issueReadiness.Development = true
					case "pm-ready":
						issueReadiness.Product = true
					case "doc-ready":
						issueReadiness.Documentation = true
					case "px-ready":
						issueReadiness.Support = true
					case "qa-ready":
						issueReadiness.Quality = true
					case "ux-ready":
						issueReadiness.Experience = true
					}
				}
			}

			issuePlanning := IssuePlanning{false, false, false}

			if val := i.Fields.Unknowns[c.CustomFieldID.Planning]; val != nil {
				for _, p := range val.([]interface{}) {
					switch p.(map[string]interface{})["value"].(string) {
					case "no-feature":
						issuePlanning.NoFeature = true
					case "no-doc":
						issuePlanning.NoDoc = true
					case "no-qe":
						issuePlanning.NoQE = true
					}
				}
			}

			designLink := ""

			if val := i.Fields.Unknowns[c.CustomFieldID.Design]; val != nil {
				designLink = val.(string)
			}

			parentLink := ""

			if val := i.Fields.Unknowns[c.CustomFieldID.ParentLink]; val != nil {
				parentLink = val.(string)
			}

			if val := i.Fields.Unknowns[c.CustomFieldID.EpicLink]; i.Fields.Epic == nil && val != nil {
				i.Fields.Epic = &jira.Epic{Key: val.(string)}
			}

			qeAssignee := ""

			if val := i.Fields.Unknowns[c.CustomFieldID.QEAssignee]; val != nil {
				qeAssignee = (val.(map[string]interface{})["key"]).(string)
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
				nil,
				NewIssueCollection(0),
				storyPoints,
				issueReadiness,
				issuePlanning,
				designLink,
				qeAssignee,
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

	epics := issues.FilterByFunction(func(i *Issue) bool {
		return i.IsType(IssueTypeEpic)
	})

	ch := make(chan error)
	defer close(ch)

	for _, i := range epics {
		go func(i *Issue, ch chan<- error) { ch <- addLinkedIssues(c, i) }(i, ch)
	}

	linksErr := error(nil)

	for range epics {
		if err := <-ch; err != nil {
			linksErr = err
		}
	}

	return issues, linksErr
}

func addLinkedIssues(c *Client, i *Issue) error {
	jql := fmt.Sprintf("issueFunction in linkedIssuesOfRecursive(\"issue = %s\", \"is child of\") AND type = \"Market Problem\"", i.Key)
	marketProblem, err := c.FindIssues(jql)

	switch {
	case err != nil:
		return err
	case len(marketProblem) > 1:
		return ErrMultipleIssues
	case len(marketProblem) < 1:
		return nil
	}

	i.MarketProblem = marketProblem[0]

	jql = fmt.Sprintf("issueFunction in issuesInEpics(\"Key = %s\")", i.Key)
	linkedIssues, err := c.FindIssues(jql)

	if err != nil {
		return err
	}

	i.LinkedIssues = linkedIssues

	return nil
}
