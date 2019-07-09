package analysis

import (
	"strings"

	"github.com/simon3z/jiracsv/jira"
)

// CheckResultStatus represent the status of a check
type CheckResultStatus int

const (
	// CheckStatusNone represent an unknown status
	CheckStatusNone CheckResultStatus = iota

	// CheckStatusGreen represent no impediment and confidence to deliver in time
	CheckStatusGreen

	// CheckStatusYellow represents minor impediments that could put at risk the delivery in time
	CheckStatusYellow

	// CheckStatusRed represents impediments, major roadblocks and impossibility to deliver in time
	CheckStatusRed
)

type CheckResult struct {
	Ready    bool
	Status   CheckResultStatus
	Messages []string
}

func (s CheckResultStatus) String() string {
	switch s {
	case CheckStatusNone:
		return "NONE"
	case CheckStatusGreen:
		return "GREEN"
	case CheckStatusYellow:
		return "YELLOW"
	case CheckStatusRed:
		return "RED"
	}

	return "UNKOWN"
}

// NewCheckResult returns a new CheckResult
func NewCheckResult(a *IssueAnalysis) *CheckResult {
	result := &CheckResult{
		Ready:  true,
		Status: CheckStatusNone,
	}

	if a.Issue.InStatus(jira.IssueStatusObsolete) {
		result.AddMessage("OBSOLETE")
		return result
	}

	checks := []func(*IssueAnalysis){
		result.checkAlongside,
		result.checkVersion,
		result.checkActivities,
		result.checkDescription,
		result.checkReadiness,
		result.checkApprovals,
		// result.checkPlanning,
		result.checkDeliveryOwner,
		result.checkQEAssignee,
		result.checkAcceptanceCriteria,
		result.checkPriority,
		result.checkStarted,
		// result.checkStoryPoints,
		result.checkImpediment,
		result.checkMarketProblem,
		result.checkIssueComponent,
		result.checkMultiComponent,
		result.checkDone,
		result.checkStartedStories,
		result.checkLinkedEpic,
		result.checkStatusComment,
		result.checkDesign,
	}

	for _, f := range checks {
		f(a)
	}

	return result
}

// SetReady sets the ready status
func (r *CheckResult) SetReady(ready bool) *CheckResult {
	if r.Ready && !ready {
		r.Ready = false
	}
	return r
}

// SetStatus sets the status
func (r *CheckResult) SetStatus(status CheckResultStatus) *CheckResult {
	if status > r.Status {
		r.Status = status
	}
	return r
}

// AddMessage adds a message
func (r *CheckResult) AddMessage(message string) *CheckResult {
	r.Messages = append(r.Messages, message)
	return r
}

func (r *CheckResult) checkAlongside(a *IssueAnalysis) {
	for _, v := range a.Issue.Fields.FixVersions {
		if strings.HasPrefix(v.Name, "Alongside") {
			r.AddMessage("ALONGSIDE")
			return
		}
	}
}

// checkVersion verifies that there is at least one version set
func (r *CheckResult) checkVersion(a *IssueAnalysis) {
	if len(a.Issue.Fields.FixVersions) == 0 {
		r.SetReady(false).AddMessage("NOVERSION")
	}

	if len(a.Issue.Fields.FixVersions) > 1 {
		r.AddMessage("MULTIVERSION")
	}
}

// checkActivities verifies that there is at least one story attached
func (r *CheckResult) checkActivities(a *IssueAnalysis) {
	if a.Issue.IsType(jira.IssueTypeEpic) && a.NumActivities == 0 {
		r.SetReady(false).AddMessage("NOSTORIES")
	}
}

// checkDescription verifies that the description is set
func (r *CheckResult) checkDescription(a *IssueAnalysis) {
	if a.Issue.Fields.Description == "" {
		r.SetReady(false).AddMessage("NODESCRIPTION")
	}
}

// checkReadiness verifies that the ready checks are set
func (r *CheckResult) checkReadiness(a *IssueAnalysis) {
	if a.Issue.IsType(jira.IssueTypeEpic) && !a.Issue.Ready() {
		r.SetReady(false).AddMessage("NOTREADY")
	}
}

// checkApprovals verifies that the approvals are set
func (r *CheckResult) checkApprovals(a *IssueAnalysis) {
	if a.Issue.IsType(jira.IssueTypeEpic) && !a.Issue.IsCommitted() {
		r.SetStatus(CheckStatusRed).AddMessage("NOACKS")
	}
}

// checkDeliveryOwner verifies that an owner has been assigned
func (r *CheckResult) checkDeliveryOwner(a *IssueAnalysis) {
	if a.Issue.Owner == "" {
		r.SetReady(false).SetStatus(CheckStatusRed).AddMessage("NODELIVERYOWNER")
	}
}

func (r *CheckResult) checkPlanning(a *IssueAnalysis) {
	if a.Issue.Planning.NoQuality {
		r.AddMessage("NOQE")
	}
	if a.Issue.Planning.NoFeature {
		r.AddMessage("NOFEATURE")
	}
	if a.Issue.Planning.NoDocumentation {
		r.AddMessage("NODOC")
	}
}

