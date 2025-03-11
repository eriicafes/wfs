package wfs

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"syscall"
	"testing/fstest"
	"time"
)

// mapFs mirrors os filesystem using [fstest.MapFS] and a [bytes.Reader].
type mapFs struct{ fstest.MapFS }

func Map(fs fstest.MapFS) FS {
	return &mapFs{fs}
}

func (f *mapFs) OpenFile(name string, flag int, perm fs.FileMode) (File, error) {
	file, err := f.Open(name)
	// create file if it does not exist and os.0_CREATE flag is present
	if errors.Is(err, fs.ErrNotExist) && flag&os.O_CREATE != 0 {
		// use perm only when creating new files
		f.MapFS[name] = &fstest.MapFile{Mode: perm}
		file, err = f.Open(name)
	}
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	// return an error if write flags are used to open a directory
	if info.IsDir() && flag&(os.O_WRONLY|os.O_RDWR) != 0 {
		return nil, &os.PathError{Op: "open", Path: name, Err: syscall.EISDIR}
	}
	// read file contents into bytes reader
	b, _ := io.ReadAll(file)
	mfile := &mapFsFile{
		File:   file,
		mfile:  f.MapFS[name],
		name:   name,
		flag:   flag,
		perm:   info.Mode(),
		reader: bytes.NewReader(b),
	}
	// truncate file if O_TRUNC flag is present
	if flag&os.O_TRUNC != 0 {
		mfile.Truncate(0)
	}
	// move file cursor to end if O_APPEND flag is present
	if flag&os.O_APPEND != 0 {
		mfile.Seek(0, io.SeekEnd)
	}
	return mfile, nil
}

func (f *mapFs) Stat(name string) (fs.FileInfo, error) {
	file, err := f.Open(name)
	if err != nil {
		return nil, &os.PathError{Op: "stat", Path: name, Err: err}
	}
	return file.Stat()
}

func (f *mapFs) Rename(oldpath, newpath string) error {
	oldinfo, err := f.Stat(oldpath)
	if err != nil {
		if pe, ok := err.(*fs.PathError); ok {
			err = pe.Err
		}
		return &os.LinkError{Op: "rename", Old: oldpath, New: newpath, Err: err}
	}
	if oldpath == newpath {
		return &os.LinkError{Op: "rename", Old: oldpath, New: newpath, Err: syscall.EEXIST}
	}
	// return an error if newpath is a directory
	newinfo, err := f.Stat(newpath)
	if err == nil && newinfo.IsDir() {
		return &os.LinkError{Op: "rename", Old: oldpath, New: newpath, Err: syscall.EEXIST}
	}

	// check if new parent directory exists
	dir, _ := path.Split(newpath)
	if dir != "" {
		dirinfo, err := f.Stat(dir)
		if err != nil {
			return &os.LinkError{Op: "rename", Old: oldpath, New: newpath, Err: syscall.ENOENT}
		} else if !dirinfo.IsDir() {
			return &os.LinkError{Op: "rename", Old: oldpath, New: newpath, Err: syscall.ENOTDIR}
		}
	}

	movepath := true
	if oldinfo.IsDir() {
		// for a directory move each file that exists under oldpath
		for name := range f.MapFS {
			if strings.HasPrefix(name, oldpath) {
				newname := strings.Replace(name, oldpath, newpath, 1)
				f.MapFS[newname] = f.MapFS[name]
				delete(f.MapFS, name)
				movepath = false
			}
		}
	}
	// movepath remains true if oldpath is a file or an empty directory
	// an empty directory will exist explicitly as a map entry in [fstest.MapFS]
	if movepath {
		f.MapFS[newpath] = f.MapFS[oldpath]
		delete(f.MapFS, oldpath)
	}
	return nil
}

func (f *mapFs) Remove(name string) error {
	_, ok := f.MapFS[name]
	if !ok {
		return &fs.PathError{Op: "remove", Path: "name", Err: syscall.ENOENT}
	}
	entries, _ := fs.ReadDir(f, name)
	if len(entries) > 0 {
		return &fs.PathError{Op: "remove", Path: "name", Err: syscall.ENOTEMPTY}
	}
	delete(f.MapFS, name)
	return nil
}

func (f *mapFs) RemoveAll(path string) error {
	for name := range f.MapFS {
		if strings.HasPrefix(name, path) {
			delete(f.MapFS, name)
		}
	}
	return nil
}

func (f *mapFs) Mkdir(name string, perm fs.FileMode) error {
	dir, _ := path.Split(name)
	if dir != "" {
		info, err := f.Stat(dir)
		if err != nil {
			return &os.PathError{Op: "mkdir", Path: name, Err: syscall.ENOENT}
		}
		if !info.IsDir() {
			return &os.PathError{Op: "mkdir", Path: name, Err: syscall.ENOTDIR}
		}
	}
	f.MapFS[name] = &fstest.MapFile{
		Mode:    perm,
		ModTime: time.Now(),
	}
	return nil
}

