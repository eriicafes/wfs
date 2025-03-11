package wfs_test

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/eriicafes/wfs"
)

var fileSystems = []struct {
	name string
	fsys func(fstest.MapFS) (fs wfs.FS, base string, cleanup func(), err error)
}{
	{"OS FS", func(fsys fstest.MapFS) (wfs.FS, string, func(), error) {
		dir, err := os.MkdirTemp("", "testdata")
		if err != nil {
			return nil, "", nil, err
		}

		for name, file := range fsys {
			name = filepath.Join(dir, name)
			if file.Mode == 0 {
				file.Mode = os.ModePerm
			}
			if file.Mode.IsDir() {
				err = os.MkdirAll(name, file.Mode)
			} else {
				err = os.MkdirAll(filepath.Dir(name), 0755)
				if err != nil {
					break
				}
				err = os.WriteFile(name, file.Data, file.Mode)
			}
			if err != nil {
				break
			}
		}

		cleanup := func() { os.RemoveAll(dir) }
		return wfs.OS(), dir, cleanup, err
	}},
	{"Map FS", func(fsys fstest.MapFS) (wfs.FS, string, func(), error) {
		return wfs.Map(fsys), "", func() {}, nil
	}},
}

func TestFileReadAt(t *testing.T) {
	for _, tt := range fileSystems {
		t.Run(tt.name, func(t *testing.T) {
			fsys, base, cleanup, err := tt.fsys(fstest.MapFS{
				"testfile": &fstest.MapFile{
					Data: []byte("Hello, World!"),
				}},
			)
			if err != nil {
				t.Fatalf("failed to create file system: %v", err)
			}
			defer cleanup()

			filePath := filepath.Join(base, "testfile")
			f, err := fsys.OpenFile(filePath, os.O_RDONLY, 0)
			if err != nil {
				t.Fatalf("failed to open file: %v", err)
			}
			defer f.Close()

			buf := make([]byte, 5)
			if _, err := f.ReadAt(buf, 7); err != nil {
				t.Fatalf("ReadAt failed: %v", err)
			}

			if string(buf) != "World" {
				t.Errorf("expected 'World', got %q", buf)
			}
		})
	}
}

func TestFileWriteAt(t *testing.T) {
	for _, tt := range fileSystems {
		t.Run(tt.name, func(t *testing.T) {
			fsys, base, cleanup, err := tt.fsys(fstest.MapFS{
				"testfile": &fstest.MapFile{
					Data: []byte("Hello, World!"),
				},
			})
			if err != nil {
				t.Fatalf("failed to create file system: %v", err)
			}
			defer cleanup()

			filePath := filepath.Join(base, "testfile")
			f, err := fsys.OpenFile(filePath, os.O_WRONLY, 0)
			if err != nil {
				t.Fatalf("failed to open file: %v", err)
			}
			defer f.Close()

			data := []byte("There")
			if _, err := f.WriteAt(data, 7); err != nil {
				t.Fatalf("WriteAt failed: %v", err)
			}

			b, err := fs.ReadFile(fsys, filePath)
			if err != nil || string(b) != "Hello, There!" {
				t.Errorf("expected 'Hello, There!', got %q err: %v", b, err)
			}
		})
	}
}

func TestFileSeek(t *testing.T) {
	for _, tt := range fileSystems {
		t.Run(tt.name, func(t *testing.T) {
			fsys, base, cleanup, err := tt.fsys(fstest.MapFS{
				"testfile": &fstest.MapFile{
					Data: []byte("Hello, World!"),
				},
			})
			if err != nil {
				t.Fatalf("failed to create file system: %v", err)
			}
			defer cleanup()

			filePath := filepath.Join(base, "testfile")
			f, err := fsys.OpenFile(filePath, os.O_RDONLY, 0)
			if err != nil {
				t.Fatalf("failed to open file: %v", err)
			}
			defer f.Close()

			_, err = f.Seek(7, io.SeekStart)
			if err != nil {
				t.Errorf("Seek failed: %v", err)
			}

			b, err := io.ReadAll(f)
			if err != nil || string(b) != "World!" {
				t.Errorf("expected 'World!', got %q err: %v", b, err)
			}
		})
	}
}

