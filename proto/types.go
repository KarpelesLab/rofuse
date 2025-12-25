// Package proto contains the FUSE wire protocol structures.
// These structures must match the kernel's fuse.h exactly for binary compatibility.
package proto

// InHeader is the header for all FUSE requests from the kernel.
// Size: 40 bytes
type InHeader struct {
	Len     uint32 // Total message length including header
	Opcode  uint32 // Operation code
	Unique  uint64 // Request ID for matching responses
	NodeID  uint64 // Inode number (0 for some operations)
	Uid     uint32 // User ID of calling process
	Gid     uint32 // Group ID of calling process
	Pid     uint32 // Process ID of calling process
	Padding uint32
}

// InHeaderSize is the size of InHeader in bytes.
const InHeaderSize = 40

// OutHeader is the header for all FUSE responses to the kernel.
// Size: 16 bytes
type OutHeader struct {
	Len    uint32 // Total message length including header
	Error  int32  // Error code (0 for success, negative errno)
	Unique uint64 // Request ID from InHeader
}

// OutHeaderSize is the size of OutHeader in bytes.
const OutHeaderSize = 16

// Attr represents file attributes in the FUSE wire format.
// Size: 88 bytes
type Attr struct {
	Ino       uint64
	Size      uint64
	Blocks    uint64
	Atime     uint64
	Mtime     uint64
	Ctime     uint64
	AtimeNsec uint32
	MtimeNsec uint32
	CtimeNsec uint32
	Mode      uint32
	Nlink     uint32
	Uid       uint32
	Gid       uint32
	Rdev      uint32
	Blksize   uint32
	Flags     uint32
}

// AttrSize is the size of Attr in bytes.
const AttrSize = 88

// EntryOut is the response to FUSE_LOOKUP.
// Size: 128 bytes (40 + 88)
type EntryOut struct {
	NodeID         uint64 // Inode ID
	Generation     uint64 // Inode generation
	EntryValid     uint64 // Entry cache timeout (seconds)
	AttrValid      uint64 // Attribute cache timeout (seconds)
	EntryValidNsec uint32
	AttrValidNsec  uint32
	Attr           Attr
}

// EntryOutSize is the size of EntryOut in bytes.
const EntryOutSize = 128

// AttrOut is the response to FUSE_GETATTR.
// Size: 104 bytes (16 + 88)
type AttrOut struct {
	AttrValid     uint64 // Attribute cache timeout (seconds)
	AttrValidNsec uint32
	Dummy         uint32
	Attr          Attr
}

// AttrOutSize is the size of AttrOut in bytes.
const AttrOutSize = 104

// GetAttrIn is the request body for FUSE_GETATTR.
// Size: 16 bytes
type GetAttrIn struct {
	Flags uint32
	Dummy uint32
	Fh    uint64
}

// GetAttrInSize is the size of GetAttrIn in bytes.
const GetAttrInSize = 16

// OpenIn is the request body for FUSE_OPEN and FUSE_OPENDIR.
// Size: 8 bytes
type OpenIn struct {
	Flags     uint32 // Open flags (O_RDONLY, etc.)
	OpenFlags uint32 // FUSE_OPEN_* flags (v7.12+)
}

// OpenInSize is the size of OpenIn in bytes.
const OpenInSize = 8

// OpenOut is the response for FUSE_OPEN and FUSE_OPENDIR.
// Size: 16 bytes
type OpenOut struct {
	Fh        uint64 // File handle
	OpenFlags uint32 // FOPEN_* flags
	Padding   uint32
}

// OpenOutSize is the size of OpenOut in bytes.
const OpenOutSize = 16

// ReadIn is the request body for FUSE_READ and FUSE_READDIR.
// Size: 40 bytes
type ReadIn struct {
	Fh        uint64
	Offset    uint64
	Size      uint32
	ReadFlags uint32
	LockOwner uint64
	Flags     uint32
	Padding   uint32
}

// ReadInSize is the size of ReadIn in bytes.
const ReadInSize = 40

// ReleaseIn is the request body for FUSE_RELEASE and FUSE_RELEASEDIR.
// Size: 24 bytes
type ReleaseIn struct {
	Fh           uint64
	Flags        uint32
	ReleaseFlags uint32
	LockOwner    uint64
}

// ReleaseInSize is the size of ReleaseIn in bytes.
const ReleaseInSize = 24

// ForgetIn is the request body for FUSE_FORGET.
// Size: 8 bytes
type ForgetIn struct {
	Nlookup uint64
}

// ForgetInSize is the size of ForgetIn in bytes.
const ForgetInSize = 8

// BatchForgetIn is the request body for FUSE_BATCH_FORGET.
// Size: 8 bytes (followed by Count ForgetOne entries)
type BatchForgetIn struct {
	Count uint32
	Dummy uint32
}

// BatchForgetInSize is the size of BatchForgetIn in bytes.
const BatchForgetInSize = 8

// ForgetOne is one entry in FUSE_BATCH_FORGET.
// Size: 16 bytes
type ForgetOne struct {
	NodeID  uint64
	Nlookup uint64
}

// ForgetOneSize is the size of ForgetOne in bytes.
const ForgetOneSize = 16

// AccessIn is the request body for FUSE_ACCESS.
// Size: 8 bytes
type AccessIn struct {
	Mask    uint32
	Padding uint32
}

// AccessInSize is the size of AccessIn in bytes.
const AccessInSize = 8

// Dirent is the directory entry format for FUSE_READDIR.
// Size: 24 bytes (followed by variable-length name, padded to 8-byte boundary)
type Dirent struct {
	Ino     uint64
	Off     uint64 // Offset for next entry
	Namelen uint32
	Type    uint32
}

// DirentSize is the size of Dirent in bytes (excluding name).
const DirentSize = 24

// DirentPlus is the directory entry format for FUSE_READDIRPLUS.
// Contains EntryOut followed by Dirent.
// Size: 152 bytes (128 + 24, followed by variable-length name)
type DirentPlus struct {
	EntryOut EntryOut
	Dirent   Dirent
}

// DirentPlusSize is the size of DirentPlus in bytes (excluding name).
const DirentPlusSize = EntryOutSize + DirentSize

// StatfsOut is the response for FUSE_STATFS.
// Size: 64 bytes
type StatfsOut struct {
	St Kstatfs
}

// StatfsOutSize is the size of StatfsOut in bytes.
const StatfsOutSize = 64

// Kstatfs is the filesystem statistics structure.
// Size: 64 bytes
type Kstatfs struct {
	Blocks  uint64
	Bfree   uint64
	Bavail  uint64
	Files   uint64
	Ffree   uint64
	Bsize   uint32
	Namelen uint32
	Frsize  uint32
	Padding uint32
	Spare   [6]uint32
}

// KstatfsSize is the size of Kstatfs in bytes.
const KstatfsSize = 64

// FlushIn is the request body for FUSE_FLUSH.
// Size: 24 bytes
type FlushIn struct {
	Fh        uint64
	Unused    uint32
	Padding   uint32
	LockOwner uint64
}

// FlushInSize is the size of FlushIn in bytes.
const FlushInSize = 24

// InterruptIn is the request body for FUSE_INTERRUPT.
// Size: 8 bytes
type InterruptIn struct {
	Unique uint64
}

// InterruptInSize is the size of InterruptIn in bytes.
const InterruptInSize = 8
