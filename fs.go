package rofuse

import "syscall"

// Filesystem is the interface that read-only filesystems must implement.
// All methods operate on inode numbers, not paths.
// Methods should be goroutine-safe as they may be called concurrently.
type Filesystem interface {
	// Init is called during FUSE_INIT to allow filesystem initialization.
	// The Config contains negotiated protocol parameters.
	Init(ctx Context, config *Config) error

	// Destroy is called during FUSE_DESTROY when unmounting.
	Destroy(ctx Context)

	// Lookup finds a directory entry by name within a parent directory.
	// Returns the entry with inode number, attributes, and cache timeouts.
	// Should return syscall.ENOENT if not found.
	Lookup(ctx Context, parent Inode, name string) (*Entry, error)

	// GetAttr retrieves attributes for an inode.
	// If fh is non-nil, it's a file handle from a previous Open.
	GetAttr(ctx Context, ino Inode, fh *FileHandle) (*Attr, error)

	// ReadLink reads the target of a symbolic link.
	ReadLink(ctx Context, ino Inode) (string, error)

	// Open opens a file and returns a file handle.
	// flags contains O_RDONLY, O_NONBLOCK, etc.
	Open(ctx Context, ino Inode, flags uint32) (*OpenResponse, error)

	// Read reads data from an open file.
	// Returns data read. May return less than size bytes.
	Read(ctx Context, ino Inode, fh FileHandle, offset int64, size uint32) ([]byte, error)

	// Release closes a file handle opened by Open.
	Release(ctx Context, ino Inode, fh FileHandle) error

	// OpenDir opens a directory for reading.
	OpenDir(ctx Context, ino Inode, flags uint32) (*OpenResponse, error)

	// ReadDir reads directory entries.
	// offset is the position in the directory stream (from previous DirEntry.Offset).
	// Returns entries that fit within size bytes when serialized.
	ReadDir(ctx Context, ino Inode, fh FileHandle, offset int64, size uint32) ([]DirEntry, error)

	// ReadDirPlus reads directory entries with attributes (READDIRPLUS).
	// This combines ReadDir + Lookup for better performance.
	ReadDirPlus(ctx Context, ino Inode, fh FileHandle, offset int64, size uint32) ([]DirEntryPlus, error)

	// ReleaseDir closes a directory handle.
	ReleaseDir(ctx Context, ino Inode, fh FileHandle) error

	// StatFS returns filesystem statistics.
	StatFS(ctx Context, ino Inode) (*StatFS, error)

	// Access checks file permissions.
	// mask contains the requested permission bits (R_OK, W_OK, X_OK).
	// Return nil to allow, syscall.EACCES to deny.
	// Return syscall.ENOSYS to use kernel default_permissions.
	Access(ctx Context, ino Inode, mask uint32) error

	// Forget decrements the lookup count for an inode.
	// Called when the kernel removes inode from cache.
	// nlookup is the number of lookups to forget.
	Forget(ctx Context, ino Inode, nlookup uint64)

	// BatchForget is like Forget but for multiple inodes at once.
	BatchForget(ctx Context, entries []ForgetEntry)
}

// FilesystemBase provides default implementations for optional methods.
// Embed this in your filesystem implementation to provide sensible defaults.
type FilesystemBase struct{}

// Init is a no-op by default.
func (FilesystemBase) Init(ctx Context, config *Config) error {
	return nil
}

// Destroy is a no-op by default.
func (FilesystemBase) Destroy(ctx Context) {}

// ReadLink returns ENOSYS by default.
func (FilesystemBase) ReadLink(ctx Context, ino Inode) (string, error) {
	return "", syscall.ENOSYS
}

// Open returns a zero handle by default.
func (FilesystemBase) Open(ctx Context, ino Inode, flags uint32) (*OpenResponse, error) {
	return &OpenResponse{Handle: 0}, nil
}

// Release is a no-op by default.
func (FilesystemBase) Release(ctx Context, ino Inode, fh FileHandle) error {
	return nil
}

// OpenDir returns a zero handle by default.
func (FilesystemBase) OpenDir(ctx Context, ino Inode, flags uint32) (*OpenResponse, error) {
	return &OpenResponse{Handle: 0}, nil
}

// ReadDirPlus returns ENOSYS by default, falling back to ReadDir.
func (FilesystemBase) ReadDirPlus(ctx Context, ino Inode, fh FileHandle, offset int64, size uint32) ([]DirEntryPlus, error) {
	return nil, syscall.ENOSYS
}

// ReleaseDir is a no-op by default.
func (FilesystemBase) ReleaseDir(ctx Context, ino Inode, fh FileHandle) error {
	return nil
}

// StatFS returns default filesystem statistics.
func (FilesystemBase) StatFS(ctx Context, ino Inode) (*StatFS, error) {
	return &StatFS{
		Blocks:  0,
		Bfree:   0,
		Bavail:  0,
		Files:   0,
		Ffree:   0,
		Bsize:   4096,
		Namelen: 255,
		Frsize:  4096,
	}, nil
}

// Access allows all access by default. Override for custom permissions.
func (FilesystemBase) Access(ctx Context, ino Inode, mask uint32) error {
	return nil
}

// Forget is a no-op by default.
func (FilesystemBase) Forget(ctx Context, ino Inode, nlookup uint64) {}

// BatchForget calls Forget for each entry by default.
func (fs FilesystemBase) BatchForget(ctx Context, entries []ForgetEntry) {
	for _, e := range entries {
		fs.Forget(ctx, e.Ino, e.Nlookup)
	}
}
