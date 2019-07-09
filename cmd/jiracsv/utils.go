package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/simon3z/jiracsv/analysis"
	"github.com/simon3z/jiracsv/jira"
	"golang.org/x/crypto/ssh/terminal"
)

// ArrayFlag is used for command line flags with multiple values
type ArrayFlag []string

func (i *ArrayFlag) String() string {
	return strings.Join(*i, ", ")
}

// Set function adds a value to the array
func (i *ArrayFlag) Set(value string) error {
	*i = append(*i, value)
	return nil
}

// GetPassword gets the password either form an environment variable or interactively
func GetPassword(env string, interactive bool) (string, error) {
	password := os.Getenv(env)

	if interactive && password == "" && terminal.IsTerminal(syscall.Stdin) {
		os.Stdin.Write([]byte("Password: "))

		pw, err := terminal.ReadPassword(syscall.Stdin)
		defer os.Stdin.Write([]byte("\n"))

		if err != nil {
			return "", err
		}

		password = string(pw)
	}

	return password, nil
}

func sortedIssuesMapKeys(m map[string][]*jira.Issue) []string {
	keys := make([]string, 0, len(m))

	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}

func jiraIssueMarketProblemLink(i *jira.Issue) (string, string) {
	if i.MarketProblem == nil {
		return "", ""
	}
	return i.MarketProblem.Link, i.MarketProblem.Fields.Summary
}

func googleSheetLink(link, text string) string {
	return fmt.Sprintf("=HYPERLINK(\"%s\",\"%s\")", link, text)
}

func googleSheetBallot(value bool) string {
	if value {
		return "\u2713" // UTF-8 Mark
	}

	return "\u2717"
}

func googleSheetProgressBar(value, max int) string {
	if value > max || (max == 0 && value == 0) {
		return "\u2014" // UTF-8 Dash
	}

	return fmt.Sprintf("=SPARKLINE({%d,%d},{\"charttype\",\"bar\";\"color1\",\"#93c47d\";\"color2\",\"#efefef\"})", value, max-value)
}

func googleSheetStoryPointsBar(value, max int, complete bool) string {
	if !complete {
		return "\u2014" // UTF-8 Dash
	}

	return googleSheetProgressBar(value, max)
}

func googleSheetCheckStatus(status analysis.CheckResultStatus) string {
	if status == analysis.CheckStatusNone {
		return "\u2014" // UTF-8 Dash
	}

	return status.String()
}

func googleSheetTime(time *time.Time) string {
	if time == nil {
		return "\u2014" // UTF-8 Dash
	}

	return time.Format("2006-01-02")
}

// MessagesString returns a string representing all messages
func googleSheetSortedMessages(messages []string, sep string) string {
	sortedMessages := make([]string, len(messages))

	copy(sortedMessages, messages)
	sort.Strings(sortedMessages)

	return strings.Join(sortedMessages, sep)
}