func TestFileTruncate(t *testing.T) {
	for _, tt := range fileSystems {
		t.Run(tt.name, func(t *testing.T) {
			fsys, base, cleanup, err := tt.fsys(fstest.MapFS{
				"testfile": &fstest.MapFile{
					Data: []byte("Hello, World!"),
				},
			})
			if err != nil {
				t.Fatalf("failed to create file system: %v", err)
			}
			defer cleanup()

			filePath := filepath.Join(base, "testfile")
			f, err := fsys.OpenFile(filePath, os.O_WRONLY, 0)
			if err != nil {
				t.Fatalf("failed to open file: %v", err)
			}
			defer f.Close()

			if err := f.Truncate(5); err != nil {
				t.Fatalf("Truncate failed: %v", err)
			}

			b, err := fs.ReadFile(fsys, filePath)
			if err != nil || string(b) != "Hello" {
				t.Errorf("expected 'Hello', got %q err %v", b, err)
			}
		})
	}
}

func TestFileName(t *testing.T) {
	for _, tt := range fileSystems {
		t.Run(tt.name, func(t *testing.T) {
			fsys, base, cleanup, err := tt.fsys(fstest.MapFS{"testfile": &fstest.MapFile{}})
			if err != nil {
				t.Fatalf("failed to create file system: %v", err)
			}
			defer cleanup()

			filePath := filepath.Join(base, "testfile")
			f, err := fsys.OpenFile(filePath, os.O_RDONLY, 0)
			if err != nil {
				t.Fatalf("failed to open file: %v", err)
			}
			defer f.Close()

			if f.Name() != filePath {
				t.Errorf("expected name %q, got %q", filePath, f.Name())
			}
		})
	}
}

func TestOpenFile(t *testing.T) {
	for _, tt := range fileSystems {
		t.Run(tt.name, func(t *testing.T) {
			fsys, base, cleanup, err := tt.fsys(fstest.MapFS{
				"testfile": &fstest.MapFile{
					Data: []byte("Hello, World!"),
				},
			})
			if err != nil {
				t.Fatalf("failed to create file system: %v", err)
			}
			defer cleanup()

			tests := []struct {
				name         string
				flag         int
				shouldCreate bool
				shouldOpen   bool
				shouldRead   bool
				shouldWrite  bool
			}{
				{"ReadOnly", os.O_RDONLY, false, true, true, false},
				{"WriteOnly", os.O_WRONLY, false, true, false, true},
				{"ReadWrite", os.O_RDWR, false, true, true, true},
				{"Append", os.O_WRONLY | os.O_APPEND, false, true, false, true},
				{"Truncate", os.O_WRONLY | os.O_TRUNC, false, true, false, true},
				{"Create", os.O_WRONLY | os.O_CREATE, true, true, false, true},
			}

			for _, tc := range tests {
				t.Run(tc.name, func(t *testing.T) {
					// open missing file
					filePath := filepath.Join(base, "missingfile")
					f, err := fsys.OpenFile(filePath, tc.flag, fs.ModePerm)
					if err != nil && tc.shouldCreate {
						t.Errorf("OpenFile '%s' failed: expected to create %v", tc.name, err)
					}
					if err == nil && !tc.shouldCreate {
						t.Errorf("OpenFile '%s' failed: expected to fail create", tc.name)
					}
					if err == nil {
						f.Close()
					}

					// open file
					filePath = filepath.Join(base, "testfile")
					f, err = fsys.OpenFile(filePath, tc.flag, fs.ModePerm)
					if err != nil && tc.shouldOpen {
						t.Fatalf("OpenFile '%s' failed: expected to open %v", tc.name, err)
					}
					if err == nil && !tc.shouldOpen {
						t.Fatalf("OpenFile '%s' failed: expected to fail open", tc.name)
					}
					defer f.Close()

					if !tc.shouldOpen {
						return
					}

					// read file
					b, err := io.ReadAll(f)
					expected := "Hello, World!"
					if err != nil && tc.shouldRead {
						t.Errorf("OpenFile '%s' failed: expected to read %q got %q, err: %v", tc.name, expected, string(b), err)
					}
					if tc.shouldRead && string(b) != expected {
						t.Errorf("OpenFile '%s' failed: expected to read %q got %q, err: %v", tc.name, expected, string(b), err)
					}
					if err == nil && !tc.shouldRead {
						t.Errorf("OpenFile '%s' failed: expected to fail read", tc.name)
					}

					// write to file
					n, err := f.Write([]byte(expected))
					if err != nil && tc.shouldWrite {
						t.Errorf("OpenFile '%s' failed: expected to write: %v", tc.name, err)
					}
					if tc.shouldWrite && n != len(expected) {
						t.Errorf("OpenFile '%s' failed: expected to write %d got %d, err: %v", tc.name, len(expected), n, err)
					}
					if err == nil && !tc.shouldWrite {
						t.Errorf("OpenFile '%s' failed: expected to fail write", tc.name)
					}
				})
			}
		})
	}
}

