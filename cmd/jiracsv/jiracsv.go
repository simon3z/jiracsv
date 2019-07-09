package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/simon3z/jiracsv/analysis"
	"github.com/simon3z/jiracsv/jira"
)

var commandFlags = struct {
	Configuration string
	Profile       string
	Username      string
}{}

func init() {
	flag.StringVar(&commandFlags.Username, "u", "", "Jira username")
	flag.StringVar(&commandFlags.Configuration, "c", "", "Configuration file")
	flag.StringVar(&commandFlags.Profile, "p", "", "Search profile")
}

func writeIssues(w *csv.Writer, component *string, issues []*jira.Issue) {
	for _, i := range issues {
		a := analysis.NewIssueAnalysis(i, component)
		r := analysis.NewCheckResult(a)

		w.Write([]string{
			googleSheetLink(i.Link, i.Key),
			i.Fields.Summary,
			googleSheetLink(jiraIssueMarketProblemLink(i)),
			i.Fields.Priority.Name,
			i.Fields.Status.Name,
			i.Owner,
			i.QEAssignee,
			googleSheetProgressBar(a.IssuesCompletion.Status, a.IssuesCompletion.Total),
			googleSheetStoryPointsBar(a.PointsCompletion.Status, a.PointsCompletion.Total, a.PointsCompletion.Unknown == 0),
			googleSheetTime(a.CommentDate),
			googleSheetBallot(r.Ready),
			googleSheetCheckStatus(r.Status),
			googleSheetSortedMessages(r.Messages, ","),
		})
	}
}

func main() {
	flag.Parse()

	if commandFlags.Configuration == "" {
		panic("configuration file not specified")
	}

	if commandFlags.Profile == "" {
		panic("profile id file not specified")
	}

	if commandFlags.Username == "" {
		panic("jira username not specified")
	}

	config, err := ReadConfigFile(commandFlags.Configuration)

	if err != nil {
		panic(err)
	}

	profile := config.FindProfile(commandFlags.Profile)

	if profile == nil {
		panic(fmt.Errorf("profile '%s' not found", commandFlags.Profile))
	}

	password, err := GetPassword("PASSWORD", true)

	if err != nil {
		panic(err)
	}

	jiraClient, err := jira.NewClient(config.Instance.URL, &commandFlags.Username, &password)

	if err != nil {
		panic(err)
	}

	w := csv.NewWriter(os.Stdout)
	w.Comma = '\t'

	componentIssues := NewComponentsCollection()

	for _, c := range profile.Components.Include {
		componentIssues.Add(c)
	}

	log.Printf("JQL = %s\n", profile.JQL)

	issues, err := jiraClient.FindEpics(profile.JQL)

	if err != nil {
		panic(err)
	}

	log.Printf("JQL returned issues: %d", len(issues))

	componentIssues.AddIssues(issues)

	for _, k := range componentIssues.Items {
		skipComponent := false

		for _, c := range profile.Components.Exclude {
			if k.Name == c {
				skipComponent = true
				break
			}
		}

		if skipComponent {
			continue
		}

		w.Write(append([]string{k.Name}, make([]string, 12)...))
		writeIssues(w, &k.Name, k.Issues)

		w.Flush()
	}

	w.Write([]string{"[UNASSIGNED]"})
	writeIssues(w, nil, componentIssues.Orphans)

	w.Flush()
}
