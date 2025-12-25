package proto

// FUSE capability flags for FUSE_INIT.
// These are negotiated between kernel and userspace.
const (
	CapAsyncRead         uint64 = 1 << 0  // Asynchronous read requests
	CapPosixLocks        uint64 = 1 << 1  // POSIX file locking
	CapFileOps           uint64 = 1 << 2  // Kernel file cache based on modification time
	CapAtomicOTrunc      uint64 = 1 << 3  // Atomic O_TRUNC
	CapExportSupport     uint64 = 1 << 4  // Export operations
	CapBigWrites         uint64 = 1 << 5  // Larger than 4KB writes (obsolete in 7.27+)
	CapDontMask          uint64 = 1 << 6  // Don't apply umask to file mode
	CapSpliceWrite       uint64 = 1 << 7  // Splice for writing
	CapSpliceMove        uint64 = 1 << 8  // Splice move (obsolete)
	CapSpliceRead        uint64 = 1 << 9  // Splice for reading
	CapFlockLocks        uint64 = 1 << 10 // Flock locks
	CapIoctlDir          uint64 = 1 << 11 // Ioctl on directories
	CapAutoInvalData     uint64 = 1 << 12 // Auto invalidate cached pages
	CapReaddirplus       uint64 = 1 << 13 // READDIRPLUS support
	CapReaddirplusAuto   uint64 = 1 << 14 // Auto READDIRPLUS
	CapAsyncDIO          uint64 = 1 << 15 // Async direct I/O
	CapWritebackCache    uint64 = 1 << 16 // Writeback cache
	CapNoOpenSupport     uint64 = 1 << 17 // No open/release for files
	CapParallelDirops    uint64 = 1 << 18 // Parallel directory operations
	CapHandleKillpriv    uint64 = 1 << 19 // Kernel handles SUID/SGID clearing
	CapPosixACL          uint64 = 1 << 20 // POSIX ACL support
	CapAbortError        uint64 = 1 << 21 // Return ECONNABORTED on abort
	CapMaxPages          uint64 = 1 << 22 // Max pages for read/write
	CapCacheSymlinks     uint64 = 1 << 23 // Cache symlink targets
	CapNoOpendirSupport  uint64 = 1 << 24 // No opendir/releasedir for dirs
	CapExplicitInvalData uint64 = 1 << 25 // Explicit data invalidation
	CapMapAlignment      uint64 = 1 << 26 // Map alignment
	CapSubmounts         uint64 = 1 << 27 // Submount support
	CapHandleKillprivV2  uint64 = 1 << 28 // Handle KILLPRIV v2
	CapSetxattrExt       uint64 = 1 << 29 // Extended setxattr
	CapInitExt           uint64 = 1 << 30 // Extended init
	CapInitReserved      uint64 = 1 << 31 // Reserved for extension
	CapSecurityCtx       uint64 = 1 << 32 // Security context
	CapHasInode          uint64 = 1 << 33 // Request has inode
	CapCreateSuppGroup   uint64 = 1 << 34 // Use supplementary groups
	CapExpireOnly        uint64 = 1 << 35 // Allow FUSE_EXPIRE_ONLY
	CapPassthrough       uint64 = 1 << 39 // Passthrough mode
	CapNoExportSupport   uint64 = 1 << 40 // No NFS export support
	CapSameFiNode        uint64 = 1 << 41 // Same FI node
)

// Open flags returned by filesystem from Open/OpenDir.
const (
	FopenDirectIO         uint32 = 1 << 0 // Bypass page cache for this file
	FopenKeepCache        uint32 = 1 << 1 // Don't invalidate cache on open
	FopenNonSeekable      uint32 = 1 << 2 // File is not seekable
	FopenCacheDir         uint32 = 1 << 3 // Allow caching directory contents
	FopenStream           uint32 = 1 << 4 // File is stream-like (no splice)
	FopenNoFlush          uint32 = 1 << 5 // Don't flush data on close
	FopenParallelDirectWr uint32 = 1 << 6 // Parallel direct writes allowed
	FopenPassthrough      uint32 = 1 << 7 // Use passthrough mode
)

// GetAttr flags
const (
	GetattrFh uint32 = 1 << 0 // Fh field is valid
)

// Read flags (from FUSE_READ_* in kernel)
const (
	ReadLockowner uint32 = 1 << 1 // Lock owner is valid
)

// Release flags
const (
	ReleaseFlush     uint32 = 1 << 0 // FLUSH operation at release
	ReleaseFlock     uint32 = 1 << 1 // Release flock
	ReleaseFlushSync uint32 = 1 << 2 // Synchronous flush
)

// File types for directory entries (DT_* from dirent.h)
const (
	DtUnknown uint32 = 0
	DtFifo    uint32 = 1
	DtChr     uint32 = 2
	DtDir     uint32 = 4
	DtBlk     uint32 = 6
	DtReg     uint32 = 8
	DtLnk     uint32 = 10
	DtSock    uint32 = 12
	DtWht     uint32 = 14
)

// File mode bits for Attr.Mode
const (
	ModeTypeMask uint32 = 0170000 // Mask for file type

	ModeSocket  uint32 = 0140000
	ModeSymlink uint32 = 0120000
	ModeRegular uint32 = 0100000
	ModeBlock   uint32 = 0060000
	ModeDir     uint32 = 0040000
	ModeChar    uint32 = 0020000
	ModeFifo    uint32 = 0010000

	ModeSetuid uint32 = 04000
	ModeSetgid uint32 = 02000
	ModeSticky uint32 = 01000

	ModePermMask uint32 = 0777 // Mask for permission bits
)

// Access mode bits for FUSE_ACCESS
const (
	AccessExec  uint32 = 1 // X_OK
	AccessWrite uint32 = 2 // W_OK
	AccessRead  uint32 = 4 // R_OK
)
