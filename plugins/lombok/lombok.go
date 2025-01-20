package lombok

import (
	"fmt"
	"github.com/mrnavastar/lyra/lyra"
	"github.com/urfave/cli/v2"
)

const lombokVersion = "1.18.36"

var lombokJar = lyra.Artifact{
	Name:    "lombok",
	Group:   "org.projectlombok",
	Version: lombokVersion,
	Main:    fmt.Sprintf("https://repo1.maven.org/maven2/org/projectlombok/lombok/%s/lombok-%s.jar", lombokVersion, lombokVersion),
	Sources: fmt.Sprintf("https://repo1.maven.org/maven2/org/projectlombok/lombok/%s/lombok-%s-sources.jar", lombokVersion, lombokVersion),
	Docs:    fmt.Sprintf("https://repo1.maven.org/maven2/org/projectlombok/lombok/%s/lombok-%s-javadoc.jar", lombokVersion, lombokVersion),
}

func init() {
	lyra.Command.Register(&cli.Command{
		Name: "lombok",
		Subcommands: []*cli.Command{
			{
				Name:        "version",
				Args:        false,
				Aliases:     []string{"v", "ver"},
				Description: "prints the version of the globally installed lombok jar",
				Action: func(ctx *cli.Context) error {
					return lombok("version")
				},
			},
			{
				Name:        "add",
				Args:        false,
				Description: "adds lombok to the current project",
				Action: func(ctx *cli.Context) error {
					return lyra.GetCurrentProject().AddDependency(lombokJar)
				},
			},
		},
	})

	/*lyra.Build.Hooks.PreCompile(func(project *lyra.Project) error {
		for _, artifact := range project.Dependencies() {
			if artifact.SameAs(lombokJar) {
				return lombok("delombok", "src", "-d", "build/override", "-e", "UTF-8", "--onlyChanged")
			}
		}
		return nil
	})*/
}

func lombok(args ...string) error {
	jarPath, err := lombokJar.Resolve()
	if err != nil {
		return err
	}
	return lyra.Java.Run(lyra.JavaRunOptions{
		Jar:         jarPath,
		ProgramArgs: args,
	})
}
