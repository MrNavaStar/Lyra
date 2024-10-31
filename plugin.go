package main

import (
	"context"
	"path"

	"github.com/mrnavastar/golua/lua"
	"github.com/mrnavastar/goluare"
)

type PluginFunc struct {
	state *lua.State
	function int
}

func (f PluginFunc) Call(args ...any) error {
	f.state.RawGeti(lua.LUA_REGISTRYINDEX, f.function)
	for _, arg := range args {
		f.state.PushGoInterface(arg)
	}
	return f.state.Call(len(args), 0)
}

var api = map[string]lua.LuaGoFunction{
	"add_dependency_resolver": AddDependencyResolver,
}

func LoadPlugin(ctx context.Context, name string) error {
	l := lua.NewState()
	l.OpenLibs()

	l.AtPanic(func(L *lua.State) int {
		print("uh oh!")
		return 1
	})

	l.RegisterLib("string", goluare.REGEX)
	l.RegisterLib("lyra", api)
	l.Pop(2)

	if err := l.DoFile(path.Join(name, "init.lua")); err != nil {
		return err
	}
	return nil
}
