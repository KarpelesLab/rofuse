package proto

// FUSE operation codes
const (
	OpLookup        uint32 = 1
	OpForget        uint32 = 2 // No reply
	OpGetattr       uint32 = 3
	OpSetattr       uint32 = 4
	OpReadlink      uint32 = 5
	OpSymlink       uint32 = 6
	OpMknod         uint32 = 8
	OpMkdir         uint32 = 9
	OpUnlink        uint32 = 10
	OpRmdir         uint32 = 11
	OpRename        uint32 = 12
	OpLink          uint32 = 13
	OpOpen          uint32 = 14
	OpRead          uint32 = 15
	OpWrite         uint32 = 16
	OpStatfs        uint32 = 17
	OpRelease       uint32 = 18
	OpFsync         uint32 = 20
	OpSetxattr      uint32 = 21
	OpGetxattr      uint32 = 22
	OpListxattr     uint32 = 23
	OpRemovexattr   uint32 = 24
	OpFlush         uint32 = 25
	OpInit          uint32 = 26
	OpOpendir       uint32 = 27
	OpReaddir       uint32 = 28
	OpReleasedir    uint32 = 29
	OpFsyncdir      uint32 = 30
	OpGetlk         uint32 = 31
	OpSetlk         uint32 = 32
	OpSetlkw        uint32 = 33
	OpAccess        uint32 = 34
	OpCreate        uint32 = 35
	OpInterrupt     uint32 = 36
	OpBmap          uint32 = 37
	OpDestroy       uint32 = 38
	OpIoctl         uint32 = 39
	OpPoll          uint32 = 40
	OpNotifyReply   uint32 = 41
	OpBatchForget   uint32 = 42
	OpFallocate     uint32 = 43
	OpReaddirplus   uint32 = 44
	OpRename2       uint32 = 45
	OpLseek         uint32 = 46
	OpCopyFileRange uint32 = 47
	OpSetupMapping  uint32 = 48
	OpRemoveMapping uint32 = 49
	OpSyncfs        uint32 = 50
	OpTmpfile       uint32 = 51
	OpStatx         uint32 = 52
)

// OpcodeName returns the string name of an opcode.
func OpcodeName(op uint32) string {
	switch op {
	case OpLookup:
		return "LOOKUP"
	case OpForget:
		return "FORGET"
	case OpGetattr:
		return "GETATTR"
	case OpSetattr:
		return "SETATTR"
	case OpReadlink:
		return "READLINK"
	case OpSymlink:
		return "SYMLINK"
	case OpMknod:
		return "MKNOD"
	case OpMkdir:
		return "MKDIR"
	case OpUnlink:
		return "UNLINK"
	case OpRmdir:
		return "RMDIR"
	case OpRename:
		return "RENAME"
	case OpLink:
		return "LINK"
	case OpOpen:
		return "OPEN"
	case OpRead:
		return "READ"
	case OpWrite:
		return "WRITE"
	case OpStatfs:
		return "STATFS"
	case OpRelease:
		return "RELEASE"
	case OpFsync:
		return "FSYNC"
	case OpSetxattr:
		return "SETXATTR"
	case OpGetxattr:
		return "GETXATTR"
	case OpListxattr:
		return "LISTXATTR"
	case OpRemovexattr:
		return "REMOVEXATTR"
	case OpFlush:
		return "FLUSH"
	case OpInit:
		return "INIT"
	case OpOpendir:
		return "OPENDIR"
	case OpReaddir:
		return "READDIR"
	case OpReleasedir:
		return "RELEASEDIR"
	case OpFsyncdir:
		return "FSYNCDIR"
	case OpGetlk:
		return "GETLK"
	case OpSetlk:
		return "SETLK"
	case OpSetlkw:
		return "SETLKW"
	case OpAccess:
		return "ACCESS"
	case OpCreate:
		return "CREATE"
	case OpInterrupt:
		return "INTERRUPT"
	case OpBmap:
		return "BMAP"
	case OpDestroy:
		return "DESTROY"
	case OpIoctl:
		return "IOCTL"
	case OpPoll:
		return "POLL"
	case OpNotifyReply:
		return "NOTIFY_REPLY"
	case OpBatchForget:
		return "BATCH_FORGET"
	case OpFallocate:
		return "FALLOCATE"
	case OpReaddirplus:
		return "READDIRPLUS"
	case OpRename2:
		return "RENAME2"
	case OpLseek:
		return "LSEEK"
	case OpCopyFileRange:
		return "COPY_FILE_RANGE"
	case OpSetupMapping:
		return "SETUPMAPPING"
	case OpRemoveMapping:
		return "REMOVEMAPPING"
	case OpSyncfs:
		return "SYNCFS"
	case OpTmpfile:
		return "TMPFILE"
	case OpStatx:
		return "STATX"
	default:
		return "UNKNOWN"
	}
}
