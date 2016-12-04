package conflation

type Scenario3a struct {
}

// DOC: Scenario3a filters for "naked" pull requests.
//      These are pull requests without an associated issue.
func (s *Scenario3a) Filter(expandedIssue *ExpandedIssue) bool {
	result := false
	if expandedIssue.PullRequest.IssueURL == nil {
		result = true
	}
	return result
}
