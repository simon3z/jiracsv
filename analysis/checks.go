package analysis

import "strings"

// CheckResultStatus represent the status of a check
type CheckResultStatus int

const (
	// CheckStatusNone represent an unknown status
	CheckStatusNone CheckResultStatus = iota

	// CheckStatusGreen represent no impediment and confidence to deliver in time
	CheckStatusGreen

	// CheckStatusYellow represents minor impediments that could put at risk the delivery in time
	CheckStatusYellow

	// CheckStatusRed represents impediments, major roadbloack and the impossibility to deliver in time
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
func NewCheckResult(ready bool, status CheckResultStatus) *CheckResult {
	return &CheckResult{
		Ready:  ready,
		Status: status,
	}
}

func (r *CheckResult) SetReady(ready bool) *CheckResult {
	if r.Ready && !ready {
		r.Ready = false
	}
	return r
}

func (r *CheckResult) SetStatus(status CheckResultStatus) *CheckResult {
	if status > r.Status {
		r.Status = status
	}
	return r
}

func (r *CheckResult) AddMessage(message string) *CheckResult {
	r.Messages = append(r.Messages, message)
	return r
}

func (r *CheckResult) MessagesString() string {
	return strings.Join(r.Messages, ",")
}
