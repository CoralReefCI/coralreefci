package models

import (
	"strconv"
	"testing"
	"time"

	"github.com/google/go-github/github"

	"core/models/bhattacharya"
	"core/pipeline/gateway/conflation"
)

func buildTestIssues() []conflation.ExpandedIssue {
	issues := []conflation.ExpandedIssue{}
	for i := 1; i < 21; i++ {
		id := int64(i)
		number := 10 + i
		check := new(bool)
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
		assignees := []*github.User{&assignee}
		body := "Let the race begin!"
		resolved := time.Time{}
		name := "pit-droid"
		labels := []github.Label{github.Label{Name: &name}}
		githubIssue := github.Issue{
			ID:        &id,
			Number:    &number,
			URL:       &url,
			Title:     &title,
			Assignees: assignees,
			Body:      &body,
			ClosedAt:  &resolved,
			Labels:    labels,
		}
		crIssue := conflation.CRIssue{
			githubIssue,
			[]int{},
			[]conflation.CRPullRequest{},
			check,
			check,
		}
		issues = append(issues, conflation.ExpandedIssue{Issue: crIssue})
	}
	return issues
}

func TestJohnFold(t *testing.T) {
	nbModel := Model{Algorithm: &bhattacharya.NBModel{}}
	testingIssues := buildTestIssues()
	result := nbModel.JohnFold(testingIssues)
	if result < 0.00 && result > 1.00 {
		t.Error(
			"\nRESULT IS OUTSIDE ACCEPTABLE RANGE - JOHN FOLD",
			"\nEXPECTED BETWEEN 0.00 AND 1.00 - ACTUAL: %f", result,
		)
	}
}

func TestTwoFold(t *testing.T) {
	nbModel := Model{Algorithm: &bhattacharya.NBModel{}}
	testingIssues := buildTestIssues()
	result := nbModel.TwoFold(testingIssues)
	number, _ := strconv.ParseFloat(result, 64)
	if number < 0.00 && number > 1.00 {
		t.Error(
			"\nRESULT IS OUTSIDE ACCEPTABLE RANGE - TWO FOLD",
			"\nEXPECTED BETWEEN 0.00 AND 1.00 - ACTUAL: %f", number,
		)
	}
}

func TestTenFold(t *testing.T) {
	nbModel := Model{Algorithm: &bhattacharya.NBModel{}}
	testingIssues := buildTestIssues()
	result := nbModel.TenFold(testingIssues)
	number, _ := strconv.ParseFloat(result, 64)
	if number < 0.00 && number > 1.00 {
		t.Error(
			"\nRESULT IS OUTSIDE ACCEPTABLE RANGE - TEN FOLD",
			"\nEXPECTED BETWEEN 0.00 AND 1.00 - ACTUAL: %f", number,
		)
	}
}
