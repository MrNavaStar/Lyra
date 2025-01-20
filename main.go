package main

import (
	"github.com/mrnavastar/lyra/lyra"
	"log"
	"os"
)

func main() {
	if lyra.Java.GetPath() == "" {
		if err := lyra.Java.SetPath(os.Getenv("JAVA_HOME")); err != nil {
			log.Fatal(err)
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
