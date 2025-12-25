package rofuse

import (
	"context"
)

// Context extends context.Context with FUSE-specific information.
type Context interface {
	context.Context

	// Uid returns the user ID of the calling process.
	Uid() uint32

	// Gid returns the group ID of the calling process.
	Gid() uint32

	// Pid returns the process ID of the calling process.
	Pid() uint32

	// Unique returns the unique request ID.
	Unique() uint64
}

// fuseContext implements Context.
type fuseContext struct {
	context.Context
	uid    uint32
	gid    uint32
	pid    uint32
	unique uint64
}

func (c *fuseContext) Uid() uint32    { return c.uid }
func (c *fuseContext) Gid() uint32    { return c.gid }
func (c *fuseContext) Pid() uint32    { return c.pid }
func (c *fuseContext) Unique() uint64 { return c.unique }

// newContext creates a FUSE context from request header.
func newContext(parent context.Context, uid, gid, pid uint32, unique uint64) Context {
	return &fuseContext{
		Context: parent,
		uid:     uid,
		gid:     gid,
		pid:     pid,
		unique:  unique,
	}
}
