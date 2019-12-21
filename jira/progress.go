package jira

// Progress represent the progress in a series of activities
type Progress struct {
	Status  int
	Total   int
	Unknown int
}

// Percentage returns the percentage of the progress
func (p *Progress) Percentage() float64 {
	return float64(p.Status) / float64(p.Total)

}

// Remaining returns the number of items remaining
func (p *Progress) Remaining() int {
	return p.Total - p.Status
}
