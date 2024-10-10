package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/mrnavastar/assist/web"
)

type Artifact struct {
	GroupId         string
	Id              string
	Version         string
	Extension       string
	Classifier      string
	IsSnapshot      bool
	SnapshotVersion string
}

type metadata struct {
	Timestamp   string `xml:"versioning>snapshot>timestamp"`
	BuildNumber string `xml:"versioning>snapshot>buildNumber"`
	Latest		string `xml:"versioning>latest"`
}

func ParseArtifact(coordinate string) (Artifact, error) {
	parts := strings.Split(coordinate, ":")
	artifact := Artifact{}
	l := len(parts)
	if l >= 2 {
		artifact.GroupId = parts[0]
		artifact.Id = parts[1]

		if l > 2 {
			artifact.Version = parts[l-1]
		}
		if l > 3 {
			artifact.Extension = parts[2]
		}
		if l > 4 {
			artifact.Classifier = parts[3]
		}
		if strings.HasSuffix(artifact.Version, "-SNAPSHOT") {
			artifact.IsSnapshot = true
			artifact.Version = strings.Trim(artifact.Version, "-SNAPSHOT")
		}
		return artifact, nil
	}
	return artifact, fmt.Errorf("invalid package coordinate: %s Try groupId:artifactId:version", coordinate)
}

func (a Artifact) Equals(artifact Artifact) bool {
	return a.GroupId == artifact.GroupId && a.Id == artifact.Id
}

func (a Artifact) Filename() string {
	ext := "jar"
	if a.Extension != "" {
		ext = a.Extension
	}
	v := a.Version
	if a.IsSnapshot {
		if a.SnapshotVersion != "" {
			v += "-" + a.SnapshotVersion
		} else {
			v += "-SNAPSHOT"
		}
	}
	if a.Classifier != "" {
		return fmt.Sprintf("%s-%s-%s.%s", a.Id, v, a.Classifier, ext)
	} else {
		return fmt.Sprintf("%s-%s.%s", a.Id, v, ext)
	}
}

func (a Artifact) ArtifactUrl(repo string) (string, error) {
	if a.IsSnapshot {
		var err error
		a.SnapshotVersion, err = a.LatestSnapshotVersion(repo)
		if err != nil {
			return "", err
		}
	}
	return repo + a.ArtifactPath(), nil
}

func (a Artifact) LatestVersion(repos []string) (string, error) {
	for _, repo := range repos {	
		metadataUrl := repo + a.GroupPath() + "maven-metadata.xml"
		resp, err := http.Get(metadataUrl)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		m := metadata{}
		if err := xml.NewDecoder(resp.Body).Decode(&m); err != nil {
			return "", err
		}
		return m.Latest, nil
	}
	return "", errors.New("failed to find artifact on any of the provided repos")
}

func (a Artifact) LatestSnapshotVersion(repo string) (string, error) {
	metadataUrl := repo + a.GroupPath() + "maven-metadata.xml"
	resp, err := http.Get(metadataUrl)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	m := metadata{}
	if err := xml.NewDecoder(resp.Body).Decode(&m); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%s", m.Timestamp, m.BuildNumber), nil
}

func (a Artifact) ArtifactPath() string {
	return a.GroupPath() + "/" + a.Filename()
}

func (a Artifact) GroupPath() string {
	parts := append(strings.Split(a.GroupId, "."), a.Id)
	if a.IsSnapshot {
		return strings.Join(append(parts, a.Version+"-SNAPSHOT"), "/")
	} else {
		return strings.Join(append(parts, a.Version), "/")
	}
}

func (a Artifact) download(path string, repo string) (string, error) {
	return path + "/" + a.ArtifactPath(), web.Download(path+"/"+a.GroupPath(), a.Filename(), repo)
}

func (a Artifact) TryDownload(path string, repos []string) (string, error) {
	for _, repo := range repos {
		if file, err := a.download(path, repo); err == nil {
			return file, nil
		}
	}
	return "", fmt.Errorf("failed to download maven artifact: %s")
}
