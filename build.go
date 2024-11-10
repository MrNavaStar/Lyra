package main

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	fss "github.com/mrnavastar/assist/fs"
	"github.com/mrnavastar/babe/babe"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
)

const manifest = "Manifest-Version: 1.0\nMain-Class: %s\n"

func getNewestTime(directory string) (time.Time, error) {
	var newest time.Time

	if err := filepath.WalkDir(directory, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		fileInfo, err := os.Stat(path)
		if err != nil {
			return err
		}

		if fileInfo.ModTime().Compare(newest) >= 0 {
			newest = fileInfo.ModTime()
		}
		return nil
	}); err != nil {
		return time.UnixMilli(0), err
	}
	return newest, nil
}

func Build(ctx *cli.Context) error {
	project := ctx.Context.Value(projectKey).(Project)

	// Create Classpath
	var cp []string
	for _, library := range project.Dependencies {
		dep_path, err := library.ResolveMain(project)
		if err != nil {
			return err
		}
		cp = append(cp, dep_path)
	}

	files, err := os.ReadDir("src")
	if err != nil {
		return err
	}

	buildGroup, _ := errgroup.WithContext(ctx.Context)
	for _, module := range files {
		buildGroup.Go(func() error {
			// Create Sourcepath
			var sp []string
			if err := filepath.WalkDir(path.Join("src", module.Name(), "java"), func(path string, d fs.DirEntry, err error) error {
				if d.IsDir() {
					return nil
				}

				if strings.HasSuffix(path, ".java") {
					sp = append(sp, path)
				}
				return nil
			}); err != nil {
				return err
			}

			// Only recompile if the source code is newer than the already compiled code
			sourceTime, _ := getNewestTime(path.Join("src", module.Name(), "java"))
			outputTime, _ := getNewestTime(path.Join("build/output", module.Name()))
			if outputTime.Before(sourceTime) {
				if err := os.RemoveAll(path.Join("build/output", module.Name())); err != nil {
					return err
				}

				if err := JavaCompile(project, cp, sp); err != nil {
					return err
				}
				outputTime = time.Now()
			}

			buildGroup.Go(func() error {
				return Package(project, module.Name(), outputTime, ctx.Bool("fat"))
			})

			if ctx.Bool("sources") {
				buildGroup.Go(func() error {
					return PackageSources(module.Name(), outputTime)
				})
			}
			return nil
		})
	}

	return buildGroup.Wait()
}

func Package(project Project, name string, outputTime time.Time, fat bool) error {
	filename := path.Join("build/jar", name+".jar")
	resources := path.Join("src", name, "resources")
	resourceTime, _ := getNewestTime(resources)

	// Don't repackage jar if resources and compiled sources are up to date
	info, err := os.Stat(filename)
	if !os.IsNotExist(err) && outputTime.Before(info.ModTime()) && resourceTime.Before(info.ModTime()) {
		return nil
	}

	jar := babe.CreateJar(filename)

	// Package resources async
	if fss.Exists(resources) {
		jar.Task(func(jar *babe.Jar) error {
			return fss.Cwd(resources, func() error {
				return filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
					if d.IsDir() {
						return nil
					}

					jar.Task(func(jar *babe.Jar) error {
						member, err := babe.JarMemberFromFile(path)
						if err != nil {
							return err
						}
						jar.Add(member)
						return nil
					})
					return nil
				})
			})
		})
	}

	// Package class files async
	jar.Task(func(jar *babe.Jar) error {
		return fss.Cwd(path.Join("build/output", name), func() error {
			var manifestAdded atomic.Bool
			return filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
				if d.IsDir() {
					return nil
				}

				jar.Task(func(jar *babe.Jar) error {
					member, err := babe.JarMemberFromFile(path)
					if err != nil {
						return err
					}
					jar.Add(member)

					class, err := member.GetAsClass()
					if err != nil {
						return err
					}

					if class.HasMainMethod() {
						if manifestAdded.Load() {
							return fmt.Errorf("module: %s has too many main method declarations - only one allowed", name)
						}
						jar.Add(babe.JarMemberFromString("META-INF/MANIFEST.MF", fmt.Sprintf(manifest, strings.ReplaceAll(class.GetClassName(), "/", "."))))
						manifestAdded.Store(true)
					}
					return nil
				})
				return nil
			})
		})
	})
	return jar.Wait()
}

func PackageSources(name string, outputTime time.Time) error {
	filename := path.Join("build/jar", name+"-sources.jar")

	// Don't repackage sources if they are already up to date
	info, err := os.Stat(filename)
	if !os.IsNotExist(err) && outputTime.Before(info.ModTime()) {
		return nil
	}

	jar := babe.CreateJar(filename)
	if err := fss.Cwd(path.Join("src", name, "java"), func() error {
		return filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
			if d.IsDir() || !strings.HasSuffix(path, ".java") {
				return nil
			}

			jar.Task(func(jar *babe.Jar) error {
				member, err := babe.JarMemberFromFile(path)
				if err != nil {
					return err
				}
				jar.Add(member)
				return nil
			})
			return nil
		})
	}); err != nil {
		return err
	}
	return jar.Wait()
}
