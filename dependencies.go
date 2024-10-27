package main

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/aarzilli/golua/lua"
	"github.com/mrnavastar/assist/ops"
	"github.com/mrnavastar/babe/babe"
	"github.com/urfave/cli/v2"
)

var resolvers []PluginFunc

type Dependency struct {
	Coordinate string `json:",omitempty"`
	Path       string `json:",omitempty"`
	Include    bool   `json:",omitempty"`
}

func AddDependencyResolver(l *lua.State) int {
	resolvers = append(resolvers, PluginFunc{state: l, function: l.ToGoFunction(1)})
	return 1
}

func GetDependencyFromCLI(ctx *cli.Context) error {
	if ctx.Args().Len() == 0 {
		return errors.New("no maven artifact provided")
	}
	return GetDependency(ctx.Context, ctx.Args().First(), ctx.Bool("include"))
}

func GetDependency(ctx context.Context, coordinate string, include bool) error {
	project := ctx.Value(projectKey).(Project)
	if ops.IsEmpty(project) {
		return errors.New("no project found in current directory")
	}

	for _, resolver := range resolvers {
		resolver.Call(project.Repos, coordinate)
	}

	return project.Save()
}

func AddRepo(ctx *cli.Context) error {
	project := ctx.Context.Value(projectKey).(Project)
	if ctx.Args().Len() == 0 {
		return errors.New("no repo provided")
	}

	repo := ctx.Args().First()
	if !strings.HasPrefix(repo, "https://") {
		repo = "https://" + repo
	}

	repo = strings.TrimRight(repo, "/")

	for _, r := range project.Repos {
		if r == repo {
			return nil
		}
	}

	resp, err := http.Get(repo)
	if err != nil {
		return err
	} else if resp.StatusCode != http.StatusOK {
		return errors.New("repo is unreachable")
	}

	project.Repos = append(project.Repos, repo)
	return project.Save()
}

func DownloadDependencies(project Project) error {
	return nil
}

func FindModuleDependencies(project Project, name string) (dependencies []Dependency, err error) {
	var classes []string
	if err := filepath.WalkDir(path.Join("build/output", name), func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() || !strings.HasSuffix(path, ".class") {
			return nil
		}

		member, err := babe.JarMemberFromFile(path)
		if err != nil {
			return err
		}

		class, err := member.GetAsClass()
		if err != nil {
			return err
		}

		// Add class imports as dependencies unless they start with java, as those are always provided
		for _, constant := range class.ConstantPool {
			if info, ok := constant.(*babe.ClassInfo); ok {
				dep := class.GetConstant(info.NameIndex).(babe.Utf8Info).String()
				if !strings.HasPrefix(dep, "java") {
					classes = append(classes, dep)
				}
			}
		}
		return nil
	}); err != nil {
		return dependencies, err
	}

	if err := filepath.WalkDir(path.Join(project.Home, "libs"), func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() || !strings.HasSuffix(path, ".jar") {
			return nil
		}

		return babe.ForJarMember(path, func(member *babe.JarMember) error {
			for _, class := range classes {
				if class == strings.TrimSuffix(strings.ReplaceAll(member.Name, "/", "."), ".class") {
					dep, err := project.GetDependencyByPath(path)
					if err != nil {
						return err
					}
					dependencies = append(dependencies, *dep)
					return nil
				}
			}
			return nil
		})
	}); err != nil {
		return dependencies, err
	}
	return dependencies, nil
}
