package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mrnavastar/assist/bytes"
	"github.com/mrnavastar/babe/babe"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
)

const manifest = "Manifest-Version: 1.0\nMain-Class: %s\n";

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
	c, group := babe.CreateJar(mod.Name + ".jar")

	var mainClass string

	walkGroup, _ := errgroup.WithContext(context.Background())

	err := filepath.WalkDir("build/output/", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		walkGroup.Go(func() error {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			member := babe.JarMember{Name: strings.TrimPrefix(path, "build/output/"), Buffer: &bytes.Buffer{Data: &data, Index: 0}}
			c <- &member
			class, err := member.GetAsClass()
			if err != nil {
				return err
			}

			if class.HasMainMethod() {
				if mainClass != "" {
					return errors.New("jar has too many main method declarations - only one allowed")
				}
				manifestBytes := []byte(fmt.Sprintf(manifest, strings.ReplaceAll(mainClass, "/", ".")))
				c <- &babe.JarMember{Name: "META-INF/MANIFEST.MF", Buffer: &bytes.Buffer{Data: &manifestBytes, Index: 0}}
			}
			return nil

		})
		return nil
	})
	if err != nil {
		return err
	}

	if err := walkGroup.Wait(); err != nil {
		return err
	}
	close(c)

	return group.Wait()
}
