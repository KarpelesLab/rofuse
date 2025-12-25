package rofuse

import (
	"context"
	"errors"
	"io"
	"os"
	"syscall"
)

// Common errors returned by the FUSE library.
var (
	// ErrNotMounted is returned when trying to operate on an unmounted filesystem.
	ErrNotMounted = errors.New("filesystem not mounted")

	// ErrAlreadyMounted is returned when trying to mount an already mounted filesystem.
	ErrAlreadyMounted = errors.New("filesystem already mounted")

	// ErrServerClosed is returned when the server is closed.
	ErrServerClosed = errors.New("server closed")
)

// toErrno converts a Go error to a FUSE errno value.
// Returns 0 for nil errors and negative errno for errors.
func toErrno(err error) int32 {
	if err == nil {
		return 0
	}

	// Check for syscall.Errno first
	var errno syscall.Errno
	if errors.As(err, &errno) {
		return -int32(errno)
	}

	// Map common errors
	switch {
	case errors.Is(err, os.ErrNotExist):
		return -int32(syscall.ENOENT)
	case errors.Is(err, os.ErrExist):
		return -int32(syscall.EEXIST)
	case errors.Is(err, os.ErrPermission):
		return -int32(syscall.EACCES)
	case errors.Is(err, os.ErrClosed):
		return -int32(syscall.EBADF)
	case errors.Is(err, os.ErrInvalid):
		return -int32(syscall.EINVAL)
	case errors.Is(err, io.EOF):
		return 0 // EOF is not an error for reads
	case errors.Is(err, context.Canceled):
		return -int32(syscall.EINTR)
	case errors.Is(err, context.DeadlineExceeded):
		return -int32(syscall.ETIMEDOUT)
	default:
		return -int32(syscall.EIO)
	}
}
