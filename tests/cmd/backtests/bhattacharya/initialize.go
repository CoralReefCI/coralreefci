package main

import (
	"fmt"
	"strings"

	. "github.com/ahmetalpbalkan/go-linq"
	"github.com/google/go-github/github"
	"go.uber.org/zap"
	"golang.org/x/oauth2"

	"core/models"
	"core/models/bhattacharya"
	"core/pipeline/gateway"
	conf "core/pipeline/gateway/conflation"
	"core/utils"
)

type TestContext struct {
	Model models.Model
}

type BackTestRunner struct {
	Context TestContext
}

func (t *BackTestRunner) Run(repo string) {

	// defer func() {
	// 	//utils.Log.Error("Panic Recovered: ", recover(), bytes.NewBuffer(debug.Stack()).String())
	// }()

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "c813d7dab123d3c4813618bf64503a7a1efa540f"})
	tc := oauth2.NewClient(oauth2.NoContext, ts)
	client := github.NewClient(tc)
	newGateway := gateway.CachedGateway{Gateway: &gateway.Gateway{Client: client}, DiskCache: &gateway.DiskCache{}}

	r := strings.Split(repo, "/")
	githubIssues, err := newGateway.GetIssues(r[0], r[1])
	if err != nil {
		utils.AppLog.Error("Cannot get Issues from Github Gateway.", zap.Error(err))
	}
	githubPulls, err := newGateway.GetPullRequests(r[0], r[1])
	if err != nil {
		utils.AppLog.Error("Cannot get PullRequests from Github Gateway.", zap.Error(err))
	}

	context := &conf.Context{}

	// NOTE: Changing the scenarios will allow different objects in.
	// NOTE: THIS CAN BE MANIPULATED
	scenarios := []conf.Scenario{&conf.Scenario3{}, &conf.Scenario4{}}

	conflationAlgorithms := []conf.ConflationAlgorithm{&conf.ComboAlgorithm{Context: context}}
	normalizer := conf.Normalizer{Context: context}
	conflator := conf.Conflator{Scenarios: scenarios, ConflationAlgorithms: conflationAlgorithms, Normalizer: normalizer, Context: context}

	conflator.Context.Issues = []conf.ExpandedIssue{}
	conflator.SetIssueRequests(githubIssues)
	conflator.SetPullRequests(githubPulls)
	conflator.Conflate()

	trainingSet := []conf.ExpandedIssue{}

	fmt.Println(len(conflator.Context.Issues)) // TEMPORARY
	for i := 0; i < len(conflator.Context.Issues); i++ {
		expandedIssue := conflator.Context.Issues[i]
		if expandedIssue.Conflate {
			if expandedIssue.Issue.Assignees != nil && len(expandedIssue.Issue.Assignees) > 0 || expandedIssue.PullRequest.User != nil {
				trainingSet = append(trainingSet, conflator.Context.Issues[i])
			}
		}
	}

	openSet := []conf.ExpandedIssue{}
	for i := range trainingSet {
		if trainingSet[i].Issue.ID != nil {
			if *trainingSet[i].Issue.State == "open" {
				openSet = append(openSet, trainingSet[i])
			}
		}
	}

	utils.ModelLog.Info("Training set size (before Linq): ", zap.Int("TrainingSetSize", len(trainingSet)))
	fmt.Println("Training set size (before Linq): ", len(trainingSet))
	processedTrainingSet := []conf.ExpandedIssue{}

	excludeAssignees := From(trainingSet).Where(func(exclude interface{}) bool {
		if exclude.(conf.ExpandedIssue).Issue.Assignee != nil {
			assignee := *exclude.(conf.ExpandedIssue).Issue.Assignee.Login

			switch assignee {
			case
				"forstmeier",
				"fishera123",
				"irJERAD",
				"konstantinTarletskis",
				"hadim":
				return false
			}
		}
		// NOTE: THIS CAN BE MANIPULATED
		// return assignee != "AndyAyersMS" && assignee != "CarolEidt" && assignee != "mikedn" && assignee != "pgavlin" && assignee != "BruceForstall" && assignee != "RussKeldorph" && assignee != "sdmaclea"
		// return assignee != "dotnet-bot" && assignee != "dotnet-mc-bot" && assignee != "00101010b"
		// return assignee != "forstmeier" && assignee != "fishera123" && assignee != "irJERAD" && assignee != "konstantinTarletskis" && assignee != "hadim"
		return true
	})

	groupby := excludeAssignees.GroupBy(
		func(r interface{}) interface{} {
			if r.(conf.ExpandedIssue).Issue.Assignee != nil {
				return *r.(conf.ExpandedIssue).Issue.Assignee.ID
			} else {
				return *r.(conf.ExpandedIssue).PullRequest.User.ID
			}
		}, func(r interface{}) interface{} {
			return r.(conf.ExpandedIssue)
		})

	where := groupby.Where(func(groupby interface{}) bool {
		// NOTE: THIS CAN BE MANIPULATED (between 10-15 max so far)
		return len(groupby.(Group).Group) >= 10
	})

	orderby := where.OrderByDescending(func(where interface{}) interface{} {
		return len(where.(Group).Group)
	}).ThenBy(func(where interface{}) interface{} {
		return where.(Group).Key
	})

	orderby.SelectMany(func(orderby interface{}) Query {
		return From(orderby.(Group).Group).OrderBy(
			func(where interface{}) interface{} {
				if where.(conf.ExpandedIssue).Issue.ID != nil {
					return *where.(conf.ExpandedIssue).Issue.ID
				} else {
					return *where.(conf.ExpandedIssue).PullRequest.ID
				}
			}).Query
	}).ToSlice(&processedTrainingSet)

	Shuffle(processedTrainingSet, int64(5))

	for i := range processedTrainingSet {
		replace := ""
		if processedTrainingSet[i].Issue.ID != nil {
			if processedTrainingSet[i].Issue.Body == nil {
				processedTrainingSet[i].Issue.Body = &replace
			}
		} else {
			if processedTrainingSet[i].PullRequest.Body == nil {
				processedTrainingSet[i].PullRequest.Body = &replace
			}
		}
	}

	utils.ModelLog.Info("Backtest model training...")
	fmt.Println("Training set size: ", len(processedTrainingSet))

	scoreJohn := t.Context.Model.JohnFold(processedTrainingSet)
	fmt.Println("John Fold:", scoreJohn)

	for i := range openSet {
		predictions := t.Context.Model.Predict(openSet[i])
		nbm := t.Context.Model.Algorithm.(*bhattacharya.NBModel)
		nbm.GenerateProbabilityTable(
			*openSet[i].Issue.ID,
			*openSet[i].Issue.Body,
			predictions,
			"open",
		)
	}
}
