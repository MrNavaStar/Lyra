package minecraft

import (
	"github.com/mrnavastar/lyra/lyra"
	"github.com/urfave/cli/v2"
	"strings"
)

func init() {
	lyra.Dependency.RegisterParser(minecraftParser)

	lyra.Command.Register(&cli.Command{
		Name:    "minecraft",
		Aliases: []string{"mc"},
		Subcommands: []*cli.Command{
			{
				Name: "run",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name: "fabric",
					},
				},
				Action: run,
			},
		},
	})
}

func minecraftParser(slug string) (lyra.Artifact, error) {
	artifact := lyra.Dependency.ParseMavenCoordinate(slug)
	if artifact.Group != "com.mojang" || !strings.HasPrefix(artifact.Name, "minecraft") || artifact.Version == "" {
		return artifact, nil
	}

	minecraftVersion, err := GetMinecraftVersion(artifact.Version)
	if err != nil {
		return artifact, err
	}

	project := lyra.GetCurrentProject()
	for _, library := range minecraftVersion.GetLibraries() {
		project.Go(func() error {
			return project.AddDependency(library)
		})
	}

	if strings.HasSuffix(artifact.Name, "client") {
		project.Go(func() error {
			return project.AddDependency(minecraftVersion.GetClientArtifact())
		})
	}
	if strings.HasSuffix(artifact.Name, "server") {
		project.Go(func() error {
			return project.AddDependency(minecraftVersion.GetServerArtifact())
		})
	}
	return artifact, nil
}

func run(ctx *cli.Context) error {
	if err := lyra.Build.Project(lyra.BuildOptions{}); err != nil {
		return err
	}

	classpath, err := lyra.GetCurrentProject().GetClasspath()
	if err != nil {
		return err
	}

	var mainClass string
	if ctx.Bool("fabric") {
		if ctx.Bool("client") {
			mainClass = "net.fabricmc.devlaunchinjector.Main"
		} else {
			mainClass = "net.fabricmc.loader.impl.launch.knot.KnotServer"
		}
	}

	return lyra.Java.Run(lyra.JavaRunOptions{
		Classpath: classpath,
		MainClass: mainClass,
	})
}

func addFabricToProject(minecraftVersion MinecraftVersion) error {
	fabricLoaderVersion, err := GetLatestFabricVersion(minecraftVersion)
	if err != nil {
		return err
	}

	project := lyra.GetCurrentProject()
	for _, library := range fabricLoaderVersion.GetLibraries() {
		project.Go(func() error {
			return project.AddDependency(library)
		})
	}
	return project.AddDependency(fabricLoaderVersion.GetLoader())
}

func addNeoForgeToProject() {

}

func addPaperToProject() {

}
