package minecraft

import (
	"github.com/mrnavastar/assist/web"
	"github.com/mrnavastar/lyra/lyra"
	"runtime"
	"strings"
	"time"
)

type VersionType string

const (
	SNAPSHOT VersionType = "snapshot"
	RELEASE  VersionType = "release"
)
const pistonMetaUrl = "https://piston-meta.mojang.com/mc/game/version_manifest_v2.json"

var goos = goosToPistonMetaOs()

type PistonMeta struct {
	Latest struct {
		Release  string `json:"release"`
		Snapshot string `json:"snapshot"`
	} `json:"latest"`
	Versions []struct {
		ID              string    `json:"id"`
		Type            string    `json:"type"`
		URL             string    `json:"url"`
		Time            time.Time `json:"time"`
		ReleaseTime     time.Time `json:"releaseTime"`
		Sha1            string    `json:"sha1"`
		ComplianceLevel int       `json:"complianceLevel"`
	} `json:"versions"`
}

type MinecraftVersion struct {
	Arguments struct {
		Game []interface{} `json:"game"`
		Jvm  []interface{} `json:"jvm"`
	} `json:"arguments"`
	AssetIndex struct {
		ID        string `json:"id"`
		Sha1      string `json:"sha1"`
		Size      int    `json:"size"`
		TotalSize int    `json:"totalSize"`
		URL       string `json:"url"`
	} `json:"assetIndex"`
	Assets          string `json:"assets"`
	ComplianceLevel int    `json:"complianceLevel"`
	Downloads       struct {
		Client struct {
			Sha1 string `json:"sha1"`
			Size int    `json:"size"`
			URL  string `json:"url"`
		} `json:"client"`
		ClientMappings struct {
			Sha1 string `json:"sha1"`
			Size int    `json:"size"`
			URL  string `json:"url"`
		} `json:"client_mappings"`
		Server struct {
			Sha1 string `json:"sha1"`
			Size int    `json:"size"`
			URL  string `json:"url"`
		} `json:"server"`
		ServerMappings struct {
			Sha1 string `json:"sha1"`
			Size int    `json:"size"`
			URL  string `json:"url"`
		} `json:"server_mappings"`
	} `json:"downloads"`
	ID          string `json:"id"`
	JavaVersion struct {
		Component    string `json:"component"`
		MajorVersion int    `json:"majorVersion"`
	} `json:"javaVersion"`
	Libraries []struct {
		Downloads struct {
			Artifact struct {
				Path string `json:"path"`
				Sha1 string `json:"sha1"`
				Size int    `json:"size"`
				URL  string `json:"url"`
			} `json:"artifact"`
		} `json:"downloads"`
		Name  string `json:"name"`
		Rules []struct {
			Action string `json:"action"`
			Os     struct {
				Name string `json:"name"`
			} `json:"os"`
		} `json:"rules,omitempty"`
	} `json:"libraries"`
	Logging struct {
		Client struct {
			Argument string `json:"argument"`
			File     struct {
				ID   string `json:"id"`
				Sha1 string `json:"sha1"`
				Size int    `json:"size"`
				URL  string `json:"url"`
			} `json:"file"`
			Type string `json:"type"`
		} `json:"client"`
	} `json:"logging"`
	MainClass              string    `json:"mainClass"`
	MinimumLauncherVersion int       `json:"minimumLauncherVersion"`
	ReleaseTime            time.Time `json:"releaseTime"`
	Time                   time.Time `json:"time"`
	Type                   string    `json:"type"`
}

// TODO: Cache piston meta endpoints so we don't get rate limited
func GetMinecraftVersion(minecraftVersion string) (version MinecraftVersion, err error) {
	var meta PistonMeta
	if err := web.GetJson(pistonMetaUrl, &meta); err != nil {
		return MinecraftVersion{}, err
	}

	for _, versionMeta := range meta.Versions {
		if versionMeta.ID == minecraftVersion {
			if err := web.GetJson(versionMeta.URL, &version); err != nil {
				return MinecraftVersion{}, err
			}
			break
		}
	}
	return version, nil
}

func goosToPistonMetaOs() string {
	os := runtime.GOOS
	if os == "darwin" {
		return "osx"
	}
	return os
}

// TODO: native libs are handled wrong because mojang uses invalid maven coordiates
func (version MinecraftVersion) GetLibraries() (artifacts []lyra.Artifact) {
	for _, library := range version.Libraries {
		for _, rule := range library.Rules {
			if rule.Action == "allow" && rule.Os.Name != goos {
				goto skip
			}
		}
		{
			artifact := lyra.Dependency.ParseMavenCoordinate(library.Name)
			artifact.Main = library.Downloads.Artifact.URL
			artifacts = append(artifacts, artifact)
		}
	skip:
	}
	return artifacts
}

func (version MinecraftVersion) GetClientArtifact() (artifact lyra.Artifact) {
	artifact.Name = "minecraft-client"
	artifact.Group = "com.mojang"
	artifact.Version = version.ID
	artifact.Main = "minecraft://" + strings.TrimPrefix(version.Downloads.Client.URL, "https://") + "?mojmap=" + version.Downloads.ClientMappings.URL
	return artifact
}

func (version MinecraftVersion) GetServerArtifact() (artifact lyra.Artifact) {
	artifact.Name = "minecraft-server"
	artifact.Group = "com.mojang"
	artifact.Version = version.ID
	artifact.Main = "minecraft://" + strings.TrimPrefix(version.Downloads.Server.URL, "https://") + "?mojmap=" + version.Downloads.ServerMappings.URL
	return artifact
}
