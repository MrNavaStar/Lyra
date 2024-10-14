package main

import (
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
)

const manifest = "Manifest-Version: 1.0\nMain-Class: %s\n";

func Build(ctx *cli.Context) error {
	mod := ctx.Context.Value(modKey).(Module)
	mod.Sync()

	// Create Classpath
	var cp []string
	for _, library := range mod.Libraries {
		cp = append(cp, library.Path)
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

	if err := os.RemoveAll("build/output"); err != nil {
		return err
	}

	cmd := exec.Command("javac", 
		"-d", "build/output",
		"-cp", strings.Join(cp, ";"),
		strings.Join(sp, " "))

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	if ctx.Bool("sources") {
		if err := PackageSources(mod); err != nil {
			return err
		}
	}

	return Package(mod)
}

func Package(mod Module) error {
	c, group := babe.CreateJar(mod.Name + ".jar")

	// Package class files
	var mainClass string
	err := filepath.WalkDir("build/output/", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

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
				return errors.New("project has too many main method declarations - only one allowed")
			}
			mainClass = class.GetClassName()
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Package resources
	err = filepath.WalkDir("src/main/resources/", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		c <- &babe.JarMember{Name: strings.TrimPrefix(path, "src/main/resources/"), Buffer: &bytes.Buffer{Data: &data, Index: 0}}
		return nil
	})
	if err != nil {
		return err
	}

	manifestBytes := []byte(fmt.Sprintf(manifest, strings.ReplaceAll(mainClass, "/", ".")))
	c <- &babe.JarMember{Name: "META-INF/MANIFEST.MF", Buffer: &bytes.Buffer{Data: &manifestBytes, Index: 0}}
	close(c)

	if err := group.Wait(); err != nil {
		return err
	}

	return nil 
}

func PackageSources(mod Module) error {
	c, group := babe.CreateJar(mod.Name + "-sources.jar")

	err := filepath.WalkDir("src/main/java", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() || !strings.HasSuffix(path, ".java") {
			return nil
		}

		var member babe.JarMember
		if err := member.FromFile(path); err != nil {
			return err
		}
		c <- &member
		return nil
	})
	if err != nil {
		return err
	}
	return group.Wait()
}
