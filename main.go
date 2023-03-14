package main

import (
	"errors"
	"log"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/urfave/cli/v2"

	"github.com/nlepage/gitoo/reflog"
)

var app = &cli.App{
	Name:  "gitoo",
	Usage: "visualize the effect of git commands",
	Action: func(ctx *cli.Context) (err error) {
		path, err := locateRepository(ctx)
		if err != nil {
			return err
		}

		watcher, err := reflog.NewWatcher(filepath.Join(path, git.GitDirName))
		if err != nil {
			log.Fatalln(err)
		}
		defer func() {
			err = errors.Join(err, watcher.Close())
		}()

		for entry := range watcher.Entries() {
			log.Printf("%#v\n", entry)
		}

		return nil
	},
}

func main() {
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func locateRepository(ctx *cli.Context) (path string, err error) {
	if ctx.Args().Present() {
		path, err = filepath.Abs(ctx.Args().First())
		if err != nil {
			return
		}
	} else {
		path, err = os.Getwd()
		if err != nil {
			return
		}
	}

	for {
		_, err = os.Stat(filepath.Join(path, git.GitDirName))
		if err == nil {
			return
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}

		dir := filepath.Dir(path)
		if dir == path {
			return "", errors.New("not in a git directory")
		}
		path = dir
	}
}
