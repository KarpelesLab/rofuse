package rofuse

// Inode represents a filesystem inode number.
// The root inode is always 1 (FUSE_ROOT_ID).
type Inode uint64

const (
	// RootInode is the inode number of the root directory.
	RootInode Inode = 1
)

// IsRoot returns true if this is the root inode.
func (i Inode) IsRoot() bool {
	return i == RootInode
}

// Valid returns true if this is a valid inode number.
// Inode 0 is reserved and invalid.
func (i Inode) Valid() bool {
	return i != 0
}
