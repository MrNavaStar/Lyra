// Package application makes the output jar executable if the source code contains a main method.
//
// Note that this plugin will throw an error if the source code contains more than one main method.
package application

import (
	"fmt"
	"github.com/mrnavastar/babe/babe"
	"github.com/mrnavastar/lyra/lyra"
	"strings"
)

func init() {
	lyra.Build.Hooks.PackageClass(func(jar babe.Jar, class *babe.Class) error {
		if class.HasMainMethod() {
			if lyra.Build.HasManifestEntry("Main-Class") {
				return fmt.Errorf("module: %s has too many main method declarations - only one allowed", jar.Name)
			}
			lyra.Build.AddManifestEntry("Main-Class", strings.ReplaceAll(class.GetClassName(), "/", "."))
		}
		return nil
	})
}
