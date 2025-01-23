package minecraft

import (
	"github.com/mrnavastar/assist/web"
	"github.com/mrnavastar/lyra/lyra"
	"path"
	"strings"
)

const fabricMetaUrl = "https://meta.fabricmc.net/"
const fabricMavenUrl = "https://maven.fabricmc.net/"

type FabricLoaderMeta []struct {
	Separator string `json:"separator"`
	Build     int    `json:"build"`
	Maven     string `json:"maven"`
	Version   string `json:"version"`
	Stable    bool   `json:"stable"`
}

type fabricLibrary struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Md5    string `json:"md5"`
	Sha1   string `json:"sha1"`
	Sha256 string `json:"sha256"`
	Sha512 string `json:"sha512"`
	Size   int    `json:"size"`
}

type FabricLoaderVersion struct {
	Loader struct {
		Separator string `json:"separator"`
		Build     int    `json:"build"`
		Maven     string `json:"maven"`
		Version   string `json:"version"`
		Stable    bool   `json:"stable"`
	} `json:"loader"`
	Intermediary struct {
		Maven   string `json:"maven"`
		Version string `json:"version"`
		Stable  bool   `json:"stable"`
	} `json:"intermediary"`
	LauncherMeta struct {
		Version        int `json:"version"`
		MinJavaVersion int `json:"min_java_version"`
		Libraries      struct {
			Client      []fabricLibrary `json:"client"`
			Common      []fabricLibrary `json:"common"`
			Server      []fabricLibrary `json:"server"`
			Development []fabricLibrary `json:"development"`
		} `json:"libraries"`
		MainClass struct {
			Client string `json:"client"`
			Server string `json:"server"`
		} `json:"mainClass"`
	} `json:"launcherMeta"`
}

func GetLatestFabricVersion(version MinecraftVersion) (fabricLoaderMetaVersion FabricLoaderVersion, err error) {
	var fabricLoaderMeta FabricLoaderMeta
	if err := web.GetJson(path.Join(fabricMetaUrl, "v2/versions/loader"), &fabricLoaderMeta); err != nil {
		return FabricLoaderVersion{}, err
	}

	if err := web.GetJson(path.Join(fabricMetaUrl, "v2/versions/loader", version.ID, fabricLoaderMeta[0].Version), &fabricLoaderMetaVersion); err != nil {
		return FabricLoaderVersion{}, err
	}
	return fabricLoaderMetaVersion, nil
}

func convertFabricLibraries(libs []fabricLibrary) (artifacts []lyra.Artifact) {
	for _, library := range libs {
		artifact := lyra.Dependency.ParseMavenCoordinate(library.Name)
		artifact.Main = path.Join(library.URL, path.Join(strings.Split(artifact.Group, ".")...), artifact.Name, artifact.Version, artifact.Name+"-"+artifact.Version+".jar")
		artifacts = append(artifacts, artifact)
	}
	return artifacts
}

func (version FabricLoaderVersion) GetClientLibraries() (artifacts []lyra.Artifact) {
	return convertFabricLibraries(version.LauncherMeta.Libraries.Client)
}

func (version FabricLoaderVersion) GetCommonLibraries() (artifacts []lyra.Artifact) {
	return convertFabricLibraries(version.LauncherMeta.Libraries.Common)
}

func (version FabricLoaderVersion) GetServerLibraries() (artifacts []lyra.Artifact) {
	return convertFabricLibraries(version.LauncherMeta.Libraries.Server)
}

func (version FabricLoaderVersion) GetDevLibraries() (artifacts []lyra.Artifact) {
	return convertFabricLibraries(version.LauncherMeta.Libraries.Development)
}

func (version FabricLoaderVersion) GetLibraries() (artifacts []lyra.Artifact) {
	artifacts = append(artifacts, version.GetClientLibraries()...)
	artifacts = append(artifacts, version.GetCommonLibraries()...)
	artifacts = append(artifacts, version.GetServerLibraries()...)
	artifacts = append(artifacts, version.GetDevLibraries()...)
	return artifacts
}

func (version FabricLoaderVersion) GetLoader() lyra.Artifact {
	artifact := lyra.Dependency.ParseMavenCoordinate(version.Loader.Maven)
	artifact.Main = path.Join(fabricMavenUrl, path.Join(strings.Split(artifact.Group, ".")...), artifact.Name, artifact.Version, artifact.Name+"-"+artifact.Version+".jar")
	artifact.Sources = strings.Replace(artifact.Main, ".jar", "-sources.jar", 1)
	artifact.Docs = strings.Replace(artifact.Main, ".jar", "-javadoc.jar", 1)
	return artifact
}
