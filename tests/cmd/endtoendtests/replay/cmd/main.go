package main

import (
	"core/pipeline/backend"
	"core/pipeline/ingestor"
	"core/tests/cmd/endtoendtests/replay"
	"flag"
	"fmt"
	"github.com/google/go-github/github"
	"net/url"
	"time"
)

func main() {
	runBacktestFlag := flag.Bool("runbacktest", false, "runs the end to end backtest")
	loadArchiveFlag := flag.Bool("loadarchive", false, "load archive into the database")
	archivePathFlag := flag.String("archivepath", "", "location of github archive")
	flag.Parse()

	if !*runBacktestFlag && !*loadArchiveFlag {
		fmt.Println("Usage: ./cmd --loadarchive=true --archivepath=/home/michael/Data/GithubArchive/")
		fmt.Println("Usage: ./cmd --runbacktest=true")
		return
	}

	//bufferPool := ingestor.NewPool()
	db := ingestor.Database{} //{BufferPool: bufferPool}
	db.Open()

	dispatcher := ingestor.Dispatcher{}
	dispatcher.Start(5)
	ingestorServer := ingestor.IngestorServer{}
	go ingestorServer.Start()

	bs := replay.BacktestServer{DB: &db, Ingestor: &ingestorServer}
	go bs.Start()

	if *loadArchiveFlag && *archivePathFlag != "" {
		bs.LoadArchive(*archivePathFlag)
	}

	if *runBacktestFlag {
		backendServer := backend.BackendServer{}
		client := github.NewClient(nil)
		url, _ := url.Parse("http://localhost:8000/")
		client.BaseURL = url
		client.UploadURL = url

		repos, err := db.ReadBacktestRepos()
		if err != nil {
			panic(err)
		}
		backendServer.Repos = new(backend.ActiveRepos)
		backendServer.Repos.Actives = make(map[int]*backend.ArchRepo)

		for i := 0; i < len(repos); i++ {
			repo := repos[i]
			backendServer.Repos.Actives[*repo.ID] = new(backend.ArchRepo)
			backendServer.Repos.Actives[*repo.ID].Hive = new(backend.ArchHive)
			backendServer.Repos.Actives[*repo.ID].Hive.Blender = new(backend.Blender)
			backendServer.NewModel(*repo.ID)
			backendServer.Repos.Actives[*repo.ID].Client = client
		}

		for i := 0; i < len(repos); i++ {
			repo := repos[i]
			bs.AddRepo(*repo.ID, *repo.Organization.Name, *repo.Name)
		}

		fmt.Println("bs.StreamWebhookEvents() start")
		bs.StreamWebhookEvents()
		//go backendServer.Start()
		fmt.Println("bs.StreamWebhookEvents() end")

		time.Sleep(15 * time.Second)
		bs.PredictionAccuracy()
		time.Sleep(15 * time.Second)
	}
}
