package bhattacharya

import (
	"coralreefci/models/issues"
	"strings"
	"testing"
)

var withStopwords = "This is a sentence with stopwords in it, the best ever."
var withoutStopwords = "sentence stopwords"

var testIssue = []bhattacharyaIssue{bhattacharyaIssue{Body: withStopwords}}

func TestRemoveStopWords(t *testing.T) {
	RemoveStopWords(testIssue...)
	issueBody := testIssue[0].Body
	if strings.Contains(issueBody, "this") || strings.Contains(issueBody, "with") || strings.Contains(issueBody, "the") {
		t.Error(
			"\nSTOPWORDS NOT REMOVED",
			"\nEXPECTED: ", withoutStopwords,
			"\nACTUAL:   ", issueBody,
		)
	}
}
