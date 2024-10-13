package main

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/mrnavastar/assist/fs"
)

var modKey Lyra

type Module struct {
	Name string
	GroupId string
	Java string
	Repos []string
	Artifacts []Artifact
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
	data, err := json.MarshalIndent(m, "", "    ")
	if err != nil {
		return err
	}

	err = os.WriteFile("lyra.json", data, os.ModePerm)
	if err != nil {
		return err
	}
	return m.Sync()
}

func (m Module) Sync() error {
	cache, err := os.UserCacheDir()
	if err != nil {
		return err
	}

	path := strings.Join([]string{cache, "lyra", "libs"}, "/")
	for _, artifact := range m.Artifacts {
		err := artifact.Download(path, m.Repos)
		if err != nil {
			return err
		}
	}
	return nil
}
