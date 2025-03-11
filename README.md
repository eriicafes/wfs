# WFS

### Writable filesystem abstraction in Go.

WFS provides an interface that abstracts the OS filesystem, enabling support for custom writable filesystems and improved testing.

## Why Use WFS?

The Go standard library provides powerful filesystem primitives, but handling files across different storage backends can be cumbersome. While `fs.FS` is a flexible and useful abstraction, it is inherently read-only by design. `wfs` extends this concept by providing a writable filesystem interface, allowing seamless integration with custom backends, such as in-memory filesystems for testing.

## Installation

```sh
go get github.com/eriicafes/wfs
```

## Usage

`wfs` provides interfaces and top-level functions for working with files and directories.

Create an OS writable filesystem or an in-memory writable filesystem for use in tests.

```go
// os filesystem
fsys := wfs.OS()

// in-memory filesystem
fsys := wfs.Map(fstest.MapFS{})
```

## Interfaces

### FS

A `wfs.FS` implements `fs.FS`, `wfs.FileFS` and `wfs.DirFS`.
A `wfs.FS` implementation can read, write and create files and directories.

```go
type FS interface {
    fs.FS
    FileFS
    DirFS
}
```

### FileFS

A `wfs.FileFS` implementation can create, read, write and delete files.

```go
type FileFS interface {
    OpenFile(name string, flag int, perm fs.FileMode) (File, error)
    Rename(oldpath, newpath string) error
    Remove(name string) error
    RemoveAll(path string) error
}
```

### DirFS

A `wfs.DirFS` implementation can create directories.

```go
type DirFS interface {
    Mkdir(name string, perm fs.FileMode) error
    MkdirAll(path string, perm fs.FileMode) error
}
```

### File

A `wfs.File` implementation extends `fs.File` with additional methods.

```go
type File interface {
    fs.File
    io.WriteSeeker
    io.ReaderAt
    io.WriterAt
    Truncate(size int64) error
    Name() string
}
```

## Top-level Functions

### Create

Creates a new file or truncates an existing one, returning a writable file handle.

```go
f, err := wfs.Create(fsys, "filename")
```

### WriteFile

Writes data to a file, creating or replacing it while preserving file permissions.

```go
err := wfs.WriteFile(fsys, "filename", []byte(`data`), fs.ModePerm)
```
