package ingestor

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"

	"github.com/boltdb/bolt"
	"github.com/google/go-github/github"
	"go.uber.org/zap"
	"golang.org/x/oauth2"

	"core/pipeline/frontend"
	"core/utils"
	"fmt"
)

const RESTART_QUERY = `
SELECT MAX(ghe.number)
FROM (
    SELECT repo_id, number
    FROM github_events
    WHERE is_pull = false
    ORDER BY repo_id, number
) ghe
WHERE repo_id = ?

UNION ALL

SELECT MAX(ghe.number)
FROM (
    SELECT repo_id, number
    FROM github_events
    WHERE is_pull = true
    ORDER BY repo_id, number
) ghe
WHERE repo_id = ?
`

func tokenizer(tokenByte []byte) (*github.Client, error) {
	token := oauth2.Token{}
	if err := json.Unmarshal(tokenByte, &token); err != nil {
		utils.AppLog.Error("converting tokens; ", zap.Error(err))
		return nil, err
	}
	client := makeClient(token)
	return &client, nil
}

func (i *IngestorServer) Restart() error {
	bufferPool := NewPool()
	i.Database = Database{BufferPool: bufferPool}
	i.Database.Open()
	defer i.Database.Close()

	db, err := bolt.Open(utils.Config.BoltDBPath, 0644, nil)
	if err != nil {
		utils.AppLog.Error("Failed opening bolt on ingestor restart;", zap.Error(err))
		return err
	}
	defer db.Close()

	boltDB := frontend.BoltDB{DB: db}
	if err := boltDB.Initialize(); err != nil {
		utils.AppLog.Error("Ingestor server", zap.Error(err))
		panic(err)
	}

	repos, tokens, err := boltDB.RetrieveBulk("token")
	if err != nil {
		utils.AppLog.Error("Retrieve bulk tokens on ingestor restart;", zap.Error(err))
	}

	for key := range tokens {
		client, err := tokenizer(tokens[key])
		if err != nil {
			return err
		}

		repoID, err := strconv.Atoi(string(repos[key]))
		if err != nil {
			utils.AppLog.Error("Repo id int conversion;", zap.Error(err))
			return err
		}

		repo, _, err := client.Repositories.GetByID(context.Background(), repoID)
		if err != nil {
			utils.AppLog.Error("Ingestor restart get by id;", zap.Error(err))
			return err
		}

		owner := repo.Owner.Login
		name := repo.Name

		iOldest, pOldest, iNewest, pNewest := new(int), new(int), new(int), new(int)
		result := i.Database.db.QueryRow(RESTART_QUERY, repoID, repoID).Scan(&iOldest, &pOldest)
		switch {
		case result == sql.ErrNoRows:
			utils.AppLog.Error("no rows in restart query: ", zap.Error(result))
			break
		case result != nil:
			utils.AppLog.Error("restart query: ", zap.Error(result))
		default:
			continue
		}

		fmt.Println(*iOldest)
		if *iOldest == 0 && *pOldest == 0 {
			authRepo := AuthenticatedRepo{
				Repo:   repo,
				Client: client,
			}
			fmt.Println("i.RepoInitializer.AddRepo(authRepo)")
			i.RepoInitializer = RepoInitializer{}
			i.RepoInitializer.AddRepo(authRepo)
		}
		if iOldest == nil {
			iOldest = iNewest
		}
		if pOldest == nil {
			pOldest = pNewest
		}

		issue, _, err := client.Issues.ListByRepo(context.Background(), *owner, *name, &github.IssueListByRepoOptions{
			ListOptions: github.ListOptions{
				PerPage: 1,
			},
		})
		if err != nil {
			utils.AppLog.Error("newest issue retrival; ", zap.Error(err))
		} else {
			iNewest = issue[0].Number
		}

		iDiff := *iNewest - *iOldest
		missingIssues := []*github.Issue{}
		for iDiff > 1 {
			opts := github.IssueListByRepoOptions{
				ListOptions: github.ListOptions{},
			}
			switch {
			case iDiff > 1 && iDiff <= 100:
				opts.ListOptions.PerPage = iDiff
				iDiff = 0
			case iDiff > 100:
				opts.ListOptions.PerPage = 100
				iDiff = iDiff - 100
			}
			issues, resp, err := client.Issues.ListByRepo(context.Background(), *owner, *name, &opts)
			if err != nil {
				utils.AppLog.Error("newest issue retrival; ", zap.Error(err))
			}
			missingIssues = append(missingIssues, issues...)
			if resp.NextPage == 0 {
				break
			} else {
				opts.ListOptions.Page = resp.NextPage
			}
		}

		pull, _, err := client.PullRequests.List(context.Background(), *owner, *name, &github.PullRequestListOptions{
			ListOptions: github.ListOptions{
				PerPage: 1,
			},
		})
		if err != nil {
			utils.AppLog.Error("newest pull request retrival; ", zap.Error(err))
		}

		if len(pull) > 0 {
			pNewest = pull[0].Number
		} else {

		}

		pDiff := *pNewest - *pOldest
		missingPulls := []*github.PullRequest{}
		for pDiff > 1 {
			opts := github.PullRequestListOptions{
				ListOptions: github.ListOptions{},
			}
			switch {
			case pDiff > 1 && pDiff <= 100:
				opts.ListOptions.PerPage = pDiff
				pDiff = 0
			case pDiff > 100:
				opts.ListOptions.PerPage = 100
				pDiff = pDiff - 100
			}
			pulls, resp, err := client.PullRequests.List(context.Background(), *owner, *name, &opts)
			if err != nil {
				utils.AppLog.Error("newest pull request retrival; ", zap.Error(err))
			}
			missingPulls = append(missingPulls, pulls...)
			if resp.NextPage == 0 {
				break
			} else {
				opts.ListOptions.Page = resp.NextPage
			}
		}

		for j := 0; j < len(missingIssues); j++ {
			missingIssues[j].Repository = repo
		}
		i.Database.BulkInsertIssues(missingIssues)
		i.Database.BulkInsertPullRequests(missingPulls)
	}
	return nil
}