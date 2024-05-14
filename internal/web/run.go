package web

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/bigredeye/notmanytask/internal/config"
	"github.com/bigredeye/notmanytask/internal/database"
	"github.com/bigredeye/notmanytask/internal/deadlines"
	"github.com/bigredeye/notmanytask/internal/gitlab"
	"github.com/bigredeye/notmanytask/internal/scorer"
	"github.com/bigredeye/notmanytask/internal/tgbot"
	conf "github.com/bigredeye/notmanytask/pkg/conf"
	zlog "github.com/bigredeye/notmanytask/pkg/log"
	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
)

func RunInternal() error {
	flag.Parse()
	config, err := config.ParseConfig()
	if err != nil {
		return err
	}
	log.Printf("Parsed config: %+v", config)

	logger, err := zlog.Init(config.Log)
	if err != nil {
		return errors.Wrap(err, "Failed to init logger")
	}
	defer func() {
		err = zlog.Sync()
	}()

	wg := sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := database.OpenDataBase(logger.Named("database"), fmt.Sprintf("postgresql://%s:%s@%s:%d/%s",
		config.DataBase.User,
		config.DataBase.Pass,
		config.DataBase.Host,
		config.DataBase.Port,
		config.DataBase.Name,
	))
	if err != nil {
		return errors.Wrap(err, "Failed to open database")
	}

	bot, err := tgbot.NewBot(config, logger.Named("tgbot"), db)
	if err != nil {
		return errors.Wrap(err, "failed to create telegram bot")
	}

	deadlines, err := deadlines.NewFetcher(config, logger.Named("deadlines.fetcher"))
	if err != nil {
		return errors.Wrap(err, "Failed to create deadlines fetcher")
	}

	git, err := gitlab.NewClient(config, logger.Named("gitlab"))
	if err != nil {
		return errors.Wrap(err, "Failed to create gitlab client")
	}

	projects, err := gitlab.NewProjectsMaker(git, db)
	if err != nil {
		return errors.Wrap(err, "Failed to create projects maker")
	}

	pipelines, err := gitlab.NewPipelinesFetcher(git, db)
	if err != nil {
		return errors.Wrap(err, "Failed to create projects maker")
	}

	scorer := scorer.NewScorer(db, deadlines, git)

	wg.Add(5)
	go func() {
		defer wg.Done()
		deadlines.Run(ctx)
	}()
	go func() {
		defer wg.Done()
		projects.Run(ctx)
	}()
	go func() {
		defer wg.Done()
		pipelines.Run(ctx)
	}()
	go func() {
		defer wg.Done()
		pipelines.RunFresh(ctx)
	}()
	go func() {
		defer wg.Done()
		bot.Run(ctx)
	}()

	s := newServer(config, logger.Named("server"), db, deadlines, projects, pipelines, scorer, git)
	return errors.Wrap(s.run(), "Server failed")
}

func Run() {

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Error creating watcher: %v", err)
	}
	defer watcher.Close()

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("Config file modified, reloading...")
					if err != nil {
						log.Printf("Error reloading config: %v", err)
					} else {
						go RunInternal()
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("Error: %v", err)
			}
		}
	}()

	err = watcher.Add(conf.ConfigPath)
	if err != nil {
		log.Fatalf("Error adding file to watcher: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
}
