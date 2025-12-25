# rofuse

A read-only FUSE (Filesystem in Userspace) implementation for Go that communicates directly with the Linux kernel via `/dev/fuse`. No libfuse dependency required.

## Features

- **Direct kernel communication** - Talks directly to `/dev/fuse`, no C dependencies
- **Read-only by design** - All write operations return `EROFS`
- **Inode-based API** - Clean interface operating on inode numbers, not paths
- **Handle sharing** - Support for multi-process serving via FD passing and `FUSE_DEV_IOC_CLONE`
- **FUSE 7.26+** - Targets modern Linux kernels (5.x+) with capability negotiation

## Installation

```bash
go get github.com/KarpelesLab/rofuse
```

## Quick Start

Implement the `Filesystem` interface to create your own read-only filesystem:

```go
package main

import (
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/KarpelesLab/rofuse"
)

type MyFS struct {
    rofuse.FilesystemBase
}

func (fs *MyFS) Lookup(ctx rofuse.Context, parent rofuse.Inode, name string) (*rofuse.Entry, error) {
    if parent == rofuse.RootInode && name == "hello.txt" {
        return &rofuse.Entry{
            Ino:          2,
            Attr:         rofuse.Attr{
                Ino:   2,
                Mode:  0644,
                Nlink: 1,
                Size:  13,
            },
            AttrTimeout:  time.Minute,
            EntryTimeout: time.Minute,
        }, nil
    }
    return nil, syscall.ENOENT
}

func (fs *MyFS) GetAttr(ctx rofuse.Context, ino rofuse.Inode, fh *rofuse.FileHandle) (*rofuse.Attr, error) {
    switch ino {
    case rofuse.RootInode:
        return &rofuse.Attr{
            Ino:   uint64(ino),
            Mode:  os.ModeDir | 0755,
            Nlink: 2,
        }, nil
    case 2:
        return &rofuse.Attr{
            Ino:   2,
            Mode:  0644,
            Nlink: 1,
            Size:  13,
        }, nil
    }
    return nil, syscall.ENOENT
}

func (fs *MyFS) ReadDir(ctx rofuse.Context, ino rofuse.Inode, fh rofuse.FileHandle, offset int64, size uint32) ([]rofuse.DirEntry, error) {
    if ino != rofuse.RootInode {
        return nil, syscall.ENOTDIR
    }

    entries := []rofuse.DirEntry{
        {Ino: rofuse.RootInode, Offset: 1, Type: 4, Name: "."},
        {Ino: rofuse.RootInode, Offset: 2, Type: 4, Name: ".."},
        {Ino: 2, Offset: 3, Type: 8, Name: "hello.txt"},
    }

    // Skip entries before offset
    var result []rofuse.DirEntry
    for _, e := range entries {
        if int64(e.Offset) > offset {
            result = append(result, e)
        }
    }
    return result, nil
}

func (fs *MyFS) Read(ctx rofuse.Context, ino rofuse.Inode, fh rofuse.FileHandle, offset int64, size uint32) ([]byte, error) {
    if ino != 2 {
        return nil, syscall.ENOENT
    }

    content := []byte("Hello, World!")
    if offset >= int64(len(content)) {
        return nil, nil
    }

    end := offset + int64(size)
    if end > int64(len(content)) {
        end = int64(len(content))
    }

    return content[offset:end], nil
}

func main() {
    server, err := rofuse.Mount("/mnt/myfs", &MyFS{}, &rofuse.MountOptions{
        FSName:  "myfs",
        Subtype: "example",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Handle graceful shutdown
    sig := make(chan os.Signal, 1)
    signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
    go func() {
        <-sig
        server.Unmount()
    }()

    log.Println("Mounted at /mnt/myfs")
    if err := server.Serve(); err != nil {
        log.Fatal(err)
    }
}
```

## Filesystem Interface

The `Filesystem` interface defines all operations. Embed `FilesystemBase` for sensible defaults:

