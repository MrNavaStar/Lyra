package main

import (
	"errors"
	"net/http"
	"strings"

	"github.com/mrnavastar/assist/ops"
	"github.com/urfave/cli/v2"
)

func Get(ctx *cli.Context) error {
	mod := ctx.Context.Value(modKey).(Module)
	if ops.IsEmpty(mod) {
		return errors.New("no project found in current directory")
	}

	if ctx.Args().Len() == 0 {
		return errors.New("no maven artifact provided")
	}

	artifact, err := ParseArtifact(ctx.Args().First())
	if err != nil {
		return err
	}

	if artifact.Version == "" {
		artifact.Version, err = artifact.LatestVersion(mod.Repos)
		if err != nil {
			return err
		}
	}

	for _, a := range mod.Artifacts {
		if a.Equals(artifact) {
			return mod.Save()
		}
	}

	mod.Artifacts = append(mod.Artifacts, artifact)
	return mod.Save()
}

func AddRepo(ctx *cli.Context) error {
	mod := ctx.Context.Value(modKey).(Module)
	if ctx.Args().Len() == 0 {
		return errors.New("no repo provided")
	}

	repo := ctx.Args().First()
	if !strings.HasPrefix(repo, "https://") {
		repo = "https://" + repo
	}

	if !strings.HasSuffix(repo, "/") {
		repo += "/"
	}

	for _, r := range mod.Repos {
		if r == repo {
			return nil
		}
	}

	resp, err := http.Get(repo)
	if err != nil {
		return err
	} else if resp.StatusCode != http.StatusOK {
		return errors.New("repo is unreachable")
	}

	mod.Repos = append(mod.Repos, repo)
	return mod.Save()
}