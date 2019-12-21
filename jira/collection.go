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

// Len returns the number of issues in the collection
func (c IssueCollection) Len() int {
	return len(c)
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
