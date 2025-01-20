package lyra

import (
	"errors"
	"github.com/mrnavastar/assist/fs"
	"github.com/mrnavastar/assist/web"
	"github.com/urfave/cli/v2"
	"net/url"
	"path"
	"path/filepath"
	"strings"
)

type Artifact struct {
	Name         string     `json:",omitempty"`
	Group        string     `json:",omitempty"`
	Version      string     `json:",omitempty"`
	Main         string     `json:",omitempty"`
	Sources      string     `json:",omitempty"`
	Docs         string     `json:",omitempty"`
	Include      bool       `json:",omitempty"`
	Dependencies []Artifact `json:",omitempty"`
}

func init() {
	Dependency.RegisterResolver("http", resolveHttp)
	Dependency.RegisterResolver("https", resolveHttp)

	Command.RegisterMany([]*cli.Command{
		{
			Name:   "get",
			Args:   true,
			Action: get,
			Subcommands: []*cli.Command{
				{
					Name:   "repo",
					Args:   true,
					Action: addRepo,
				},
			},
		},
		{
			Name: "tidy",
		},
	})
}

func resolveHttp(url *url.URL) (string, error) {
	cache, err := GetCache()
	if err != nil {
		return "", err
	}

	file := filepath.Base(url.Path)
	urlPath := strings.TrimSuffix(url.Path, file)
	localPath := path.Join(cache, "libs", "http", url.Host, urlPath, file)
	if err := web.Download(localPath, url.String()); err != nil {
		return "", err
	}
	return "file://" + localPath, nil
}

func (artifact Artifact) resolve(uri string) (string, error) {
	parsedURL, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	resolver, ok := Dependency.resolvers[parsedURL.Scheme]
	if !ok {
		return "", errors.New("no resolver registered for scheme: " + parsedURL.Scheme)
	}
	resolved, err := resolver(parsedURL)
	if err != nil {
		return "", err
	}
	resolvedURL, err := url.Parse(resolved)
	if err != nil {
		return "", err
	}
	if resolvedURL.Scheme != "file" {
		return artifact.resolve(resolved)
	}

	// For Windows, remove the leading `/` in paths like `file:///C:/path/to/file`
	localPath := resolvedURL.Path
	if strings.HasPrefix(localPath, "/") && filepath.VolumeName(localPath) != "" {
		localPath = strings.TrimPrefix(localPath, "/")
	}
	return filepath.Clean(localPath), nil
}

func (artifact Artifact) Resolve() (string, error) {
	return artifact.resolve(artifact.Main)
}

func (artifact Artifact) ResolveSources() (string, error) {
	if artifact.Sources == "" {
		return "", nil
	}
	return artifact.resolve(artifact.Sources)
}

func (artifact Artifact) ResolveDocs() (string, error) {
	if artifact.Docs == "" {
		return "", nil
	}
	return artifact.resolve(artifact.Docs)
}

func (artifact Artifact) SameAs(other Artifact) bool {
	return artifact.Name == other.Name && artifact.Group == other.Group
}

func get(ctx *cli.Context) error {
	if !fs.Exists("lyra.json") {
		return errors.New("no project in current directory")
	}

	if !ctx.Args().Present() {
		return errors.New("please specify at least one slug")
	}

	for _, slug := range ctx.Args().Slice() {
		GetCurrentProject().Go(func() error {
			for _, parser := range Dependency.parsers {
				artifact, err := parser(slug)
				if err != nil {
					return err
				}

				if err := GetCurrentProject().AddDependency(artifact); err == nil {
					return nil
				}
			}
			return nil
		})
	}
	return nil
}

func addRepo(ctx *cli.Context) error {
	if !fs.Exists("lyra.json") {
		return errors.New("no project in current directory")
	}
	if ctx.Args().Len() == 0 {
		return errors.New("no repo provided")
	}

	parsed, err := url.Parse(ctx.Args().First())
	if err != nil {
		return err
	}
	return GetCurrentProject().AddRepo(*parsed)
}

/*func FindModuleDependencies(project *Project, name string) (dependencies []Dependency, err error) {
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
}*/
