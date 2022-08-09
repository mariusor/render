//go:build go1.16
// +build go1.16

package render

import (
	"embed"
	"io"
	"io/fs"
)

// EmbedFileSystem implements FileSystem on top of an embed.FS
type EmbedFileSystem struct {
	embed.FS
}

var _ fs.FS = &EmbedFileSystem{}

func (e *EmbedFileSystem) Walk(root string, walkFn fs.WalkDirFunc) error {
	return fs.WalkDir(e.FS, root, func(path string, d fs.DirEntry, err error) error {
		if d == nil {
			return nil
		}
		return walkFn(path, d, err)
	})
}

type tmplFS struct {
	fs.FS
}

func (tfs tmplFS) Walk(root string, walkFn fs.WalkDirFunc) error {
	return fs.WalkDir(tfs, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return walkFn(path, d, err)
	})
}

func (tfs tmplFS) ReadFile(filename string) ([]byte, error) {
	f, err := tfs.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

// FS converts io/fs.FS to FileSystem
func FS(oriFS fs.FS) fs.FS {
	return tmplFS{oriFS}
}
