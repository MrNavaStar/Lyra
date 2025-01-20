package lyra

import (
	"fmt"
	"github.com/mrnavastar/assist/bytes"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	fss "github.com/mrnavastar/assist/fs"
	"github.com/mrnavastar/babe/babe"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
)

func init() {
	Command.Register(&cli.Command{
		Name:   "build",
		Args:   false,
		Action: build,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "fat",
				Aliases: []string{"f"},
			},
			&cli.BoolFlag{
				Name:    "minimize",
				Aliases: []string{"min", "m"},
			},
			&cli.BoolFlag{
				Name:    "sources",
				Aliases: []string{"s"},
			},
			&cli.BoolFlag{
				Name:    "docs",
				Aliases: []string{"d"},
			},
		},
	})

	Build.AddManifestEntry("Created-By", "Lyra")
}

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

func build(ctx *cli.Context) error {
	project := GetCurrentProject()
	for _, hook := range Build.Hooks.preCompile {
		if err := hook(project); err != nil {
			return err
		}
	}

	// Create Classpath
	var cp []string
	for _, artifact := range project.artifacts {
		artifactPath, err := artifact.Resolve()
		if err != nil {
			return err
		}
		cp = append(cp, artifactPath)
	}

	files, err := os.ReadDir("src")
	if err != nil {
		return err
	}

	buildGroup, _ := errgroup.WithContext(ctx.Context)
	for _, module := range files {
		buildGroup.Go(func() error {
			// Create Sourcepath
			var sources []string
			if err := filepath.WalkDir(path.Join("src", module.Name(), "java"), func(path string, d fs.DirEntry, err error) error {
				if d.IsDir() {
					return nil
				}

				if strings.HasSuffix(path, ".java") {
					sources = append(sources, path)
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

				classpath, err := project.GetClasspath()
				if err != nil {
					return err
				}

				if err := Java.Compile(JavaCompileOptions{
					Classpath: classpath,
					Sources:   sources,
				}); err != nil {
					return err
				}
				outputTime = time.Now()
			}

			buildGroup.Go(func() error {
				return Package(module.Name(), outputTime, ctx.Bool("fat"))
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

func Package(name string, outputTime time.Time, fat bool) error {
	project := GetCurrentProject()
	filename := path.Join("build/jar", name+".jar")
	resources := path.Join("src", name, "resources")
	resourceTime, _ := getNewestTime(resources)

	// Don't repackage jar if resources and compiled sources are up to date
	info, err := os.Stat(filename)
	if !os.IsNotExist(err) && outputTime.Before(info.ModTime()) && resourceTime.Before(info.ModTime()) {
		return nil
	}

	jar := babe.CreateJar(filename)
	for _, hook := range Build.Hooks.prePackageJar {
		if err := hook(project, jar); err != nil {
			return err
		}
	}

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
			return filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
				if d.IsDir() {
					return nil
				}

				jar.Task(func(jar *babe.Jar) error {
					member, err := babe.JarMemberFromFile(path)
					if err != nil {
						return err
					}

					class, err := member.GetAsClass()
					if err != nil {
						return err
					}

					for _, hook := range Build.Hooks.packageClass {
						err := hook(project, *jar, &class)
						if err != nil {
							return err
						}
					}
					var b []byte
					class.Write(&b)
					member.Buffer = &bytes.Buffer{Data: &[]byte{}, Index: 0}
					jar.Add(member)

					return nil
				})
				return nil
			})
		})
	})

	if err := jar.Wait(); err != nil {
		return err
	}

	// Create manifest
	manifest := "Manifest-Version: 1.0\n"
	for field, value := range Build.manifestEntries {
		manifest += fmt.Sprintf("%s: %s\n", field, value)
	}
	jar.Add(babe.JarMemberFromString("META-INF/MANIFEST.MF", manifest))
	return nil
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
