package main

import (
	"context"
	"fmt"
	"path"

	"github.com/aarzilli/golua/lua"
	"github.com/mrnavastar/goluahttp"
	"github.com/mrnavastar/goluaparse"
	"github.com/mrnavastar/goluare"
	"github.com/stevedonovan/luar"
)

type PluginFunc struct {
	state *lua.State
	function int
}

func (f PluginFunc) Call(ret int, args ...any) error {
	f.state.RawGeti(lua.LUA_REGISTRYINDEX, f.function)
	for _, arg := range args {
		luar.GoToLua(f.state, arg)
	}
	return f.state.Call(len(args), ret)
}

func (f PluginFunc) ReturnTable(v interface{}) {
	luar.LuaToGo(f.state, -1, v)
}

func export(l *lua.State) int {
	return 1
}

var api = map[string]lua.LuaGoFunction{
	"add_dependency_resolver": AddDependencyResolver,
	"export_function": export,
}

func LoadPlugin(ctx context.Context, name string) error {
	l := lua.NewState()
	l.OpenLibs()

	l.AtPanic(func(L *lua.State) int {
		print("uh oh!")
		return 1
	})

	l.RegisterLibrary("lyra", api)
	l.RegisterLibrary("re", goluare.REGEX)
	l.RegisterLibrary("http", goluahttp.HTTP)
	l.RegisterLibrary("xml", goluaparse.XML)
	l.RegisterLibrary("json", goluaparse.JSON)
	l.RegisterLibrary("yaml", goluaparse.YAML)
	l.Pop(l.GetTop())

	if err := l.DoFile(path.Join(name, "init.lua")); err != nil {
		return err
	}
	return nil
}

func PrintStack(L *lua.State) {
	t := L.GetTop()
	fmt.Printf("~ | TOP\n")
	for i := t; i >= 1; i-- {
		if L.IsBoolean(i) {
			fmt.Printf("%d | BOOL : %t\n", i, L.ToBoolean(i))
			continue
		}
		if L.IsNumber(i) {
			fmt.Printf("%d | NUM  : %f\n", i, L.ToNumber(i))
			continue
		}
		if L.IsString(i) {
			fmt.Printf("%d | STR  : %s\n", i, L.ToString(i))
			continue
		}
		if L.IsTable(i) {
			fmt.Printf("%d | TBL  : Size:%d\n", i, L.ObjLen(i))
			continue
		}
		if L.IsFunction(i) || L.IsGoFunction(i) {
			fmt.Printf("%d | FUNC\n", i)
			continue
		}
		if L.IsUserdata(i) {
			fmt.Printf("%d | USR\n", i)
			continue
		}
		if L.IsLightUserdata(i) {
			fmt.Printf("%d | LUSR\n", i)
			continue
		}
		if L.IsNil(i) {
			fmt.Printf("%d | NIL\n", i)
		}
	}
	fmt.Printf("~ | BOTTOM\n")
}
