package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/aarzilli/golua/lua"
	"github.com/mrnavastar/assist/ops"
	"github.com/mrnavastar/assist/web"
	"github.com/mrnavastar/babe/babe"
	"github.com/urfave/cli/v2"
)

var resolvers []PluginFunc

type Dependency struct {
	Name	string `json:",omitempty"`
	Version string `json:",omitempty"`
	Main	string `json:",omitempty"`
	Sources string `json:",omitempty"`
	Docs	string `json:",omitempty"`
	Include bool   `json:",omitempty"`
}

func (d *Dependency) resolve(project Project, uri *string) (string, error) {
	parsedURL, err := url.Parse(*uri)
    if err != nil {
        return "", err
    }

	switch parsedURL.Scheme {
	case "http", "https":
		path := path.Join(project.Home, "libs", parsedURL.Path)

		if err := web.Download(path, *uri); err != nil {
			return "", err
		}

		*uri = "file://" + path
		return d.resolve(project, uri)

	case "file", "":
		path := parsedURL.Path
        // For Windows, remove the leading `/` in paths like `file:///C:/path/to/file`
        if strings.HasPrefix(path, "/") && filepath.VolumeName(path) != "" {
            path = strings.TrimPrefix(path, "/")
        }
        return filepath.Clean(path), nil
	}

	return "", errors.New("unable to resolve dependency: " + d.Main)
}

func (d *Dependency) ResolveMain(project Project) (string, error) {
	return d.resolve(project, &d.Main)
}

func (d *Dependency) ResolveSources(project Project) (string, error) {
	if d.Sources == "" {
		return "", nil
	}
	return d.resolve(project, &d.Sources)
}

func (d *Dependency) ResolveDocs(project Project) (string, error) {
	if d.Docs == "" {
		return "", nil
	}
	return d.resolve(project, &d.Docs)
}

func AddDependencyResolver(l *lua.State) int {
	resolvers = append(resolvers, PluginFunc{state: l, function: l.Ref(lua.LUA_REGISTRYINDEX)})
	return 1
}

func GetDependencyFromCLI(ctx *cli.Context) error {
	if ctx.Args().Len() == 0 {
		return errors.New("no maven artifact provided")
	}
	return GetDependency(ctx.Context, ctx.Args().First(), ctx.Bool("include"))
}

func GetDependencyFromAPI(l *lua.State) int {
	if err := GetDependency(context.Background(), l.ToString(1), false); err != nil {
		l.PushString(fmt.Sprintf("%s", err))
		return 1
	}
	return 0
}

func GetDependency(ctx context.Context, coordinate string, include bool) error {
	project := ctx.Value(projectKey).(Project)
	if ops.IsEmpty(project) {
		return errors.New("no project found in current directory")
	}

	for _, resolver := range resolvers {
		var deps map[string]Dependency
		if err := resolver.Call(1, project.Repos, coordinate); err != nil {
			return err
		}
		resolver.ReturnTable(&deps)

		if ops.IsEmpty(deps) {
			continue
		} 

		for key, dep := range deps {
			if _, err := dep.ResolveMain(project); err != nil {
				return err
			}
			if _, err := dep.ResolveSources(project); err != nil {
				dep.Sources = ""
			}
			if _, err := dep.ResolveDocs(project); err != nil {
				dep.Docs = ""
			}
			project.Dependencies[key] = dep
			println("got: " + key + ":" + dep.Version)
		}
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

	_, err := http.Get(repo)
	if err != nil {
		return fmt.Errorf("repo is unreachable: %s", err)
	}

	project.Repos = append(project.Repos, repo)
	return project.Save()
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
