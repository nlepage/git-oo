package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/urfave/cli/v2"

	gitoo "github.com/nlepage/git-oo"
	"github.com/nlepage/git-oo/reflog"
)

var app = &cli.App{
	Name:  "git-oo",
	Usage: "visualize the effect of git commands",
	Action: func(ctx *cli.Context) (err error) {
		var path string
		if ctx.Args().Present() {
			path, err = filepath.Abs(ctx.Args().First())
			if err != nil {
				return err
			}
		} else {
			path, err = os.Getwd()
			if err != nil {
				return err
			}
		}

		gitDir, err := gitoo.LocateGitDir(path)
		if err != nil {
			return err
		}

		events, err := reflog.Watch(ctx.Context, gitDir)
		if err != nil {
			return err
		}

		for entry := range events {
			log.Printf("%#v\n", entry)
		}

		return nil
	},
}

func main() {
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)
	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Fatal(err)
	}
}
