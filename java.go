package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"

	"github.com/codeclysm/extract"
	"github.com/mrnavastar/assist/fs"
	"github.com/mrnavastar/assist/web"
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

func ensureJavaInstalled(mod Project) error {
	if fs.Exists(path.Join(mod.Home, "java", strconv.Itoa(mod.Java), "corretto/bin/java")) {
		return nil
	}

	java := path.Join(mod.Home, "java", strconv.Itoa(mod.Java), "corretto"+getExtension())

	if err := web.Download(java, getCorretoURL(mod.Java)); err != nil {
		return err
	}

	if err := decompress(java); err != nil {
		return err
	}

	os.Remove(java)
	return nil
}

func JavaCompile(mod Project, classpath []string, sourcepath []string) error {
	if err := ensureJavaInstalled(mod); err != nil {
		return err
	}

	cmd := exec.Command(path.Join(mod.Home, "java", strconv.Itoa(mod.Java), "corretto/bin/javac"),
		"-d", "build/output",
		"-cp", strings.Join(classpath, string(os.PathListSeparator)),
		strings.Join(sourcepath, " "))

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func JavaRun(mod Project, classpath []string, jar string) error {
	if err := ensureJavaInstalled(mod); err != nil {
		return err
	}

	cmd := exec.Command(path.Join(mod.Home, "java", strconv.Itoa(mod.Java), "corretto/bin/java"),
		"-cp", strings.Join(classpath, string(os.PathListSeparator)),
		"-jar", jar)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
