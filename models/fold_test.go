package models

import (
	"coralreefci/engine/gateway/conflation"
    "coralreefci/models/bhattacharya"
	"github.com/google/go-github/github"
	"strconv"
	"testing"
    "time"
)

func buildTestIssues() []conflation.ExpandedIssue {
	issues := []conflation.ExpandedIssue{}
	for i := 1; i < 21; i++ {
        id := i
        number := 10 + i
        url := "http://podracing.com"
        title := "Boonta Eve Classic"
        assignee := github.User{}
		if i%2 == 0 {
			login := "Anakin"
			assignee = github.User{Login: &login}
		} else {
			login := "Sebulba"
			assignee = github.User{Login: &login}
		}
        body := "Let the race begin!"
        resolved := time.Time{}
        name := "pit-droid"
        labels := []github.Label{github.Label{Name: &name}}
        githubIssue := github.Issue{ID: &id, Number: &number, URL: &url, Title: &title, Assignee: &assignee, Body: &body, ClosedAt: &resolved, Labels: labels}
		// NOTE: THIS IS THE PROPER INITIALIZATION WITH EMPTY VALUES FOR ALL FIELDS
		crIssue := conflation.CRIssue{githubIssue, []int{}, []conflation.CRPullRequest{}}
		issues = append(issues, conflation.ExpandedIssue{Issue: crIssue})
	}
	return issues
}

// TODO: ADD IN ADDITIONAL UNIT TESTS FOR
//       - LENGTH OF ISSUES TESTING ON (FAIL LESS THAN 10)
//       - ALL OTHER FOLD PARENTS
func TestFold(t *testing.T) {
    nbModel := Model{Algorithm: &bhattacharya.NBClassifier{}}
	testingIssues := buildTestIssues()
	result := nbModel.JohnFold(testingIssues)
	number, _ := strconv.ParseFloat(result, 64)
	if number < 0.00 && number > 1.00 {
		t.Error(
			"\nRESULT IS OUTSIDE ACCEPTABLE RANGE - JOHN FOLD",
			"\nEXPECTED BETWEEN 0.00 AND 1.00",
			"\nACTUAL: %f", number,
		)
	}
}