func TestRename(t *testing.T) {
	for _, tt := range fileSystems {
		t.Run(tt.name, func(t *testing.T) {
			fsys, base, cleanup, err := tt.fsys(fstest.MapFS{
				"oldname":        &fstest.MapFile{},
				"oldnested/file": &fstest.MapFile{},
				"oldemptydir":    &fstest.MapFile{Mode: fs.ModeDir | 0755},
			})
			if err != nil {
				t.Fatalf("failed to create file system: %v", err)
			}
			defer cleanup()

			// rename file
			oldPath := filepath.Join(base, "oldname")
			newPath := filepath.Join(base, "newname")
			if err := fsys.Rename(oldPath, newPath); err != nil {
				t.Fatalf("Rename failed: %v", err)
			}
			if _, err := fs.Stat(fsys, newPath); err != nil {
				t.Errorf("Renamed file should exist: %v", err)
			}
			if _, err := fs.Stat(fsys, oldPath); err == nil {
				t.Errorf("Original file should no longer exist")
			}

			// rename dir with contents
			oldPath = filepath.Join(base, "oldnested")
			newPath = filepath.Join(base, "newnested")
			if err := fsys.Rename(oldPath, newPath); err != nil {
				t.Fatalf("Rename failed: %v", err)
			}
			if _, err := fs.Stat(fsys, newPath); err != nil {
				t.Errorf("Renamed dir should exist: %v", err)
			}
			newFilePath := filepath.Join(newPath, "file")
			if _, err := fs.Stat(fsys, newFilePath); err != nil {
				t.Errorf("Renamed dir file should exist: %v", err)
			}
			if _, err := fs.Stat(fsys, oldPath); err == nil {
				t.Errorf("Original dir should no longer exist")
			}
			oldFilePath := filepath.Join(oldPath, "file")
			if _, err := fs.Stat(fsys, oldFilePath); err == nil {
				t.Errorf("Original dir file should no longer exist")
			}
		})
	}
}

func TestRemove(t *testing.T) {
	for _, tt := range fileSystems {
		t.Run(tt.name, func(t *testing.T) {
			fsys, base, cleanup, err := tt.fsys(fstest.MapFS{
				"testfile":        &fstest.MapFile{},
				"testdir/file":    &fstest.MapFile{},
				"emptydir":        &fstest.MapFile{Mode: fs.ModeDir | 0755},
				"nested/dir":      &fstest.MapFile{Mode: fs.ModeDir | 0755},
				"nested/dir/file": &fstest.MapFile{},
			})
			if err != nil {
				t.Fatalf("failed to create file system: %v", err)
			}
			defer cleanup()

			// remove file
			filePath := filepath.Join(base, "testfile")
			if err := fsys.Remove(filePath); err != nil {
				t.Fatalf("Remove should succeed for file: %v", err)
			}
			if _, err := fs.Stat(fsys, filePath); err == nil {
				t.Errorf("Removed file should no longer exist")
			}

			// attempt to remove non-empty directory
			dirPath := filepath.Join(base, "testdir")
			if err := fsys.Remove(dirPath); err == nil {
				t.Errorf("Remove should fail for non-empty directory")
			}
			if _, err := fs.Stat(fsys, dirPath); err != nil {
				t.Errorf("Non-empty directory should still exist")
			}

			// remove empty directory
			emptyDirPath := filepath.Join(base, "emptydir")
			if err := fsys.Remove(emptyDirPath); err != nil {
				t.Fatalf("Remove should succeed for empty directory: %v", err)
			}
			if _, err := fs.Stat(fsys, emptyDirPath); err == nil {
				t.Errorf("Removed empty directory should no longer exist")
			}
		})
	}
}

