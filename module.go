package main

import (
	"encoding/json"
	"os"

	"github.com/mrnavastar/assist/fs"
)

var modKey Lyra

type Library struct {
	Coordinate string
	Path string
}

type Module struct {
	Name string
	GroupId string
	Java int
	Home string
	Repos []string
	Libraries []Library
}

func (m *Module) Load() error {
	if !fs.Exists("lyra.json") {
		return nil
	}

	data, err := os.ReadFile("lyra.json")
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &m)
}

func (m *Module) Save() error {
	if err := m.Sync(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(m, "", "    ")
	if err != nil {
		return err
	}

	return os.WriteFile("lyra.json", data, os.ModePerm)
}

func (m *Module) Sync() error {
	if err := EnsureJavaInstalled(m.Java); err != nil {
		return err
	}

	for _, library := range m.Libraries {
		artifact, err := ParseArtifact(library.Coordinate)
		if err != nil {
			return err
		}

		err = artifact.Download(m.Home + "/libs", m.Repos)
		if err != nil {
			return err
		}
	}
	return nil
}
