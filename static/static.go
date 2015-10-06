package static

import (
	"net/http"

	"github.com/elazarl/go-bindata-assetfs"
)

//go:generate go run ../contrib/generate-js.go -dir scripts/ -o scripts_gen/drone.min.js
//go:generate go run ../contrib/generate-api-docs.go -input ../docs/swagger.yml -template ../template/amber/swagger.amber -output docs_gen/api/index.html
//go:generate go run ../contrib/generate-docs.go -input ../docs/build/README.md -template ../template/amber/docs.amber -output docs_gen/build/

//go:generate sassc --style compact styles/style.sass styles_gen/style.css
//go:generate go-bindata-assetfs -ignore "\\.go" -pkg static -o static_gen.go ./...

func FileSystem() http.FileSystem {
	fs := &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, Prefix: ""}
	return &binaryFileSystem{
		fs,
	}
}

type binaryFileSystem struct {
	fs http.FileSystem
}

func (b *binaryFileSystem) Open(name string) (http.File, error) {
	return b.fs.Open(name[1:])
}