// checkQEAssignee verifies that a QA Contact has been assigned
func (r *CheckResult) checkQEAssignee(a *IssueAnalysis) {
	if a.Issue.Planning.NoQuality {
		if a.Issue.QEAssignee != "" {
			r.SetReady(false).AddMessage("NOQEMISMATCH")
		}
	} else if a.Issue.QEAssignee == "" {
		r.SetReady(false).SetStatus(CheckStatusRed).AddMessage("NOQEASSIGNEE")
	}
}

// checkAcceptanceCriteria verifies that the acceptance criteria are set
func (r *CheckResult) checkAcceptanceCriteria(a *IssueAnalysis) {
	if a.Issue.Acceptance == "" {
		r.SetReady(false).SetStatus(CheckStatusRed).AddMessage("NOCRITERIA")
	}
}

// checkPriority veirfies that the priority is set
func (r *CheckResult) checkPriority(a *IssueAnalysis) {
	if !a.Issue.IsPrioritized() {
		r.SetReady(false).SetStatus(CheckStatusRed).AddMessage("NOPRIORITY")
	}
}

// checkStarted verifies that the status is active or done
func (r *CheckResult) checkStarted(a *IssueAnalysis) {
	if !a.Issue.IsActive() && !a.Issue.InStatus(jira.IssueStatusDone) {
		r.SetStatus(CheckStatusYellow).AddMessage("NOTSTARTED")
	}
}

// checkStoryPoints verifies that all the stories have story points
func (r *CheckResult) checkStoryPoints(a *IssueAnalysis) {
	if a.PointsCompletion.Unknown > 0 {
		r.AddMessage("NOSTORYPOINTS")
	}
}

// checkImpediment notifies if there is an impediment flagged
func (r *CheckResult) checkImpediment(a *IssueAnalysis) {
	if a.Issue.Impediment || a.Issue.LinkedIssues.AnyImpediment() {
		r.SetStatus(CheckStatusRed).AddMessage("IMPEDIMENT")
	}
}

// checkMarketProblem verifies that there is a linked initiative
func (r *CheckResult) checkMarketProblem(a *IssueAnalysis) {
	if a.Issue.IsType(jira.IssueTypeEpic) && a.Issue.MarketProblem == nil {
		r.SetReady(false).AddMessage("NOMARKETPROBLEM")
	}
}

// checkIssueComponent verifies that all the linked issues have at least a component
func (r *CheckResult) checkIssueComponent(a *IssueAnalysis) {
	if a.IssueNoComponent {
		r.SetReady(false).AddMessage("ISSUENOCOMPONENT")
	}
}

// checkMultiComponent verifies that the relevant component is
func (r *CheckResult) checkMultiComponent(a *IssueAnalysis) {
	if a.Component == nil {
		return
	}

	if len(a.Issue.Fields.Components) != 1 {
		r.SetReady(false).SetStatus(CheckStatusYellow).AddMessage("MULTICOMPONENT")
		return
	}
}

// checkDone verifies that all the conditions are met for the done status
func (r *CheckResult) checkDone(a *IssueAnalysis) {
	if !a.Issue.InStatus(jira.IssueStatusDone) {
		return
	}

	if a.IssuesCompletion.Status != a.IssuesCompletion.Total ||
		a.PointsCompletion.Status != a.PointsCompletion.Total {
		r.SetStatus(CheckStatusRed).AddMessage("NOTDONE")
	} else {
		r.SetStatus(CheckStatusGreen)
	}
}

// checkStatusComment verifies the status comment
func (r *CheckResult) checkStatusComment(a *IssueAnalysis) {
	if a.CommentStatus == CheckStatusNone {
		r.AddMessage("NOSTATUSCOMMENT")
	} else {
		r.SetStatus(a.CommentStatus)
	}
}

func (r *CheckResult) checkLinkedEpic(a *IssueAnalysis) {
	if !a.Issue.IsType(jira.IssueTypeStory) {
		return
	}

	if a.Issue.Fields.Epic == nil || a.Issue.Fields.Epic.Key == "" {
		r.SetReady(false).AddMessage("NOEPIC")
	}
}

func (r *CheckResult) checkStartedStories(a *IssueAnalysis) {
	if !a.Issue.IsType(jira.IssueTypeEpic) || !a.Issue.IsActive() {
		return
	}

	linkedIssues := a.Issue.LinkedIssues

	if a.Component != nil {
		linkedIssues = linkedIssues.FilterByFunction(func(i *jira.Issue) bool {
			return i.HasComponent(*a.Component)
		})
	}

	activeIssues := linkedIssues.FilterByFunction(func(i *jira.Issue) bool {
		return i.IsActive() || i.InStatus(jira.IssueStatusDone)
	})

	if len(activeIssues) == 0 {
		r.SetStatus(CheckStatusRed).AddMessage("NOACTIVESTORIES")
	}
}

func (r *CheckResult) checkVersions(a *IssueAnalysis) {
	if !a.Issue.IsType(jira.IssueTypeEpic) {

	}
}

func (r *CheckResult) checkDesign(a *IssueAnalysis) {
	if !a.Issue.Planning.NoFeature && a.Issue.Design == "" {
		r.SetReady(false).AddMessage("NODESIGN")
	}
}
