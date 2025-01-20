package lyra

import (
	"errors"
	"github.com/mrnavastar/babe/babe"
	"github.com/urfave/cli/v2"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"sync"
)

//----- [App] ----------------------------------------------------------------------------------------------------------

var app = cli.App{
	Name:                   "lyra",
	Args:                   true,
	UseShortOptionHandling: true,
	EnableBashCompletion:   true,
	Suggest:                true,
	Authors: []*cli.Author{
		{
			Name:  "MrNavaStar",
			Email: "Mr.NavaStar@gmail.com",
		},
	},
}

func (*CommandAPI) Run(args ...string) error {
	app.Commands = Command.commands
	return app.Run(args)
}

//----- [Project] ------------------------------------------------------------------------------------------------------

var project Project

func init() {
	err := project.Load()
	if err != nil {
		log.Fatal(err)
	}
}

func GetCurrentProject() *Project {
	return &project
}

//----- [Util] ---------------------------------------------------------------------------------------------------------

// GetCache returns the full path to the lyra cache directory. This location should be used to store any temp files.
func GetCache() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return path.Join(dir, "lyra"), nil
}

// PingResource returns true if a resource at a given endpoint is reachable without downloading the file
func PingResource(uri *url.URL) bool {
	request := http.Request{
		Host:   uri.String(),
		Method: "GET",
		Header: http.Header{
			"Range": {"bytes=0-5"},
		},
	}
	client := &http.Client{}
	response, err := client.Do(&request)
	if err != nil {
		return false
	}
	return response.StatusCode == 200
}

//----- [Java] ---------------------------------------------------------------------------------------------------------

type JavaAPI struct {
	mu sync.Mutex

	java string
}

var Java JavaAPI

func (*JavaAPI) SetPath(javaPath string) error {
	Java.mu.Lock()
	defer Java.mu.Unlock()
	if Java.java != "" {
		return errors.New("java path has already been set by another plugin")
	}
	Java.java = path.Clean(javaPath)
	return nil
}

func (*JavaAPI) GetPath() string {
	Java.mu.Lock()
	defer Java.mu.Unlock()
	return Java.java
}

//----- [CommandAPI] -----------------------------------------------------------------------------------------------------

type CommandAPI struct {
	mu sync.Mutex

	commands []*cli.Command
}

var Command CommandAPI

// Register registers a command for the current lyra session.
func (*CommandAPI) Register(command *cli.Command) {
	Command.mu.Lock()
	defer Command.mu.Unlock()
	Command.commands = append(Command.commands, command)
}

// RegisterMany registers a list of commands for the current lyra session.
func (*CommandAPI) RegisterMany(commands []*cli.Command) {
	Command.mu.Lock()
	defer Command.mu.Unlock()
	Command.commands = append(Command.commands, commands...)
}

//----- [DependencyAPI] -------------------------------------------------------------------------------------------------

type DependencyAPI struct {
	mu sync.Mutex

	repoAcceptors []func(uri url.URL) bool
	parsers       []func(slug string) (Artifact, error)
	resolvers     map[string]func(uri *url.URL) (string, error)
}

var Dependency DependencyAPI

func (*DependencyAPI) RegisterRepoAcceptor(acceptor func(uri url.URL) bool) {
	Dependency.repoAcceptors = append(Dependency.repoAcceptors, acceptor)
}

func (*DependencyAPI) RegisterParser(parser func(slug string) (Artifact, error)) {
	Dependency.mu.Lock()
	defer Dependency.mu.Unlock()
	Dependency.parsers = append(Dependency.parsers, parser)
}

func (*DependencyAPI) RegisterResolver(scheme string, resolver func(uri *url.URL) (string, error)) {
	Dependency.mu.Lock()
	defer Dependency.mu.Unlock()
	if Dependency.resolvers == nil {
		Dependency.resolvers = map[string]func(uri *url.URL) (string, error){}
	}
	Dependency.resolvers[scheme] = resolver
}

//----- [BuildAPI] -----------------------------------------------------------------------------------------------------

type BuildHooks struct {
	preCompile    []func(*Project) error
	prePackageJar []func(*Project, babe.Jar) error
	packageClass  []func(*Project, babe.Jar, *babe.Class) error
}

type BuildAPI struct {
	mu sync.Mutex

	Hooks           BuildHooks
	manifestEntries map[string]string
}

var Build BuildAPI

func (*BuildAPI) AddManifestEntry(field string, value string) {
	Build.mu.Lock()
	defer Build.mu.Unlock()
	if Build.manifestEntries == nil {
		Build.manifestEntries = map[string]string{}
	}
	Build.manifestEntries[field] = value
}

func (*BuildAPI) HasManifestEntry(field string) bool {
	Build.mu.Lock()
	defer Build.mu.Unlock()

	if Build.manifestEntries == nil {
		return false
	}
	_, ok := Build.manifestEntries[field]
	return ok
}

func (BuildHooks) PreCompile(hook func(*Project) error) {
	Build.mu.Lock()
	defer Build.mu.Unlock()
	Build.Hooks.preCompile = append(Build.Hooks.preCompile, hook)
}

func (BuildHooks) PrePackageJar(hook func(*Project, babe.Jar) error) {
	Build.mu.Lock()
	defer Build.mu.Unlock()
	Build.Hooks.prePackageJar = append(Build.Hooks.prePackageJar, hook)
}

func (BuildHooks) PackageClass(hook func(*Project, babe.Jar, *babe.Class) error) {
	Build.mu.Lock()
	defer Build.mu.Unlock()
	Build.Hooks.packageClass = append(Build.Hooks.packageClass, hook)
}
