package main

import (
	"encoding/json"
	"errors"
	"os"
	"path"
	"strings"

	"github.com/mrnavastar/assist/fs"
)

var projectKey Lyra

type Project struct {
	Name         string       `json:",omitempty"`
	GroupId      string       `json:",omitempty"`
	Java         int          `json:",omitempty"`
	Home         string       `json:",omitempty"`
	Repos        []string     `json:",omitempty"`
	Dependencies []Dependency `json:",omitempty"`
	Plugins		 []string
}

func (p *Project) Load() error {
	if !fs.Exists("lyra.json") {
		return nil
	}

	data, err := os.ReadFile("lyra.json")
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &p)
}

func (p *Project) Save() error {
	if err := p.Sync(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(p, "", "    ")
	if err != nil {
		return err
	}

	return os.WriteFile("lyra.json", data, os.ModePerm)
}

func (p *Project) Sync() error {
	if err := EnsureJavaInstalled(*p); err != nil {
		return err
	}
	return DownloadDependencies(*p)
}

func (p *Project) GetDependencyByCoordinate(coordinate string) (*Dependency, error) {
	for _, dep := range p.Dependencies {
		if strings.Split(dep.Coordinate, ":")[0] == strings.Split(coordinate, ":")[0] {
			return &dep, nil
		}
	}
	return &Dependency{}, errors.New("no dependency found")
}

func (p *Project) GetDependencyByPath(dependencyPath string) (*Dependency, error) {
	for _, dep := range p.Dependencies {
		if path.Clean(dep.Path) == path.Clean(dependencyPath) {
			return &dep, nil
		}
	}
	return &Dependency{}, errors.New("no dependency found")
}
