package mvn

import (
	"encoding/xml"
	"errors"
	"github.com/mrnavastar/lyra/lyra"
	"net/http"
	"net/url"
	"regexp"
)

type meta struct {
	XMLName    xml.Name `xml:"metadata"`
	GroupId    string   `xml:"groupId"`
	ArtifactId string   `xml:"artifactId"`
	Versioning struct {
		Latest  string `xml:"latest"`
		Release string `xml:"release"`
	} `xml:"versioning"`
}

// TODO : I think the dep struct is wrong - maybe also versions in the meta data
type pom struct {
	XMLName      xml.Name `xml:"project"`
	GroupId      string   `xml:"groupId"`
	ArtifactId   string   `xml:"artifactId"`
	Version      string   `xml:"version"`
	Dependencies struct {
		Dependency []struct {
			GroupId    string `xml:"groupId"`
			ArtifactId string `xml:"artifactId"`
			Version    string `xml:"version"`
			Scope      string `xml:"scope"`
		} `xml:"dependency"`
	} `xml:"dependencies"`
}

var pattern = regexp.MustCompile("([^: ]+):([^: ]+)(:([^: ]*)(:([^: ]+))?)?:([^: ]+)")

func init() {
	lyra.Dependency.RegisterParser(mvnParser)
}

func getMeta(repo url.URL, artifact lyra.Artifact) (metaData meta, err error) {
	metaUrl := repo.JoinPath(artifact.Group, artifact.Name).JoinPath("maven-metadata.xml")

	response, err := http.Get(metaUrl.String())
	if err != nil {
		return meta{}, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return meta{}, errors.New("not a valid maven repo: " + response.Status)
	}

	err = xml.NewDecoder(response.Body).Decode(&metaData)
	return
}

func getPom(repo url.URL, artifact lyra.Artifact) (pomData pom, err error) {
	pomUrl := repo.JoinPath(artifact.Group, artifact.Name, artifact.Version, artifact.Name+"-"+artifact.Version, ".pom")

	response, err := http.Get(pomUrl.String())
	if err != nil {
		return pom{}, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return pom{}, errors.New("not a valid maven repo: " + response.Status)
	}

	err = xml.NewDecoder(response.Body).Decode(&pomData)
	return
}

func mvnParser(slug string) (artifact lyra.Artifact, err error) {
	groups := pattern.FindStringSubmatch(slug)
	artifact.Group = groups[1]
	artifact.Name = groups[2]
	artifact.Version = groups[7]
	return loadFromMaven(artifact)
}

func loadFromMaven(artifact lyra.Artifact) (lyra.Artifact, error) {
	var repos []url.URL
	var failed []url.URL
	for _, repo := range repos {
		// Find latest version if it is not present
		if len(artifact.Version) == 0 {
			metaData, err := getMeta(repo, artifact)
			if err != nil {
				failed = append(failed, repo)
				continue
			}
			artifact.Version = metaData.Versioning.Latest
		}

		pomData, err := getPom(repo, artifact)
		if err != nil {
			failed = append(failed, repo)
			continue
		}

		base := repo.JoinPath(artifact.Group, artifact.Name, artifact.Version)
		jar := base.JoinPath(artifact.Name + "-" + artifact.Version + ".jar")
		sources := base.JoinPath(artifact.Name + "-" + artifact.Version + "-sources.jar")
		docs := base.JoinPath(artifact.Name + "-" + artifact.Version + "-javadocs.jar")

		if lyra.PingResource(jar) {
			artifact.Main = jar.String()
		}
		if lyra.PingResource(sources) {
			artifact.Main = sources.String()
		}
		if lyra.PingResource(docs) {
			artifact.Docs = docs.String()
		}

		cmd := []string{"get"}
		for _, dependency := range pomData.Dependencies.Dependency {
			var indirect lyra.Artifact
			indirect.Name = dependency.ArtifactId
			indirect.Group = artifact.Group
			indirect.Version = artifact.Version

			cmd = append(cmd, dependency.GroupId+":"+dependency.ArtifactId+":"+dependency.Version)
			artifact.Dependencies = append(artifact.Dependencies, indirect)
		}

		if err := lyra.Command.Run(cmd...); err != nil {
			return artifact, nil
		}
		break
	}
	return artifact, nil
}
