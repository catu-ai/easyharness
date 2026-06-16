package helpassets

import (
	"embed"
	"io/fs"
	"strings"
)

var (
	//go:embed *.md repo/*.md
	embeddedAssets embed.FS
)

func Read(name string) (string, error) {
	data, err := fs.ReadFile(embeddedAssets, name)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(data), "\n") + "\n", nil
}
