package wfs

import (
	"io"
	"io/fs"
	"os"
)

type File interface {
	fs.File
	io.WriteSeeker
	io.ReaderAt
	io.WriterAt

	// Truncate changes the size of the file.
	// It does not change the I/O offset.
	// If there is an error, it will be of type [*fs.PathError].
	Truncate(size int64) error

	// Name returns the name of the file as presented to Open.
	//
	// It is safe to call Name after [Close].
	Name() string
}

type FS interface {
	fs.FS
	FileFS
	DirFS
}

type FileFS interface {
	// OpenFile is the generalized open call; most users will use Open
	// or Create instead. It opens the named file with specified flag
	// ([os.O_RDONLY] etc.). If the file does not exist, and the [os.O_CREATE] flag
	// is passed, it is created with mode perm (before umask);
	// the containing directory must exist. If successful,
	// methods on the returned File can be used for I/O.
	// If there is an error, it will be of type [*fs.PathError].
	OpenFile(name string, flag int, perm fs.FileMode) (File, error)

	// Stat returns a FileInfo describing the file.
	// If there is an error, it should be of type [*fs.PathError].
	Stat(name string) (fs.FileInfo, error)

	// Rename renames (moves) oldpath to newpath.
	// If newpath already exists and is not a directory, Rename replaces it.
	// If newpath already exists and is a directory, Rename returns an error.
	// OS-specific restrictions may apply when oldpath and newpath are in different directories.
	// Even within the same directory, on non-Unix platforms Rename is not an atomic operation.
	// If there is an error, it will be of type [*os.LinkError].
	Rename(oldpath, newpath string) error

	// Remove removes the named file or (empty) directory.
	// If there is an error, it will be of type [*fs.PathError].
	Remove(name string) error

	// RemoveAll removes path and any children it contains.
	// It removes everything it can but returns the first error
	// it encounters. If the path does not exist, RemoveAll
	// returns nil (no error).
	// If there is an error, it will be of type [*fs.PathError].
	RemoveAll(path string) error
}

type DirFS interface {
	// Mkdir creates a new directory with the specified name and permission
	// bits (before umask).
	// If there is an error, it will be of type [*fs.PathError].
	Mkdir(name string, perm fs.FileMode) error

	// MkdirAll creates a directory named path,
	// along with any necessary parents, and returns nil,
	// or else returns an error.
	// The permission bits perm (before umask) are used for all
	// directories that MkdirAll creates.
	// If path is already a directory, MkdirAll does nothing
	// and returns nil.
	MkdirAll(path string, perm fs.FileMode) error
}

// Create creates or truncates the named file. If the file already exists,
// it is truncated. If the file does not exist, it is created with mode 0o666
// (before umask). If successful, methods on the returned File can
// be used for I/O; the associated file descriptor has mode [os.O_RDWR].
// The directory containing the file must already exist.
// If there is an error, it will be of type [*fs.PathError].
func Create(fs FileFS, name string) (File, error) {
	return fs.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
}

// WriteFile writes data to the named file, creating it if necessary.
// If the file does not exist, WriteFile creates it with permissions perm (before umask);
// otherwise WriteFile truncates it before writing, without changing permissions.
// Since WriteFile requires multiple system calls to complete, a failure mid-operation
// can leave the file in a partially written state.
func WriteFile(fs FileFS, name string, data []byte, perm fs.FileMode) error {
	f, err := fs.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if err1 := f.Close(); err1 != nil && err == nil {
		err = err1
	}
	return err
}
