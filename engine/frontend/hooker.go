package frontend

import (
	// "net/http"

	"coralreefci/engine/gateway"
	"coralreefci/engine/gateway/conflation"
	"coralreefci/models"
	"coralreefci/models/bhattacharya"
	"coralreefci/utils"
	"github.com/google/go-github/github"
)

func (h *HeuprServer) NewHook(repo *github.Repository, client *github.Client) error {
	if check, err := h.CheckHook(repo, client); check {
		// TODO: Logic for handling an error here will be implemented; this
		//       will take the form of an exit from the parent NewHook method
		//       as well as a generation of an error/redirect page option to
		//       the end user of the Heupr application.
		return err
	}

	name := *repo.Name
	owner := *repo.Owner.Login
	url := "http://758a4cc0.ngrok.io/hook"
	secret := "chalmun's-spaceport-cantina"

	hook, _, err := client.Repositories.CreateHook(owner, name, &github.Hook{
		Name:   github.String("web"),
		Events: []string{"issues"},
		Active: github.Bool(true),
		Config: map[string]interface{}{
			"url":          url,
			"secret":       secret,
			"content_type": "json",
			"insecure_ssl": false,
		},
	})
	if err != nil {
		return err
	}

	err = storeData(*repo.ID, "hookID", *hook.ID)
	if err != nil {
		return err
	}
	return nil
}

func (h *HeuprServer) CheckHook(repo *github.Repository, client *github.Client) (bool, error) {
	name := *repo.Name
	owner := *repo.Owner.Login
	hookID, err := retrieveData(*repo.ID, "hookID")
	if err != nil {
		return false, err
	}

	_, _, err = client.Repositories.GetHook(owner, name, hookID.(int))
	if err != nil {
		return false, err
	}
	return true, nil
}

func (h *HeuprServer) AddModel(repo *github.Repository, client *github.Client) error {
	name := *repo.Name
	owner := *repo.Owner.Login

	h.Models[555] = models.Model{Algorithm: &bhattacharya.NBModel{}}
	//TODO: The comments field is not cached when using CachedGateway and will
	//      need to be fixed eventually.
	newGateway := gateway.Gateway{Client: client}
	githubIssues, err := newGateway.GetIssues(owner, name)
	if err != nil {
		utils.Log.Error("Cannot get Issues from Gateway. ", err)
	}
	githubPulls, err := newGateway.GetPullRequests(owner, name)
	if err != nil {
		utils.Log.Error("Cannot get PullRequests from Gateway. ", err)
	}

	context := &conflation.Context{}
	scenarios := []conflation.Scenario{&conflation.Scenario2{}}
	conflationAlgorithms := []conflation.ConflationAlgorithm{
		&conflation.ComboAlgorithm{Context: context},
	}
	normalizer := conflation.Normalizer{Context: context}
	conflator := conflation.Conflator{
		Scenarios:            scenarios,
		ConflationAlgorithms: conflationAlgorithms,
		Normalizer:           normalizer,
		Context:              context,
	}

	issuesCopy := make([]github.Issue, len(githubIssues))
	pullsCopy := make([]github.PullRequest, len(githubPulls))

	// TODO: Evaluate this particular snippet of code as it has potential
	//       performance optimization capabilities related to the hardware
	//       level. This may ultimately live in the actual gateway.go file to
	//	     improve the actual download operations.
	for i := 0; i < len(issuesCopy); i++ {
		issuesCopy[i] = *githubIssues[i]
	}
	for i := 0; i < len(pullsCopy); i++ {
		pullsCopy[i] = *githubPulls[i]
	}

	conflator.Context.Issues = []conflation.ExpandedIssue{}
	conflator.SetIssueRequests(issuesCopy)
	conflator.SetPullRequests(pullsCopy)
	conflator.Conflate()

	trainingSet := []conflation.ExpandedIssue{}

	for i := 0; i < len(conflator.Context.Issues); i++ {
		expandedIssue := conflator.Context.Issues[i]
		if expandedIssue.Conflate {
			if expandedIssue.Issue.Assignee == nil {
				continue
			} else {
				trainingSet = append(trainingSet, conflator.Context.Issues[i])
			}
		}
	}
	h.Models[555].Algorithm.Learn(trainingSet)
	return nil
}