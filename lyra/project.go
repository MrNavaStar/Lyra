package lyra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/urfave/cli/v2"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/mrnavastar/assist/fs"
	"golang.org/x/sync/errgroup"
)

type Project struct {
	mu    sync.Mutex
	group *errgroup.Group

	name      string
	groupId   string
	repos     []url.URL
	artifacts []Artifact
	plugins   []string
}

type projectProxy struct {
	Name      string     `json:",omitempty"`
	Group     string     `json:",omitempty"`
	Artifacts []Artifact `json:",omitempty"`
}

func (project *Project) modify(modifier func(*Project)) {
	project.mu.Lock()
	defer project.mu.Unlock()
	modifier(project)
}

func (project *Project) Name() string {
	return project.name
}

func (project *Project) Group() string {
	return project.groupId
}

func (project *Project) Dependencies() []Artifact {
	return project.artifacts
}

func (project *Project) Repos() []string {

}

func (project *Project) GetClasspath() (classpath []string, err error) {
	for _, artifact := range project.artifacts {
		resolved, err := artifact.Resolve()
		if err != nil {
			return nil, err
		}
		classpath = append(classpath, resolved)
	}
	return classpath, nil
}

func (project *Project) Go(f func() error) {
	project.group.Go(f)
}

func (project *Project) AddRepo(repo url.URL) error {
	for _, r := range project.repos {
		if r == repo {
			return nil
		}
	}

	_, err := http.Get(repo)
	if err != nil {
		return fmt.Errorf("repo is unreachable: %s", err)
	}

	project.modify(func(project *Project) {
		project.repos = append(project.repos, repo)
	})
	return nil
}

func (project *Project) AddDependency(artifact Artifact) error {
	index := -1
	for i, existingArtifact := range project.artifacts {
		if existingArtifact.SameAs(artifact) {
			index = i
			break
		}
	}
	_, err := artifact.Resolve()
	if err != nil {
		return err
	}
	_, err = artifact.ResolveSources()
	if err != nil {
		return err
	}
	_, err = artifact.ResolveDocs()
	if err != nil {
		return err
	}
	project.modify(func(project *Project) {
		if index == -1 {
			project.artifacts = append(project.artifacts, artifact)
			return
		}
		project.artifacts[index] = artifact
	})
	return nil
}

func (project *Project) Load() error {
	project.group, _ = errgroup.WithContext(context.Background())
	if !fs.Exists("lyra.json") {
		return nil
	}
	data, err := os.ReadFile("lyra.json")
	if err != nil {
		return err
	}
	proxy := projectProxy{}
	if err := json.Unmarshal(data, &proxy); err != nil {
		return err
	}
	project.name = proxy.Name
	project.groupId = proxy.Group
	project.artifacts = proxy.Artifacts
	return nil
}

func (project *Project) Save() error {
	if err := project.group.Wait(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(projectProxy{
		Name:      project.name,
		Group:     project.groupId,
		Artifacts: project.artifacts,
	}, "", "    ")
	if err != nil {
		return err
	}

	if len(data) <= 2 {
		return nil
	}
	return os.WriteFile("lyra.json", data, os.ModePerm)
}

func init() {
	Command.RegisterMany([]*cli.Command{
		{
			Name:   "init",
			Args:   true,
			Action: initProject,
		},
		{
			Name: "repo",
			Subcommands: []*cli.Command{
				{
					Name:   "add",
					Args:   true,
					Action: addRepo,
				},
			},
		},
		{
			Name:   "classpath",
			Args:   false,
			Action: showClasspath,
		},
	})
}

func initProject(ctx *cli.Context) error {
	if ctx.Args().Len() == 0 {
		return errors.New("no project name provided")
	}
	if fs.Exists("lyra.json") {
		return nil
	}
	project := GetCurrentProject()
	project.name = ctx.Args().First()
	project.groupId = ctx.String("group")
	project.repos = append(project.repos, "https://repo.maven.apache.org/maven2")

	return os.MkdirAll(strings.Join([]string{"src/main/java", strings.ReplaceAll(project.groupId, ".", "/"), project.name}, "/"), os.ModePerm)
}

func addRepo(ctx *cli.Context) error {
	if !fs.Exists("lyra.json") {
		return errors.New("no project in current directory")
	}
	if ctx.Args().Len() == 0 {
		return errors.New("no repo provided")
	}
	return GetCurrentProject().AddRepo(ctx.Args().First())
}

func showClasspath(ctx *cli.Context) error {
	if !fs.Exists("lyra.json") {
		return nil
	}
	classpath, err := GetCurrentProject().GetClasspath()
	if err != nil {
		return err
	}
	println(strings.Join(classpath, ";"))
	return nil
}
