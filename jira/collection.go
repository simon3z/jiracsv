package jira

// IssueCollection is a collection of Jira Issues
type IssueCollection []*Issue

// NewIssueCollection creates and returns a new Jira Issue Collection
func NewIssueCollection(size int) IssueCollection {
	return make([]*Issue, size)
}

// FilterByFunction returns jira issues from collection that satisfy the provided function
func (c IssueCollection) FilterByFunction(fn func(*Issue) bool) IssueCollection {
	r := NewIssueCollection(0)

	for _, i := range c {
		if fn(i) {
			r = append(r, i)
		}
	}

	return r
}

// StoryPoints returns the total number of story points for the issues in the collection
func (c IssueCollection) StoryPoints() int {
	points := 0

	for _, i := range c {
		if i.HasStoryPoints() {
			points += i.StoryPoints
		}
	}

	return points
}

// AnyImpediment returns if any of the issues has an impediment
func (c IssueCollection) AnyImpediment() bool {
	for _, i := range c {
		if i.Impediment {
			return true
		}
	}

	return false
}

// Progress returns the progress of the issues in the collection
func (c IssueCollection) Progress() Progress {
	p := Progress{Total: 0, Status: 0, Unknown: 0}

	for _, i := range c {
		if i.InStatus(IssueStatusObsolete) {
			continue
		}

		p.Total = p.Total + 1

		if i.IsResolved() {
			p.Status = p.Status + 1
		}
	}

	return p
}

// StoryPointsProgress returns the progress of the story points of the issues in the collection
func (c IssueCollection) StoryPointsProgress() Progress {
	p := Progress{Total: 0, Status: 0, Unknown: 0}

	for _, i := range c {
		if !i.IsType(IssueTypeStory) || i.InStatus(IssueStatusObsolete) {
			continue
		}

		if i.HasStoryPoints() {
			p.Total = p.Total + i.StoryPoints

			if i.IsResolved() {
				p.Status = p.Status + i.StoryPoints
			}
		} else {
			p.Unknown = p.Unknown + 1
		}
	}

	return p
}
