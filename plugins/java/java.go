package java

import (
	"context"
	"fmt"
	"github.com/codeclysm/extract"
	"os"
	"path"
	"runtime"
	"strings"
)

const (
	CorrettoURL    = "https://corretto.aws/downloads/latest/amazon-corretto-%d-%s-jdk%s"
	CorrettoLatest = 23
)

var goToCorretto = map[string]string{
	"linux/amd64":   "x64-linux",
	"linux/arm64":   "aarch64-linux",
	"darwin/amd64":  "x64-macos",
	"darwin/arm64":  "aarch64-macos",
	"windows/amd64": "x64-windows",
}

func getExtension() string {
	os := runtime.GOOS
	extension := ".tar.gz"
	if strings.HasPrefix(os, "windows") {
		extension = ".zip"
	}
	return extension
}

func decompress(file string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}

	renamer := func(s string) string {
		p := strings.Split(s, string(os.PathSeparator))
		p[0] = "corretto"
		return strings.Join(p, string(os.PathSeparator))
	}

	if getExtension() == ".zip" {
		return extract.Zip(context.Background(), f, path.Dir(file), renamer)
	}
	return extract.Gz(context.Background(), f, path.Dir(file), renamer)
}

func getCorretoURL(version int) string {
	return fmt.Sprintf(CorrettoURL, version, goToCorretto[runtime.GOOS+"/"+runtime.GOARCH], getExtension())
}
