package rofuse

import (
	"os"
	"time"

	"github.com/KarpelesLab/rofuse/proto"
)

// Attr represents file/directory attributes.
type Attr struct {
	Ino     Inode       // Inode number
	Size    uint64      // File size in bytes
	Blocks  uint64      // Number of 512B blocks allocated
	Atime   time.Time   // Access time
	Mtime   time.Time   // Modification time
	Ctime   time.Time   // Status change time
	Mode    os.FileMode // File mode and permissions
	Nlink   uint32      // Number of hard links
	Uid     uint32      // Owner user ID
	Gid     uint32      // Owner group ID
	Rdev    uint32      // Device ID (for special files)
	Blksize uint32      // Block size for filesystem I/O
}

// Entry represents a directory entry lookup result.
type Entry struct {
	Ino          Inode         // Inode number of the entry
	Generation   uint64        // Inode generation (for NFS exports)
	Attr         Attr          // Attributes of the entry
	AttrTimeout  time.Duration // How long to cache attributes
	EntryTimeout time.Duration // How long to cache the entry
}

// DirEntry represents a directory entry for ReadDir.
type DirEntry struct {
	Ino    Inode  // Inode number
	Offset uint64 // Offset for next entry (cookie)
	Type   uint32 // File type (DT_REG, DT_DIR, etc.)
	Name   string // Entry name
}

// DirEntryPlus is a DirEntry with full attributes for ReadDirPlus.
type DirEntryPlus struct {
	Entry Entry  // Full entry with attributes
	Name  string // Entry name
}

// FileHandle represents an open file or directory handle.
type FileHandle uint64

// OpenResponse contains the result of an Open or OpenDir operation.
type OpenResponse struct {
	Handle FileHandle // Handle to use for subsequent operations
	Flags  OpenFlags  // Response flags (FOPEN_*)
}

// OpenFlags are flags returned from Open/OpenDir.
type OpenFlags uint32

const (
	// OpenDirectIO bypasses the page cache for this file.
	OpenDirectIO OpenFlags = OpenFlags(proto.FopenDirectIO)

	// OpenKeepCache prevents cache invalidation on open.
	OpenKeepCache OpenFlags = OpenFlags(proto.FopenKeepCache)

	// OpenNonSeekable indicates the file is not seekable.
	OpenNonSeekable OpenFlags = OpenFlags(proto.FopenNonSeekable)

	// OpenCacheDir allows caching directory contents.
	OpenCacheDir OpenFlags = OpenFlags(proto.FopenCacheDir)

	// OpenStream indicates the file is stream-like (no splice).
	OpenStream OpenFlags = OpenFlags(proto.FopenStream)

	// OpenNoFlush prevents data flush on close.
	OpenNoFlush OpenFlags = OpenFlags(proto.FopenNoFlush)
)

// StatFS represents filesystem statistics.
type StatFS struct {
	Blocks  uint64 // Total data blocks in filesystem
	Bfree   uint64 // Free blocks in filesystem
	Bavail  uint64 // Free blocks available to unprivileged users
	Files   uint64 // Total file nodes in filesystem
	Ffree   uint64 // Free file nodes in filesystem
	Bsize   uint32 // Optimal transfer block size
	Namelen uint32 // Maximum length of filenames
	Frsize  uint32 // Fragment size
}

// ForgetEntry represents an entry in BatchForget.
type ForgetEntry struct {
	Ino     Inode
	Nlookup uint64
}

// Config contains the negotiated FUSE configuration.
// It is passed to Filesystem.Init after protocol negotiation.
type Config struct {
	ProtoMajor   uint32 // Negotiated protocol major version
	ProtoMinor   uint32 // Negotiated protocol minor version
	MaxReadahead uint32 // Maximum readahead size
	MaxWrite     uint32 // Maximum write size
	MaxPages     uint16 // Maximum pages per request
}

// Helper functions for converting between user types and proto types

func attrToProto(a *Attr) proto.Attr {
	return proto.Attr{
		Ino:       uint64(a.Ino),
		Size:      a.Size,
		Blocks:    a.Blocks,
		Atime:     uint64(a.Atime.Unix()),
		Mtime:     uint64(a.Mtime.Unix()),
		Ctime:     uint64(a.Ctime.Unix()),
		AtimeNsec: uint32(a.Atime.Nanosecond()),
		MtimeNsec: uint32(a.Mtime.Nanosecond()),
		CtimeNsec: uint32(a.Ctime.Nanosecond()),
		Mode:      fileModeToUnix(a.Mode),
		Nlink:     a.Nlink,
		Uid:       a.Uid,
		Gid:       a.Gid,
		Rdev:      a.Rdev,
		Blksize:   a.Blksize,
	}
}

func fileModeToUnix(mode os.FileMode) uint32 {
	m := uint32(mode.Perm())

	switch mode.Type() {
	case os.ModeDir:
		m |= proto.ModeDir
	case os.ModeSymlink:
		m |= proto.ModeSymlink
	case os.ModeNamedPipe:
		m |= proto.ModeFifo
	case os.ModeSocket:
		m |= proto.ModeSocket
	case os.ModeDevice:
		if mode&os.ModeCharDevice != 0 {
			m |= proto.ModeChar
		} else {
			m |= proto.ModeBlock
		}
	default:
		m |= proto.ModeRegular
	}

	if mode&os.ModeSetuid != 0 {
		m |= proto.ModeSetuid
	}
	if mode&os.ModeSetgid != 0 {
		m |= proto.ModeSetgid
	}
	if mode&os.ModeSticky != 0 {
		m |= proto.ModeSticky
	}

	return m
}

// durationToTimespec converts a duration to seconds and nanoseconds.
func durationToTimespec(d time.Duration) (sec uint64, nsec uint32) {
	sec = uint64(d / time.Second)
	nsec = uint32((d % time.Second) / time.Nanosecond)
	return
}

// fileModeToType converts os.FileMode to a DT_* type constant.
func fileModeToType(mode os.FileMode) uint32 {
	switch mode.Type() {
	case os.ModeDir:
		return proto.DtDir
	case os.ModeSymlink:
		return proto.DtLnk
	case os.ModeNamedPipe:
		return proto.DtFifo
	case os.ModeSocket:
		return proto.DtSock
	case os.ModeDevice:
		if mode&os.ModeCharDevice != 0 {
			return proto.DtChr
		}
		return proto.DtBlk
	default:
		return proto.DtReg
	}
}
