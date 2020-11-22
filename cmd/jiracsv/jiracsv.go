package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"

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
		stories := i.LinkedIssues.FilterByFunction(func(i *jira.Issue) bool {
			if i.Fields.Status != nil && jira.IssueStatus(i.Fields.Status.Name) == jira.IssueStatusObsolete {
				return false
			}
			return true
		})

		if component != nil {
			stories = stories.FilterByFunction(func(i *jira.Issue) bool {
				if i.HasComponent(*component) {
					return true
				}
				return false
			})
		}

		storiesProgress := stories.Progress()
		storyPointsProgress := stories.StoryPointsProgress()

		w.Write([]string{
			googleSheetLink(i.Link, i.Key),
			i.Fields.Summary,
			googleSheetLink(jiraIssueMarketProblemLink(i)),
			i.Fields.Priority.Name,
			i.Fields.Status.Name,
			i.Owner,
			i.QEAssignee,
			googleSheetBallot(i.Ready()),
			googleSheetProgressBar(storiesProgress.Status, storiesProgress.Total),
			googleSheetStoryPointsBar(storyPointsProgress.Status, storyPointsProgress.Total, storyPointsProgress.Unknown == 0),
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
	log.Printf("JQL returned issues: %d", len(issues))

	if err != nil {
		panic(err)
	}

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

		w.Write([]string{k.Name})
		writeIssues(w, &k.Name, k.Issues)

		w.Flush()
	}

	w.Write([]string{"[UNASSIGNED]"})
	writeIssues(w, nil, componentIssues.Orphans)

	w.Flush()
}