func TestRemoveAll(t *testing.T) {
	for _, tt := range fileSystems {
		t.Run(tt.name, func(t *testing.T) {
			fsys, base, cleanup, err := tt.fsys(fstest.MapFS{
				"dir/file":        &fstest.MapFile{},
				"dir/nested/file": &fstest.MapFile{},
			})
			if err != nil {
				t.Fatalf("failed to create file system: %v", err)
			}
			defer cleanup()

			dirPath := filepath.Join(base, "dir")
			if err := fsys.RemoveAll(dirPath); err != nil {
				t.Fatalf("RemoveAll failed: %v", err)
			}
			if _, err := fs.Stat(fsys, dirPath); err == nil {
				t.Errorf("Stat should fail for removed directory")
			}

			nestedFilePath := filepath.Join(dirPath, "file")
			if _, err := fs.Stat(fsys, nestedFilePath); err == nil {
				t.Errorf("Stat should fail for removed nested file")
			}
			nestedDirPath := filepath.Join(dirPath, "nested")
			if _, err := fs.Stat(fsys, nestedDirPath); err == nil {
				t.Errorf("Stat should fail for removed nested directory")
			}
			nestedDirFilePath := filepath.Join(dirPath, "nested", "file")
			if _, err := fs.Stat(fsys, nestedDirFilePath); err == nil {
				t.Errorf("Stat should fail for removed nested directory file")
			}
		})
	}
}

func TestMkdir(t *testing.T) {
	for _, tt := range fileSystems {
		t.Run(tt.name, func(t *testing.T) {
			fsys, base, cleanup, err := tt.fsys(fstest.MapFS{})
			if err != nil {
				t.Fatalf("failed to create file system: %v", err)
			}
			defer cleanup()

			dirPath := filepath.Join(base, "testdir")
			if err := fsys.Mkdir(dirPath, 0755); err != nil {
				t.Fatalf("Mkdir failed: %v", err)
			}

			if _, err := fs.Stat(fsys, dirPath); err != nil {
				t.Errorf("Stat failed for created directory: %v", err)
			}

			parentChildPath := filepath.Join(base, "parent", "child")
			if err := fsys.Mkdir(parentChildPath, 0755); err == nil {
				t.Errorf("Mkdir should fail when trying to create a nested directory: %v", err)
			}
		})
	}
}

func TestMkdirAll(t *testing.T) {
	for _, tt := range fileSystems {
		t.Run(tt.name, func(t *testing.T) {
			fsys, base, cleanup, err := tt.fsys(fstest.MapFS{})
			if err != nil {
				t.Fatalf("failed to create file system: %v", err)
			}
			defer cleanup()

			dirPath := filepath.Join(base, "parent", "child")
			if err := fsys.MkdirAll(dirPath, 0755); err != nil {
				t.Fatalf("MkdirAll failed: %v", err)
			}

			if _, err := fs.Stat(fsys, dirPath); err != nil {
				t.Errorf("Stat failed for created directory structure: %v", err)
			}
		})
	}
}

func TestCreate(t *testing.T) {
	for _, tt := range fileSystems {
		t.Run(tt.name, func(t *testing.T) {
			fsys, base, cleanup, err := tt.fsys(fstest.MapFS{})
			if err != nil {
				t.Fatalf("failed to create file system: %v", err)
			}
			defer cleanup()

			filePath := filepath.Join(base, "testfile")
			// create file
			f, err := wfs.Create(fsys, filePath)
			if err != nil {
				t.Fatalf("failed to create file: %v", err)
			}
			defer f.Close()

			// truncate file
			f, err = wfs.Create(fsys, filePath)
			if err != nil {
				t.Fatalf("failed to create file: %v", err)
			}
			defer f.Close()
		})
	}
}

func TestWriteFile(t *testing.T) {
	for _, tt := range fileSystems {
		t.Run(tt.name, func(t *testing.T) {
			fsys, base, cleanup, err := tt.fsys(fstest.MapFS{})
			if err != nil {
				t.Fatalf("failed to create file system: %v", err)
			}
			defer cleanup()

			filePath := filepath.Join(base, "testfile")
			// create file
			data := []byte("Hello")
			err = wfs.WriteFile(fsys, filePath, data, 0755)
			if err != nil {
				t.Fatalf("failed to write file: %v", err)
			}
			b, err := fs.ReadFile(fsys, filePath)
			if err != nil || string(b) != "Hello" {
				t.Errorf("expected 'Hello', got %q err: %v", b, err)
			}

			// replace file
			data = []byte("World")
			err = wfs.WriteFile(fsys, filePath, data, 0755)
			if err != nil {
				t.Fatalf("failed to write file: %v", err)
			}
			b, err = fs.ReadFile(fsys, filePath)
			if err != nil || string(b) != "World" {
				t.Errorf("expected 'World', got %q err: %v", b, err)
			}
		})
	}
}
