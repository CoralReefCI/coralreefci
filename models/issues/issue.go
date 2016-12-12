/*
The issues package creates an interal structure containing relavent GitHub object information.
This provides a universal object that CoralReefCI can operate on and pass around.
*/
package issues

import (
	"time"
)

type Issue struct {
	RepoID   int
	IssueID  int
	URL      string
	Assignee string
	Body     string
	Resolved time.Time
	Labels   []string
}