```go
type Filesystem interface {
    Init(ctx Context, config *Config) error
    Destroy(ctx Context)
    Lookup(ctx Context, parent Inode, name string) (*Entry, error)
    GetAttr(ctx Context, ino Inode, fh *FileHandle) (*Attr, error)
    ReadLink(ctx Context, ino Inode) (string, error)
    Open(ctx Context, ino Inode, flags uint32) (*OpenResponse, error)
    Read(ctx Context, ino Inode, fh FileHandle, offset int64, size uint32) ([]byte, error)
    Release(ctx Context, ino Inode, fh FileHandle) error
    OpenDir(ctx Context, ino Inode, flags uint32) (*OpenResponse, error)
    ReadDir(ctx Context, ino Inode, fh FileHandle, offset int64, size uint32) ([]DirEntry, error)
    ReadDirPlus(ctx Context, ino Inode, fh FileHandle, offset int64, size uint32) ([]DirEntryPlus, error)
    ReleaseDir(ctx Context, ino Inode, fh FileHandle) error
    StatFS(ctx Context, ino Inode) (*StatFS, error)
    Access(ctx Context, ino Inode, mask uint32) error
    Forget(ctx Context, ino Inode, nlookup uint64)
    BatchForget(ctx Context, entries []ForgetEntry)
}
```

## Mount Options

```go
type MountOptions struct {
    Debug              bool   // Enable debug logging
    MaxReadahead       uint32 // Maximum readahead size (default: 128KB)
    MaxWrite           uint32 // Maximum write size (default: 128KB)
    MaxBackground      uint16 // Max background requests (default: 12)
    DirectMount        bool   // Bypass fusermount (requires CAP_SYS_ADMIN)
    AllowOther         bool   // Allow other users to access mount
    DefaultPermissions bool   // Use kernel permission checks
    FSName             string // Filesystem name in /proc/mounts
    Subtype            string // Filesystem subtype
}
```

## Handle Sharing

For load balancing or seamless process upgrades, you can share the FUSE file descriptor:

### Using FUSE_DEV_IOC_CLONE (same process, multiple workers)

```go
import "github.com/KarpelesLab/rofuse/sharing"

// Clone the FD for worker goroutines
workerFds, err := sharing.CloneMultiple(server.Fd(), numWorkers)
defer sharing.CloseAll(workerFds)
```

### Using FD Passing (multiple processes)

Coordinator process:
```go
import "github.com/KarpelesLab/rofuse/sharing"

coord, err := sharing.NewCoordinator("/tmp/fuse.sock", server.Fd())
defer coord.Close()

// Accept workers
for {
    worker, err := coord.AcceptWorker()
    // worker is now serving FUSE requests
}
```

Worker process:
```go
import "github.com/KarpelesLab/rofuse/sharing"

client, err := sharing.ConnectToCoordinator("/tmp/fuse.sock", os.Getpid())
defer client.Close()

// Use client.Fd() for FUSE I/O
```

## Supported Operations

| Operation | Description |
|-----------|-------------|
| INIT | Protocol handshake |
| DESTROY | Unmount notification |
| LOOKUP | Find entry by name |
| FORGET | Release inode reference |
| BATCH_FORGET | Release multiple inodes |
| GETATTR | Get file attributes |
| READLINK | Read symlink target |
| OPEN | Open file |
| READ | Read file data |
| RELEASE | Close file |
| OPENDIR | Open directory |
| READDIR | List directory |
| READDIRPLUS | List directory with attributes |
| RELEASEDIR | Close directory |
| STATFS | Get filesystem statistics |
| ACCESS | Check permissions |

Write operations (SETATTR, WRITE, CREATE, MKDIR, etc.) return `EROFS`.

## Requirements

- Linux kernel 5.x+ (FUSE protocol 7.26+)
- Go 1.21+
- For `DirectMount`: CAP_SYS_ADMIN or root
- For `AllowOther`: `user_allow_other` in `/etc/fuse.conf`

## License

MIT License
