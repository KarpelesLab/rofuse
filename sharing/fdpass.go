package sharing

import (
	"fmt"
	"net"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

// FDPasser handles passing FUSE FDs between processes using Unix sockets.
type FDPasser struct {
	sockPath string
	listener *net.UnixListener
}

// NewFDPasser creates a new FD passer that listens on the given Unix socket path.
// The socket will be created and removed when Close is called.
func NewFDPasser(sockPath string) (*FDPasser, error) {
	// Remove existing socket if present
	os.Remove(sockPath)

	addr := &net.UnixAddr{Name: sockPath, Net: "unix"}
	ln, err := net.ListenUnix("unix", addr)
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}

	return &FDPasser{sockPath: sockPath, listener: ln}, nil
}

// Accept waits for a client to connect and returns the connection.
func (p *FDPasser) Accept() (*net.UnixConn, error) {
	return p.listener.AcceptUnix()
}

// Close closes the listener and removes the socket file.
func (p *FDPasser) Close() error {
	err := p.listener.Close()
	os.Remove(p.sockPath)
	return err
}

// SockPath returns the socket path.
func (p *FDPasser) SockPath() string {
	return p.sockPath
}

// SendFD sends a file descriptor to another process via a Unix socket connection.
// The connection must be a Unix socket (SOCK_STREAM or SOCK_SEQPACKET).
func SendFD(conn *net.UnixConn, fd int) error {
	// Get the underlying file
	f, err := conn.File()
	if err != nil {
		return fmt.Errorf("get file: %w", err)
	}
	defer f.Close()

	// Build SCM_RIGHTS message
	rights := syscall.UnixRights(fd)

	// Must send at least one byte of data with SCM_RIGHTS
	data := []byte{0}

	err = syscall.Sendmsg(int(f.Fd()), data, rights, nil, 0)
	if err != nil {
		return fmt.Errorf("sendmsg: %w", err)
	}

	return nil
}

// ReceiveFD receives a file descriptor from another process via a Unix socket.
func ReceiveFD(conn *net.UnixConn) (int, error) {
	// Get the underlying file
	f, err := conn.File()
	if err != nil {
		return -1, fmt.Errorf("get file: %w", err)
	}
	defer f.Close()

	data := make([]byte, 1)
	oob := make([]byte, unix.CmsgSpace(4)) // Space for one FD

	_, oobn, _, _, err := syscall.Recvmsg(int(f.Fd()), data, oob, 0)
	if err != nil {
		return -1, fmt.Errorf("recvmsg: %w", err)
	}

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

	return -1, fmt.Errorf("no file descriptor received")
}

// ConnectAndReceiveFD connects to a Unix socket and receives a file descriptor.
// This is a convenience function for worker processes.
func ConnectAndReceiveFD(sockPath string) (int, error) {
	addr := &net.UnixAddr{Name: sockPath, Net: "unix"}
	conn, err := net.DialUnix("unix", nil, addr)
	if err != nil {
		return -1, fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	return ReceiveFD(conn)
}
