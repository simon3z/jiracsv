package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strings"
	"syscall"

	jira "github.com/andygrunwald/go-jira"
	"golang.org/x/crypto/ssh/terminal"
)

// PasswordEnvVariableName is the environment variable name used to
var PasswordEnvVariableName = "PASSWORD"

// DeliveryOwnerRegExp is the Regular Expression used to collect the Epic Delivery Owner
var DeliveryOwnerRegExp = `\W*(Delivery Owner|DELIVERY OWNER)\W*:\W*\[~([a-zA-Z0-9]*)\]`

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

// CmdFlags represent the command line flag options
var CmdFlags = struct {
	JiraURL    string
	Username   string
	Password   string
	Project    string
	Version    string
	Components ArrayFlag
}{}

func jiraIssueLink(i *jira.Issue) string {
	return strings.Join([]string{CmdFlags.JiraURL, "browse", i.Key}, "/")
}

func googleSheetLink(link, text string) string {
	return fmt.Sprintf("=HYPERLINK(\"%s\",\"%s\")", link, text)
}

func jiraJQLSearch(project, version, component string) string {
	filter := []string{
		fmt.Sprintf("status != \"Obsolete\""),
	}

	if project != "" {
		filter = append(filter, fmt.Sprintf("project = \"%s\"", project))
	}

	if version != "" {
		filter = append(filter, fmt.Sprintf("fixVersion = \"%s\"", version))
	}

	if component != "" {
		filter = append(filter, fmt.Sprintf("component = \"%s\"", component))
	}

	return strings.Join(filter, " AND ") + " ORDER BY id ASC"
}

func getDeliveryOwner(i *jira.Issue) string {
	matches := regexp.MustCompile(DeliveryOwnerRegExp).FindStringSubmatch(i.Fields.Description)

	if len(matches) == 3 {
		return matches[2]
	}

	return ""
}

func iterateUnknownMaps(i *jira.Issue, f func(m map[string]interface{})) {
	for _, v := range i.Fields.Unknowns {
		if l, isSlice := v.([]interface{}); isSlice {
			for _, j := range l {
				f(j.(map[string]interface{}))
			}
		}
	}
}

func getAcksStatus(i *jira.Issue) string {
	acks := struct {
		DevelAck bool `value:"devel_ack"`
		PMAck    bool `value:"pm_ack"`
		QEAck    bool `value:"qa_ack"`
		UXAck    bool `value:"ux_ack"`
		DocAck   bool `value:"doc_ack"`
	}{false, false, false, false, false}

	acksType := reflect.TypeOf(acks)
	acksValue := reflect.ValueOf(&acks)

	iterateUnknownMaps(i, func(m map[string]interface{}) {
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

func init() {
	flag.StringVar(&CmdFlags.JiraURL, "h", "", "Jira instance URL")
	flag.StringVar(&CmdFlags.Username, "u", "", "Jira username")
	flag.StringVar(&CmdFlags.Project, "p", "", "Project to use to collect Issues")
	flag.StringVar(&CmdFlags.Version, "v", "", "Version to use to collect Issues")
	flag.Var(&CmdFlags.Components, "c", "Components to use to collect Issues")
}

func getPassword() {
	CmdFlags.Password = os.Getenv(PasswordEnvVariableName)

	if CmdFlags.Password == "" && terminal.IsTerminal(syscall.Stdin) {
		os.Stdin.Write([]byte("Password: "))

		pw, err := terminal.ReadPassword(syscall.Stdin)
		defer os.Stdin.Write([]byte("\n"))

		if err != nil {
			panic(err)
		}

		CmdFlags.Password = string(pw)
	}
}

func main() {
	flag.Parse()

	client := (*http.Client)(nil)

	if CmdFlags.Username != "" {
		getPassword()

		transport := jira.BasicAuthTransport{
			Username: CmdFlags.Username,
			Password: CmdFlags.Password,
		}

		client = transport.Client()
	}

	jiraClient, err := jira.NewClient(client, CmdFlags.JiraURL)

	if err != nil {
		panic(err)
	}

	w := csv.NewWriter(os.Stdout)
	w.Comma = '\t'

	for _, component := range CmdFlags.Components {
		issues, ret, err := jiraClient.Issue.Search(jiraJQLSearch(CmdFlags.Project, CmdFlags.Version, component), nil)

		if err != nil {
			if ret.Response.StatusCode == http.StatusForbidden || ret.Response.StatusCode == http.StatusUnauthorized {
				panic("Access Unauthorized: check basic authentication using a browser and retry")
			} else {
				panic(err)
			}
		}

		w.Write([]string{component})

		for _, i := range issues {
			w.Write([]string{
				googleSheetLink(jiraIssueLink(&i), i.Key),
				i.Fields.Summary,
				i.Fields.Priority.Name,
				i.Fields.Status.Name,
				getDeliveryOwner(&i),
				getAcksStatus(&i),
			})
		}

		w.Flush()
	}
}
