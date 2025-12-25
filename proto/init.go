package proto

// Protocol version constants
const (
	FuseKernelVersion      = 7
	FuseKernelMinorVersion = 41 // Latest as of Linux 6.12

	// Minimum version we support
	MinSupportedMinor = 26
)

// InitIn is the request body for FUSE_INIT.
// Size varies by protocol version, but we use the 7.36+ format.
type InitIn struct {
	Major        uint32
	Minor        uint32
	MaxReadahead uint32
	Flags        uint32
	Flags2       uint32 // v7.36+
	Unused       [11]uint32
}

// InitInSize is the size of InitIn in bytes.
const InitInSize = 64

// InitOut is the response for FUSE_INIT.
type InitOut struct {
	Major               uint32
	Minor               uint32
	MaxReadahead        uint32
	Flags               uint32
	MaxBackground       uint16
	CongestionThreshold uint16
	MaxWrite            uint32
	TimeGran            uint32 // Timestamp granularity (nanoseconds)
	MaxPages            uint16 // v7.28+
	MapAlignment        uint16 // v7.31+
	Flags2              uint32 // v7.36+
	MaxStackDepth       uint32 // v7.40+
	Unused              [6]uint32
}

// InitOutSize is the size of InitOut in bytes.
const InitOutSize = 64

// Default values for initialization
const (
	DefaultMaxReadahead        = 128 * 1024 // 128 KB
	DefaultMaxWrite            = 128 * 1024 // 128 KB
	DefaultMaxBackground       = 12
	DefaultCongestionThreshold = 9
	DefaultTimeGran            = 1  // Nanosecond precision
	DefaultMaxPages            = 32 // 32 * 4096 = 128 KB
)

// MinBufferSize is the minimum buffer size for reading FUSE requests.
// Must be at least FUSE_MIN_READ_BUFFER (8192) bytes.
const MinBufferSize = 8192

// MaxBufferSize is a reasonable maximum buffer size.
// Equal to MaxWrite + InHeaderSize + some extra for dirent padding.
const MaxBufferSize = 1024 * 1024 // 1 MB

// DefaultBufferSize is the default buffer size for FUSE I/O.
const DefaultBufferSize = 128*1024 + 4096
