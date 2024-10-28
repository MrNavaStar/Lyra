package main

import (
	"context"
	"path"

	"github.com/aarzilli/golua/lua"
	"github.com/ambrevar/golua/unicode"
)

type PluginFunc struct {
	state *lua.State
	function lua.LuaGoFunction
}

func test(l *lua.State) int {
	print("pog")
	return 1
}

func (f PluginFunc) Call(args ...any) error {
	//var luaArgs []lua.LValue
	//for _, arg := range args {
	//	luaArgs = append(luaArgs, gluaparse.DecodeValue(f.state, arg))
	//}

	//f.state.MustCall()

	//return f.state.CallByParam(lua.P{
	//	Fn: f.function,
	//	NRet: 1,
	//	Protect: true,
    //}, luaArgs...)
	return nil
}

var api = map[string]lua.LuaGoFunction{
	"add_dependency_resolver": AddDependencyResolver,
	"pog": test,
}

func LoadPlugin(ctx context.Context, name string) error {
	l := lua.NewState()
	l.OpenLibs()

	l.AtPanic(func(L *lua.State) int {
		print("uh oh!")
		return 1
	})

	// Load plugin libraries
	//gluaparse.PreloadJSON(l)
	//gluaparse.PreloadYAML(l)
	//gluaparse.PreloadXML(l)
	//l.PreloadModule("re", gluare.Loader)
	//l.PreloadModule("http", gluahttp.NewHttpModule(&http.Client{}).Loader)
	
	unicode.GoLuaReplaceFuncs(l)

	l.NewMetaTable("lyra")
	for name, function := range api {
		//l.PushGoFunction(function)
		//l.SetField(-2, name)
		l.SetMetaMethod(name, function)
	}

	if err := l.DoFile(path.Join(name, "based.lua")); err != nil {
		return err
	}
	return nil
}
