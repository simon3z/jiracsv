package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"

	jira "github.com/andygrunwald/go-jira"
)

var commandFlags = struct {
	JiraURL           string
	Username          string
	Password          string
	Project           string
	IncludeVersions   ArrayFlag
	ExcludeVersions   ArrayFlag
	ExcludeComponents ArrayFlag
	JiraClient        *jira.Client
}{}

func init() {
	flag.StringVar(&commandFlags.JiraURL, "h", "", "Jira instance URL")
	flag.StringVar(&commandFlags.Username, "u", "", "Jira username")
	flag.StringVar(&commandFlags.Project, "p", "", "Project to use to collect Issues")
	flag.Var(&commandFlags.IncludeVersions, "v", "Versions to include for the Issues collection")
	flag.Var(&commandFlags.ExcludeVersions, "V", "Versions to exclude for the Issues collection")
	flag.Var(&commandFlags.ExcludeComponents, "C", "Versions to exclude for the Issues collection")
}

func writeIssues(w *csv.Writer, component string, issues []*JiraIssue) {
	for _, i := range issues {
		stories := i.LinkedEpics.FilterNotObsolete()

		if component != "" {
			stories = stories.FilterByComponent(component)
		}

		w.Write([]string{
			googleSheetLink(i.Link, i.Key),
			i.Fields.Summary,
			i.Fields.Type.Name,
			i.Fields.Priority.Name,
			i.Fields.Status.Name,
			i.DeliveryOwner(),
			i.Assignee(),
			i.AcksStatusString(),
			stories.EpicsTotalStatusString(),
			stories.EpicsTotalPointsString(),
		})
	}
}

func main() {
	flag.Parse()

	commandFlags.Password = GetPassword("PASSWORD", true)

	jiraClient, err := NewJiraClient(commandFlags.JiraURL, &commandFlags.Username, &commandFlags.Password)

	if err != nil {
		panic(err)
	}

	w := csv.NewWriter(os.Stdout)
	w.Comma = '\t'

	componentIssues := map[string][]*JiraIssue{}
	orphanIssues := []*JiraIssue{}

	components, err := jiraClient.FindProjectComponents(commandFlags.Project)

	if err != nil {
		panic(err)
	}

	for _, c := range components {
		componentIssues[c.Name] = []*JiraIssue{}
	}

	epicsJql := jiraJQLEpicsSearch(commandFlags.Project, commandFlags.IncludeVersions, commandFlags.ExcludeVersions)

	fmt.Fprintf(os.Stdout, "JQL = %s\n", epicsJql)

	issues, err := jiraClient.FindEpics(epicsJql)

	if err != nil {
		panic(err)
	}

	for _, i := range issues {
		if len(i.Fields.Components) > 0 {
			for _, c := range i.Fields.Components {
				componentIssues[c.Name] = append(componentIssues[c.Name], i)
			}
		} else {
			orphanIssues = append(orphanIssues, i)
		}
	}

	for _, k := range sortedIssuesMapKeys(componentIssues) {
		skipComponent := false

		for _, c := range commandFlags.ExcludeComponents {
			if k == c {
				skipComponent = true
				break
			}
		}

		if skipComponent {
			continue
		}

		w.Write([]string{k})
		writeIssues(w, k, componentIssues[k])

		w.Flush()
	}

	w.Write([]string{"[UNASSIGNED]"})
	writeIssues(w, "", orphanIssues)

	w.Flush()
}
