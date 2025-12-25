package rofuse

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/sys/unix"
)

// MountOptions configures the FUSE mount.
type MountOptions struct {
	// Debug enables debug logging.
	Debug bool

	// MaxReadahead is the maximum readahead size in bytes.
	// Default is 128KB.
	MaxReadahead uint32

	// MaxWrite is the maximum write size in bytes.
	// Default is 128KB.
	MaxWrite uint32

	// MaxBackground is the max number of background requests.
	// Default is 12.
	MaxBackground uint16

	// DirectMount bypasses fusermount and mounts directly.
	// Requires CAP_SYS_ADMIN or root privileges.
	DirectMount bool

	// AllowOther allows other users to access the mount.
	// Requires user_allow_other in /etc/fuse.conf.
	AllowOther bool

	// DefaultPermissions uses kernel permission checks.
	DefaultPermissions bool

	// ReadOnly mounts the filesystem read-only.
	// Always true for this library.
	ReadOnly bool

	// FSName is the filesystem name shown in /proc/mounts.
	FSName string

	// Subtype is the filesystem subtype (e.g., "myfs").
	Subtype string
}

// mount opens /dev/fuse and mounts the filesystem.
func mount(mountPoint string, opts *MountOptions) (int, error) {
	if opts == nil {
		opts = &MountOptions{}
	}

	// Validate mount point exists and is a directory
	fi, err := os.Stat(mountPoint)
	if err != nil {
		return -1, fmt.Errorf("mount point: %w", err)
	}
	if !fi.IsDir() {
		return -1, fmt.Errorf("mount point is not a directory: %s", mountPoint)
	}

	if opts.DirectMount {
		return mountDirect(mountPoint, opts)
	}
	return mountFusermount(mountPoint, opts)
}

// mountDirect mounts without fusermount helper.
// Requires CAP_SYS_ADMIN or root privileges.
func mountDirect(mountPoint string, opts *MountOptions) (int, error) {
	// Open /dev/fuse
	fd, err := syscall.Open("/dev/fuse", syscall.O_RDWR|syscall.O_CLOEXEC, 0)
	if err != nil {
		return -1, fmt.Errorf("open /dev/fuse: %w", err)
	}

	// Build mount options
	mountOpts := fmt.Sprintf(
		"fd=%d,rootmode=%o,user_id=%d,group_id=%d",
		fd,
		040755, // Directory with 0755 permissions
		os.Getuid(),
		os.Getgid(),
	)

	if opts.AllowOther {
		mountOpts += ",allow_other"
	}
	if opts.DefaultPermissions {
		mountOpts += ",default_permissions"
	}

	// Mount flags
	flags := uintptr(syscall.MS_NOSUID | syscall.MS_NODEV)

	// Call mount(2)
	err = syscall.Mount(
		"fuse",     // source
		mountPoint, // target
		"fuse",     // fstype
		flags,      // flags
		mountOpts,  // data
	)
	if err != nil {
		syscall.Close(fd)
		return -1, fmt.Errorf("mount: %w", err)
	}

	return fd, nil
}

// mountFusermount mounts using the fusermount3/fusermount helper.
func mountFusermount(mountPoint string, opts *MountOptions) (int, error) {
	// Create socket pair for receiving the fd
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		return -1, fmt.Errorf("socketpair: %w", err)
	}

	// Build fusermount options
	fusermountOpts := "rw"
	if opts.AllowOther {
		fusermountOpts += ",allow_other"
	}
	if opts.DefaultPermissions {
		fusermountOpts += ",default_permissions"
	}
	if opts.FSName != "" {
		fusermountOpts += ",fsname=" + opts.FSName
	}
	if opts.Subtype != "" {
		fusermountOpts += ",subtype=" + opts.Subtype
	}

	// Try fusermount3 first, then fusermount
	fusermountPath := "fusermount3"
	if _, err := exec.LookPath(fusermountPath); err != nil {
		fusermountPath = "fusermount"
	}

	// Run fusermount
	cmd := exec.Command(fusermountPath, "-o", fusermountOpts, "--", mountPoint)
	cmd.Env = append(os.Environ(), fmt.Sprintf("_FUSE_COMMFD=%d", fds[0]))

	// Pass the socket fd to fusermount
	cmd.ExtraFiles = []*os.File{os.NewFile(uintptr(fds[0]), "fusermount-comm")}

	// Close our copy of fds[0] after starting
	defer syscall.Close(fds[0])

	if err := cmd.Start(); err != nil {
		syscall.Close(fds[1])
		return -1, fmt.Errorf("fusermount: %w", err)
	}

	// Wait for fusermount to complete
	if err := cmd.Wait(); err != nil {
		syscall.Close(fds[1])
		return -1, fmt.Errorf("fusermount: %w", err)
	}

	// Receive the fuse fd from fusermount via SCM_RIGHTS
	buf := make([]byte, 1)
	oob := make([]byte, unix.CmsgSpace(4))

	n, oobn, _, _, err := syscall.Recvmsg(fds[1], buf, oob, 0)
	syscall.Close(fds[1])

	if err != nil {
		return -1, fmt.Errorf("recvmsg: %w", err)
	}
	if n == 0 {
		return -1, fmt.Errorf("fusermount: received empty message")
	}

	// Parse the control message to get the fd
	msgs, err := syscall.ParseSocketControlMessage(oob[:oobn])
	if err != nil {
		return -1, fmt.Errorf("parse control message: %w", err)
	}

	for _, msg := range msgs {
		fds, err := syscall.ParseUnixRights(&msg)
		if err != nil {
			continue
		}
		if len(fds) > 0 {
			return fds[0], nil
		}
	}

	return -1, fmt.Errorf("fusermount: did not receive file descriptor")
}

// unmount unmounts the filesystem.
func unmount(mountPoint string) error {
	// Try lazy unmount first
	err := syscall.Unmount(mountPoint, syscall.MNT_DETACH)
	if err == nil {
		return nil
	}

	// Try normal unmount
	err = syscall.Unmount(mountPoint, 0)
	if err == nil {
		return nil
	}

	// Fall back to fusermount -u
	return execFusermount("-u", mountPoint)
}

// execFusermount runs fusermount with the given arguments.
func execFusermount(args ...string) error {
	// Try fusermount3 first
	fusermountPath := "fusermount3"
	if _, err := exec.LookPath(fusermountPath); err != nil {
		fusermountPath = "fusermount"
	}

	cmd := exec.Command(fusermountPath, args...)
	return cmd.Run()
}
