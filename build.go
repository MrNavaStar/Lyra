package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/mrnavastar/assist/bytes"
	fss "github.com/mrnavastar/assist/fs"
	"github.com/mrnavastar/babe/babe"
	"github.com/urfave/cli/v2"
)

const manifest = "Manifest-Version: 1.0\nMain-Class: %s\n"

func Build(ctx *cli.Context) error {
	mod := ctx.Context.Value(modKey).(Module)
	mod.Sync()

	// Create Classpath
	var cp []string
	for _, library := range mod.Libraries {
		cp = append(cp, path.Join(mod.Home, "libs", library.Path))
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

	if err := JavaCompile(mod, cp, sp); err != nil {
		return err
	}

	name := ctx.String("output")
	if name == "" {
		name = mod.Name
	}

	if ctx.Bool("sources") {
		if err := PackageSources(name); err != nil {
			return err
		}
	}
	return Package(name)
}

func Package(name string) error {
	c, group := babe.CreateJar(path.Join("build/jar", name+".jar"))

	// Package class files
	var mainClass string
	err := filepath.WalkDir("build/output/", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		var member babe.JarMember
		if err := member.FromFile(path); err != nil {
			return err
		}
		member.Name = strings.TrimPrefix(member.Name, "build/output/")
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

	if fss.Exists("src/main/resources/") {
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
	}

	manifestBytes := []byte(fmt.Sprintf(manifest, strings.ReplaceAll(mainClass, "/", ".")))
	c <- &babe.JarMember{Name: "META-INF/MANIFEST.MF", Buffer: &bytes.Buffer{Data: &manifestBytes, Index: 0}}
	close(c)

	if err := group.Wait(); err != nil {
		return err
	}

	return nil
}

func PackageSources(name string) error {
	c, group := babe.CreateJar(path.Join("build/jar", name+"-sources.jar"))

	err := filepath.WalkDir("src/main/java/", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() || !strings.HasSuffix(path, ".java") {
			return nil
		}

		var member babe.JarMember
		if err := member.FromFile(path); err != nil {
			return err
		}
		member.Name = strings.TrimPrefix(member.Name, "src/main/java/")
		c <- &member
		return nil
	})
	if err != nil {
		return err
	}
	close(c)
	return group.Wait()
}
