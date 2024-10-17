package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/mrnavastar/assist/web"
)

const (
	CorretoURL = "https://corretto.aws/downloads/latest/amazon-corretto-%d-%s-jdk%s"
	CorretoLatest = 23
)

var goToCorreto = map[string]string{
	"linux/amd64": "x64-linux",
	"linux/arm64": "aarch64-linux",
	"darwin/amd64": "x64-macos",
	"darwin/arm64": "aarch64-macos",
	"windows/amd64": "x64-windows",
}

func getCorretoURL(version int) string {
	os := runtime.GOOS
	extension := ".tar.gz"
	if strings.HasPrefix(os, "windows") {
		extension = ".zip"
	}
	return fmt.Sprintf(CorretoURL, version, goToCorreto[os], extension)
}

func EnsureJavaInstalled(version int) error {
	cache, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	return web.Download(cache + "/java/corretto/" + string(version) + "/", "corretto-" + string(version), getCorretoURL(version))
}
