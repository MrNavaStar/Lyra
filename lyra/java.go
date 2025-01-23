package lyra

import (
	"github.com/mrnavastar/assist/fs"
	"github.com/urfave/cli/v2"
	"os"
	"os/exec"
	"path"
	"strings"
)

func init() {
	Command.Register(&cli.Command{
		Name: "java",
		Subcommands: []*cli.Command{
			{
				Name:   "info",
				Action: javaInfo,
			},
		},
	})
}

func javaInfo(ctx *cli.Context) error {
	println(Java.GetPath())
	return nil
}

func (*JavaAPI) IsInstalled() bool {
	return fs.Exists(path.Join(Java.GetPath(), "java")) && fs.Exists(path.Join(Java.GetPath(), "javac"))
}

type JavaCompileOptions struct {
	Classpath []string
	Sources   []string
}

func (*JavaAPI) Compile(options JavaCompileOptions) error {
	cmd := exec.Command(path.Join(Java.GetPath(), "javac"),
		"-d", "build/output",
		"-cp", strings.Join(options.Classpath, string(os.PathListSeparator)),
		"-encoding", "utf8",
		"-sourcepath", "build/override:src", strings.Join(options.Sources, " "),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

type JavaRunOptions struct {
	Classpath   []string
	JvmArgs     []string
	ProgramArgs []string
	Jar         string
	MainClass   string
}

func (*JavaAPI) Run(options JavaRunOptions) error {
	cmd := exec.Command(path.Join(Java.GetPath(), "java"))
	if len(options.Classpath) > 0 {
		cmd.Args = append(cmd.Args, "-cp", strings.Join(options.Classpath, string(os.PathListSeparator)))
	}
	if options.Jar != "" {
		cmd.Args = append(cmd.Args, "-jar", options.Jar)
	}
	if options.MainClass != "" {
		cmd.Args = append(cmd.Args, options.MainClass)
	}
	cmd.Args = append(cmd.Args, options.ProgramArgs...)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
