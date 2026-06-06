package web

import (
	"embed"
	"io/fs"
)

var (
	//go:embed dist fallback/index.html
	embeddedFiles embed.FS
	distFiles     fs.FS
	hasDistIndex  bool
)

func init() {
	sub, err := fs.Sub(embeddedFiles, "dist")
	if err != nil {
		return
	}
	distFiles = sub
	_, err = fs.Stat(distFiles, "index.html")
	hasDistIndex = err == nil
}

func HasDistIndex() bool {
	return hasDistIndex
}

func HasDistAsset(name string) bool {
	if distFiles == nil {
		return false
	}
	_, err := fs.Stat(distFiles, name)
	return err == nil
}

func ReadDistAsset(name string) ([]byte, error) {
	return fs.ReadFile(distFiles, name)
}

func FallbackIndexHTML() ([]byte, error) {
	return embeddedFiles.ReadFile("fallback/index.html")
}
