package lyra

import (
	"github.com/mrnavastar/assist/fs"
	"os"
	"os/exec"
	"path"
	"strings"
)

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
}

func (*JavaAPI) Run(options JavaRunOptions) error {
	cmd := exec.Command(path.Join(Java.GetPath(), "java"),
		"-cp", strings.Join(options.Classpath, string(os.PathListSeparator)),
		"-jar", options.Jar,
	)
	cmd.Args = append(cmd.Args, options.ProgramArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
