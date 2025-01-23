package minecraft

import (
	"fmt"
	"github.com/mrnavastar/assist/fs"
	"github.com/mrnavastar/assist/web"
	"github.com/mrnavastar/lyra/lyra"
	"net/url"
	"path"
	"strings"
)

const tinyRemapperVersion = "0.10.4"

var trJar = lyra.Artifact{
	Name:    "tiny-remapper",
	Group:   "net.fabricmc",
	Version: tinyRemapperVersion,
	Main:    fmt.Sprintf("https://maven.fabricmc.net/net/fabricmc/tiny-remapper/%s/tiny-remapper-%s-fat.jar", tinyRemapperVersion, tinyRemapperVersion),
	Sources: fmt.Sprintf("https://maven.fabricmc.net/net/fabricmc/tiny-remapper/%s/tiny-remapper-%s-sources.jar", tinyRemapperVersion, tinyRemapperVersion),
}

func init() {
	lyra.Dependency.RegisterResolver("minecraft", resolveMinecraft)
}

func RemapJar(jarPath string, mappings string, forwards bool) error {
	trPath, err := trJar.Resolve()
	if err != nil {
		return err
	}

	args := []string{jarPath, strings.Replace(jarPath, ".jar", "-remapped.jar", 1), mappings}
	if forwards {
		args = append(args, "target", "source")
	} else {
		args = append(args, "source", "target")
	}

	err = lyra.Java.Run(lyra.JavaRunOptions{
		ProgramArgs: args,
		Jar:         trPath,
	})
	if err != nil {
		return err
	}
	return nil
}

func resolveMinecraft(uri *url.URL) (string, error) {
	cache, err := lyra.GetCache()
	if err != nil {
		return "", err
	}

	// Download Mojang mappings
	mojmapUrl, err := url.Parse(uri.Query().Get("mojmap"))
	if err != nil {
		return "", err
	}
	mojmapPath := path.Join(cache, "minecraft", mojmapUrl.Path)
	if err := web.Download(mojmapPath, mojmapUrl.String()); err != nil {
		return "", err
	}

	minecraftJar := path.Join(cache, "minecraft", uri.Path)
	remappedJar := strings.Replace(minecraftJar, ".jar", "-remapped.jar", 1)
	if fs.Exists(remappedJar) {
		return "file://" + remappedJar, nil
	}

	if err := web.Download(minecraftJar, strings.Replace(uri.String(), "minecraft", "https", 1)); err != nil {
		return "", err
	}
	if err := RemapJar(minecraftJar, mojmapPath, true); err != nil {
		return "", err
	}
	return "file://" + remappedJar, nil
}
