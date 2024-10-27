package main

import (
	"context"
	"errors"
	"log"
	"os"
	"path"
	"strings"

	"github.com/mrnavastar/assist/fs"
	"github.com/urfave/cli/v2"
)

type Lyra int

func init_project(ctx *cli.Context) error {
	if ctx.Args().Len() == 0 {
		return errors.New("no project name provided")
	}

	if fs.Exists("lyra.json") {
		return nil
	}

	cache, err := os.UserCacheDir()
	if err != nil {
		return err
	}

	mod := ctx.Context.Value(projectKey).(Project)
	mod.Name = ctx.Args().First()
	mod.GroupId = ctx.String("group")
	mod.Java = ctx.Int("java")
	mod.Home = path.Join(cache, "lyra")
	mod.Repos = append(mod.Repos, "https://repo.maven.apache.org/maven2")

	if err := os.MkdirAll(strings.Join([]string{"src/main/java", strings.ReplaceAll(mod.GroupId, ".", "/"), mod.Name}, "/"), os.ModePerm); err != nil {
		return err
	}
	return mod.Save()
}

func main() {
	var mod Project
	err := mod.Load()
	if err != nil {
		log.Fatal(err)
	}

	if err := LoadPlugin(context.Background(), "plugins/mvn"); err != nil {
		log.Fatal(err)
	}

	app := &cli.App{
		Name: "lyra",
		Commands: []*cli.Command{
			{
				Name:   "init",
				Action: init_project,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "group",
						Aliases: []string{"g"},
					},
					&cli.IntFlag{
						Name:    "java",
						Aliases: []string{"j"},
						Value:   CorrettoLatest,
					},
				},
			},
			{
				Name:   "get",
				Action: GetDependencyFromCLI,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "include",
						Aliases: []string{"i"},
					},
				},
			},
			{
				Name:   "build",
				Action: Build,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "fat",
						Aliases: []string{"f"},
					},
					&cli.BoolFlag{
						Name:    "minimize",
						Aliases: []string{"m"},
					},
					&cli.BoolFlag{
						Name:    "sources",
						Aliases: []string{"s"},
					},
					&cli.BoolFlag{
						Name:    "docs",
						Aliases: []string{"d"},
					},
				},
			},
			{
				Name: "run",
			},
			{
				Name:   "repo",
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

	if err := app.RunContext(context.WithValue(context.Background(), projectKey, mod), os.Args); err != nil {
		log.Fatal(err)
	}
}
