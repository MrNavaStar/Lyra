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
	mu     sync.Mutex
	groups map[string]*errgroup.Group

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
	project.mu.Lock()
	defer project.mu.Unlock()
	return project.name
}

func (project *Project) Group() string {
	project.mu.Lock()
	defer project.mu.Unlock()
	return project.groupId
}

func (project *Project) Dependencies() []Artifact {
	project.mu.Lock()
	defer project.mu.Unlock()
	return project.artifacts
}

func (project *Project) Repos() []url.URL {
	project.mu.Lock()
	defer project.mu.Unlock()
	return project.repos
}

func (project *Project) GetClasspath() (classpath []string, err error) {
	for _, artifact := range project.Dependencies() {
		resolved, err := artifact.Resolve()
		if err != nil {
			return nil, err
		}
		classpath = append(classpath, resolved)
	}
	return classpath, nil
}

func (project *Project) GoWith(id string, f func() error) {
	group, ok := project.groups[id]
	if !ok {
		group, _ = errgroup.WithContext(context.Background())
		project.groups[id] = group
	}
	group.Go(f)
}

func (project *Project) Go(f func() error) {
	project.GoWith("", f)
}

func (project *Project) WaitFor(id string) error {
	group, ok := project.groups[id]
	if !ok {
		return nil
	}
	return group.Wait()
}

func (project *Project) Wait() error {
	for _, group := range project.groups {
		if err := group.Wait(); err != nil {
			return err
		}
	}
	return nil
}

func (project *Project) AddRepo(repo url.URL) error {
	if !fs.Exists("lyra.json") {
		return nil
	}

	for _, r := range project.repos {
		if r == repo {
			return nil
		}
	}

	_, err := http.Get(repo.String())
	if err != nil {
		return fmt.Errorf("repo is unreachable: %s", err)
	}

	project.modify(func(project *Project) {
		project.repos = append(project.repos, repo)
	})
	return nil
}

func (project *Project) AddDependency(artifact Artifact) error {
	if !fs.Exists("lyra.json") {
		return nil
	}

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
	project.groups = make(map[string]*errgroup.Group)
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
	if err := project.Wait(); err != nil {
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

	parsed, err := url.Parse("https://repo.maven.apache.org/maven2")
	if err != nil {
		return err
	}
	if err := project.AddRepo(*parsed); err != nil {
		return err
	}

	if err := os.MkdirAll("src/main/resources", os.ModePerm); err != nil {
		return err
	}
	return os.MkdirAll(strings.Join([]string{"src/main/java", strings.ReplaceAll(project.groupId, ".", "/"), project.name}, "/"), os.ModePerm)
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
