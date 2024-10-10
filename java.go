package main

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"
)

func Build(ctx *cli.Context) error {
	mod := ctx.Context.Value(modKey).(Module)
	mod.Sync()

	// Create Classpath
	var cp []string
	for _, artifact := range mod.Artifacts {
		cp = append(cp, artifact.ArtifactPath())
	}

	// Create Sourcepath
	var sp []string
	filepath.WalkDir("src/main", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		if strings.HasSuffix(path, ".java") {
			sp = append(sp, path)
		}
		return nil
	})

	cmd := exec.Command("javac", 
		"-d", "build/output",
		"-cp", strings.Join(cp, ";"),
		strings.Join(sp, " "))

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return err
	}

	return Package(mod)
}

func Package(mod Module) error {
	cmd := exec.Command("jar", 
		"--create", mod.Name + ".jar",
		"-C", "build/output/",
		"--main-class", "package.MainClass", 
		"*.class")
	
	println(cmd.String())

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
