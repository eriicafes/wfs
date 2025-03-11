package wfs

import (
	"io/fs"
	"os"
)

type osFs struct{}

// OS returns a os writable file system.
func OS() FS {
	return osFs{}
}

func (osFs) Open(name string) (fs.File, error) {
	return os.Open(name)
}

func (osFs) OpenFile(name string, flag int, perm fs.FileMode) (File, error) {
	return os.OpenFile(name, flag, perm)
}

// Stat implements [fs.StatFS] for osFS.
func (osFs) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

func (osFs) Rename(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}

func (osFs) Remove(name string) error {
	return os.Remove(name)
}

func (osFs) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

func (osFs) Mkdir(name string, perm fs.FileMode) error {
	return os.Mkdir(name, perm)
}

func (osFs) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}
