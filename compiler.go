package main

import (
	"embed"
	"errors"
	"github.com/mrnavastar/lyra/lyra"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"
)

//go:embed *.mod *.sum *.go lyra/**.go plugins/**/*.go
var lyraSRC embed.FS

func init() {
	lyra.Command.Register(&cli.Command{
		Name: "plugin",
		Subcommands: []*cli.Command{
			{
				Name:   "get",
				Args:   true,
				Action: installPlugins,
			},
			{
				Name:   "list",
				Args:   false,
				Action: listPlugins,
			},
		},
	})

	if bin, err := getBinary(); err == nil {
		os.Remove(bin + "_old")
	}
}

func exportEmbededFS(embeded embed.FS, exportPath string) error {
	return fs.WalkDir(embeded, ".", func(dir string, d fs.DirEntry, err error) error {
		fullPath := path.Join(exportPath, dir)

		if d.IsDir() {
			err := os.Mkdir(fullPath, os.ModePerm)
			if err != nil && !errors.Is(err, os.ErrExist) {
				return err
			}
			return nil
		}

		bytes, err := embeded.ReadFile(dir)
		if err != nil {
			return err
		}
		return os.WriteFile(fullPath, bytes, os.ModePerm)
	})
}

func setup() (string, error) {
	cache, err := lyra.GetCache()
	if err != nil {
		return "", err
	}
	src := path.Join(cache, "plugin")
	if err := os.RemoveAll(src); err != nil {
		return "", err
	}
	if err := os.MkdirAll(src, os.ModePerm); err != nil {
		return "", err
	}
	return src, exportEmbededFS(lyraSRC, src)
}

func getBinary() (string, error) {
	bin, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(bin)
}

func recompile(src string) error {
	// Setup lyra project
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = src
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	// Rebuild lyra project
	cmd = exec.Command("go", "build", "-o", "bin")
	cmd.Dir = src
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	// Test new lyra binary
	cmd = exec.Command("./bin", "plugin", "list")
	cmd.Dir = src
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	println("The following plugins are now installed:")
	if err := cmd.Run(); err != nil {
		return err
	}

	// Overwrite old lyra binary
	binaryPath, err := getBinary()
	if err != nil {
		return err
	}
	bytes, err := os.ReadFile(path.Join(src, "bin"))
	if err != nil {
		return err
	}
	if err := os.Rename(binaryPath, binaryPath+"_old"); err != nil {
		return err
	}
	return os.WriteFile(binaryPath, bytes, os.ModePerm)
}

func installPlugins(ctx *cli.Context) error {
	src, err := setup()
	if err != nil {
		return err
	}

	// Install plugins as go modules
	pluginFile, err := os.OpenFile(path.Join(src, "plugins.go"), os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	for _, slug := range ctx.Args().Slice() {
		_, err := pluginFile.WriteString("import _ \"" + slug + "\"\n")
		if err != nil {
			return err
		}
	}
	pluginFile.Close()

	return recompile(src)
}

func listPlugins(ctx *cli.Context) error {
	bytes, err := lyraSRC.ReadFile("plugins.go")
	if err != nil {
		return err
	}

	for _, line := range strings.Split(string(bytes), "\n") {
		if strings.HasPrefix(line, "import _") {
			println(strings.Replace(strings.Replace(line, "import _ \"", "", 1), "\"", "", 1))
		}
	}
	return nil
}