func (f *mapFs) MkdirAll(name string, perm fs.FileMode) error {
	dir, _ := path.Split(name)
	info, err := f.Stat(dir)
	if err != nil {
		f.MapFS[name] = &fstest.MapFile{
			Mode:    perm,
			ModTime: time.Now(),
		}
		return nil
	}
	if !info.IsDir() {
		return &os.PathError{Op: "mkdir", Path: name, Err: syscall.ENOTDIR}
	}
	return nil
}

type mapFsFile struct {
	fs.File
	mfile  *fstest.MapFile
	name   string
	flag   int
	perm   fs.FileMode
	reader *bytes.Reader
}

func (f *mapFsFile) Name() string {
	return f.name
}

func (f *mapFsFile) Read(b []byte) (n int, err error) {
	if f.perm.IsDir() {
		return 0, &fs.PathError{Op: "read", Path: f.name, Err: syscall.EISDIR}
	}
	if f.flag&(os.O_RDONLY|os.O_RDWR) == 0 && f.flag != 0 {
		return 0, &fs.PathError{Op: "read", Path: f.name, Err: syscall.EBADF}
	}

	return f.reader.Read(b)
}

func (f *mapFsFile) ReadAt(b []byte, off int64) (n int, err error) {
	if f.perm.IsDir() {
		return 0, &fs.PathError{Op: "read", Path: f.name, Err: syscall.EISDIR}
	}
	if f.flag&(os.O_RDONLY|os.O_RDWR) == 0 && f.flag != 0 {
		return 0, &fs.PathError{Op: "read", Path: f.name, Err: syscall.EBADF}
	}

	if off < 0 || off > int64(f.reader.Size()) {
		return 0, &fs.PathError{Op: "read", Path: f.name, Err: fs.ErrInvalid}
	}
	return f.reader.ReadAt(b, off)
}

func (f *mapFsFile) Seek(offset int64, whence int) (int64, error) {
	if f.perm.IsDir() {
		return 0, &fs.PathError{Op: "seek", Path: f.name, Err: syscall.EISDIR}
	}

	n, err := f.reader.Seek(offset, whence)
	if err != nil {
		err = &fs.PathError{Op: "seek", Path: f.name, Err: err}
	}
	return n, err
}

func (f *mapFsFile) Write(b []byte) (n int, err error) {
	if f.perm.IsDir() || f.flag&(os.O_WRONLY|os.O_RDWR) == 0 {
		return 0, &fs.PathError{Op: "write", Path: f.name, Err: syscall.EBADF}
	}

	pos, _ := f.Seek(0, io.SeekCurrent)
	end := int(pos) + len(b)
	// expand the slice if necessary
	if end > len(f.mfile.Data) {
		f.mfile.Data = append(f.mfile.Data, make([]byte, end-len(f.mfile.Data))...)
	}
	n = copy(f.mfile.Data[pos:], b)
	f.reset()
	// move cursor based on amount written
	f.reader.Seek(int64(n), io.SeekCurrent)
	return
}

func (f *mapFsFile) WriteAt(b []byte, off int64) (n int, err error) {
	if f.flag&os.O_APPEND != 0 {
		return 0, errors.New("invalid use of WriteAt on file opened with O_APPEND")
	}
	if f.perm.IsDir() || f.flag&(os.O_WRONLY|os.O_RDWR) == 0 {
		return 0, &fs.PathError{Op: "write", Path: f.name, Err: syscall.EBADF}
	}

	if off < 0 {
		err = &fs.PathError{Op: "writeat", Path: f.name, Err: errors.New("negative offset")}
		return
	}
	end := int(off) + len(b)
	// expand the slice if necessary
	if end > len(f.mfile.Data) {
		f.mfile.Data = append(f.mfile.Data, make([]byte, end-len(f.mfile.Data))...)
	}
	n = copy(f.mfile.Data[off:], b)
	f.reset()
	return
}

func (f *mapFsFile) Truncate(size int64) error {
	if f.perm.IsDir() || f.flag&(os.O_WRONLY|os.O_RDWR) == 0 {
		return &fs.PathError{Op: "truncate", Path: f.name, Err: syscall.EINVAL}
	}

	if size < 0 {
		return nil
	}
	curr := int64(len(f.mfile.Data))
	if size > curr {
		// expand the slice with zero bytes
		f.mfile.Data = append(f.mfile.Data, make([]byte, size-curr)...)
	} else {
		f.mfile.Data = f.mfile.Data[:size]
	}
	f.reset()
	return nil
}

// reset updates the reader bytes reference while maintaining the cursor position.
func (f *mapFsFile) reset() {
	pos, _ := f.reader.Seek(0, io.SeekCurrent)
	f.reader.Reset(f.mfile.Data)
	f.reader.Seek(pos, io.SeekStart)
}
