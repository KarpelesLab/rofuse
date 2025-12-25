// Package sharing provides mechanisms for sharing FUSE file descriptors
// between processes for load balancing and seamless upgrades.
package sharing

import (
	"fmt"
	"syscall"
	"unsafe"
)

// FUSE_DEV_IOC_CLONE ioctl number
// Calculated as _IOR(229, 0, uint32) = 0x8004e500
const fuseDevIocClone = 0x8004e500

// CloneFuseFD creates a clone of a FUSE file descriptor for multi-threading.
// The cloned FD shares the same FUSE connection but allows concurrent reads.
//
// This is useful for:
// - Running multiple worker goroutines, each with their own FD
// - Distributing work across multiple CPUs
//
// The cloned FD must be closed when no longer needed.
func CloneFuseFD(masterFd int) (int, error) {
	// Open a new /dev/fuse
	cloneFd, err := syscall.Open("/dev/fuse", syscall.O_RDWR|syscall.O_CLOEXEC, 0)
	if err != nil {
		return -1, fmt.Errorf("open /dev/fuse: %w", err)
	}

	// Clone the master FD to the new FD
	masterFdVal := uint32(masterFd)
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(cloneFd),
		uintptr(fuseDevIocClone),
		uintptr(unsafe.Pointer(&masterFdVal)),
	)
	if errno != 0 {
		syscall.Close(cloneFd)
		return -1, fmt.Errorf("ioctl FUSE_DEV_IOC_CLONE: %w", errno)
	}

	return cloneFd, nil
}

// CloneMultiple creates multiple clones of a FUSE file descriptor.
// Returns a slice of cloned FDs. All FDs must be closed when done.
func CloneMultiple(masterFd int, count int) ([]int, error) {
	fds := make([]int, 0, count)

	for i := 0; i < count; i++ {
		fd, err := CloneFuseFD(masterFd)
		if err != nil {
			// Clean up any FDs we already created
			for _, existingFd := range fds {
				syscall.Close(existingFd)
			}
			return nil, fmt.Errorf("clone %d: %w", i, err)
		}
		fds = append(fds, fd)
	}

	return fds, nil
}

// CloseAll closes all file descriptors in the slice.
func CloseAll(fds []int) {
	for _, fd := range fds {
		if fd >= 0 {
			syscall.Close(fd)
		}
	}
}
