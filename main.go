package main

import (
	"github.com/mrnavastar/lyra/lyra"
	"log"
	"os"
	"path"
	"path/filepath"
)

func main() {
	if lyra.Java.GetPath() == "" {
		javaHome := os.Getenv("JAVA_HOME")
		if javaHome == "" {
			if link, err := filepath.EvalSymlinks("/usr/bin/java"); err == nil {
				javaHome = path.Dir(link)
			}
		}
		if javaHome != "" {
			if err := lyra.Java.SetPath(javaHome); err != nil {
				log.Fatal(err)
			}
		}
	}

	if !lyra.Java.IsInstalled() {
		log.Fatal("no JDK present on system")
	}

	if err := lyra.Command.Run(os.Args...); err != nil {
		log.Fatal(err)
	}

	if err := lyra.GetCurrentProject().Save(); err != nil {
		log.Fatal(err)
	}
}
