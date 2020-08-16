package main

import (
	"github.com/simon3z/jiracsv/jira"
)

// ComponentIssues contain issues of the relevant component
type ComponentIssues struct {
	Name   string
	Issues []*jira.Issue
}

// ComponentsCollection is a collection of ordered and unique ComponentIssues
type ComponentsCollection struct {
	Items   []*ComponentIssues
	Orphans []*jira.Issue
	index   map[string]*ComponentIssues
}

// NewComponentsCollection returns a new ComponentsCollection
func NewComponentsCollection() *ComponentsCollection {
	return &ComponentsCollection{
		index: map[string]*ComponentIssues{},
	}
}

// Add initializes the relevant component if needed and optionally adds issues
func (c *ComponentsCollection) Add(component string, issue ...*jira.Issue) {
	item, ok := c.index[component]

	if !ok {
		item = &ComponentIssues{component, []*jira.Issue{}}

		c.Items = append(c.Items, item)
		c.index[component] = item
	}

	for _, i := range issue {
		c.index[component].Issues = append(c.index[component].Issues, i)
	}
}

// AddIssues adds all the issues by component
func (c *ComponentsCollection) AddIssues(issues []*jira.Issue) {
	for _, i := range issues {
		components := map[string]bool{}

		for _, c := range i.Fields.Components {
			components[c.Name] = true
		}

		for _, j := range i.LinkedIssues {
			if j.InStatus(jira.IssueStatusObsolete) {
				continue
			}

			for _, c := range j.Fields.Components {
				components[c.Name] = true
			}
		}

		if len(components) > 0 {
			for k := range components {
				c.Add(k, i)
			}
		} else {
			c.Orphans = append(c.Orphans, i)
		}
	}
}
