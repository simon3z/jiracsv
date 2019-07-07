package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
	"io.bytenix.com/jiracsv/jira"
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
func GetPassword(env string, interactive bool) string {
	password := os.Getenv(env)

	if interactive && password == "" && terminal.IsTerminal(syscall.Stdin) {
		os.Stdin.Write([]byte("Password: "))

		pw, err := terminal.ReadPassword(syscall.Stdin)
		defer os.Stdin.Write([]byte("\n"))

		if err != nil {
			panic(err)
		}

		password = string(pw)
	}

	return password
}

func sortedIssuesMapKeys(m map[string][]*jira.Issue) []string {
	keys := make([]string, 0, len(m))

	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}

func googleSheetLink(link, text string) string {
	return fmt.Sprintf("=HYPERLINK(\"%s\",\"%s\")", link, text)
}

func jiraJQLEpicsSearch(project string, includeVersions, excludeVersions []string) string {
	filter := []string{
		fmt.Sprintf("status != \"Obsolete\""),
	}

	if project != "" {
		filter = append(filter, fmt.Sprintf("project = \"%s\"", project))
	}

	if len(includeVersions) > 0 {
		v := strings.Join(includeVersions, "\",\"")
		filter = append(filter, fmt.Sprintf("fixVersion IN (\"%s\")", v))
	}

	if len(excludeVersions) > 0 {
		v := strings.Join(excludeVersions, "\",\"")
		filter = append(filter, fmt.Sprintf("fixVersion NOT IN (\"%s\")", v))
	}

	return strings.Join(filter, " AND ") + " ORDER BY id ASC"
}
