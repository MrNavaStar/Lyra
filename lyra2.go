package main

import (
	"context"
	"errors"
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

type Lyra int

func init_project(ctx *cli.Context) error {
	if ctx.Args().Len() == 0 {
		return errors.New("no project name provided")
	}

	mod := ctx.Context.Value(modKey).(Module)
	mod.Name = ctx.Args().First()
	mod.Repos = append(mod.Repos, "https://repo.maven.apache.org/maven2/")

	err := os.MkdirAll("src/main/java/" + mod.Name, os.ModePerm)
	if err != nil {
		return err
	}
	return mod.Save()
}

func main() {
	var mod Module
	err := mod.Load()
	if err != nil {
		log.Fatal(err)
	}

	app := &cli.App{
        Name:  "lyra",
		Commands: []*cli.Command{
			{
				Name: "init",
				Action: init_project,
			},
			{
				Name: "get",
				Action: Get,
			},
			{
				Name: "build",
				Action: Build,
			},
			{
				Name: "repo",
				Action: AddRepo,
				Subcommands: []*cli.Command{
					{
						Name: "remove",
					},
					{
						Name: "list",
					},
				},
			},
		},
    }

    if err := app.RunContext(context.WithValue(context.Background(), modKey, mod), os.Args); err != nil {
        log.Fatal(err)
    }
}
