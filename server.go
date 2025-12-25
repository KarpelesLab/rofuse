package rofuse

import (
	"context"
	"sync"
	"syscall"

	"github.com/KarpelesLab/rofuse/proto"
)

// Server manages the FUSE connection and dispatches requests.
type Server struct {
	fs         Filesystem
	mountPoint string
	conn       *connection
	config     *Config

	// Buffer pool
	bufPool *bufferPool

	// Configuration
	opts *MountOptions

	// Lifecycle management
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// State
	initialized bool
	destroyed   bool
	mu          sync.RWMutex
}

// Mount mounts a filesystem at the given path and returns a Server.
func Mount(mountPoint string, fs Filesystem, opts *MountOptions) (*Server, error) {
	if opts == nil {
		opts = &MountOptions{}
	}

	// Set defaults
	if opts.MaxReadahead == 0 {
		opts.MaxReadahead = proto.DefaultMaxReadahead
	}
	if opts.MaxWrite == 0 {
		opts.MaxWrite = proto.DefaultMaxWrite
	}
	if opts.MaxBackground == 0 {
		opts.MaxBackground = proto.DefaultMaxBackground
	}

	// Mount the filesystem
	fd, err := mount(mountPoint, opts)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	s := &Server{
		fs:         fs,
		mountPoint: mountPoint,
		conn:       newConnection(fd),
		bufPool:    newBufferPool(int(opts.MaxWrite) + proto.InHeaderSize + 4096),
		opts:       opts,
		ctx:        ctx,
		cancel:     cancel,
	}

	return s, nil
}

// MountPoint returns the mount point path.
func (s *Server) MountPoint() string {
	return s.mountPoint
}

// Serve runs the server loop. Blocks until unmounted or error.
func (s *Server) Serve() error {
	for {
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		default:
		}

		req, err := s.conn.readRequest(s.bufPool)
		if err != nil {
			if err == syscall.EINTR {
				continue
			}
			if err == ErrNotMounted {
				return nil
			}
			return err
		}

		// Handle request
		s.wg.Add(1)
		go func(r *request) {
			defer s.wg.Done()
			defer r.release()
			s.handleRequest(r)
		}(req)
	}
}

// handleRequest dispatches a request to the appropriate handler.
func (s *Server) handleRequest(req *request) {
	opcode := req.header.Opcode

	// Check if it's a write operation (read-only filesystem)
	if isWriteOp(opcode) {
		s.sendError(req, syscall.EROFS)
		return
	}

	// Get handler
	h, ok := handlers[opcode]
	if !ok {
		// Unknown opcode - return ENOSYS
		if s.opts.Debug {
			// Log unknown opcode
		}
		s.sendError(req, syscall.ENOSYS)
		return
	}

	// Execute handler
	if err := h(s, req); err != nil {
		s.sendError(req, err)
		return
	}
}

// sendError sends an error response.
func (s *Server) sendError(req *request, err error) {
	// Don't send response for FORGET operations
	if req.header.Opcode == proto.OpForget || req.header.Opcode == proto.OpBatchForget {
		return
	}

	errno := toErrno(err)
	resp := newErrorResponse(req, errno)
	s.conn.writeResponse(resp.bytes())
}

// sendResponse sends a successful response.
func (s *Server) sendResponse(req *request, payload []byte) {
	resp := newResponse(req, len(payload))
	if len(payload) > 0 {
		copy(resp.payload(), payload)
	}
	s.conn.writeResponse(resp.bytes())
}

// newContext creates a FUSE context from a request.
func (s *Server) newContext(req *request) Context {
	return newContext(s.ctx, req.header.Uid, req.header.Gid, req.header.Pid, req.header.Unique)
}

// Unmount unmounts the filesystem and shuts down the server.
func (s *Server) Unmount() error {
	s.cancel()
	err := unmount(s.mountPoint)
	s.conn.close()
	return err
}

// Wait waits for all pending requests to complete.
func (s *Server) Wait() {
	s.wg.Wait()
}

// Fd returns the FUSE file descriptor.
// This can be used for handle sharing.
func (s *Server) Fd() int {
	return s.conn.Fd()
}

// isWriteOp returns true if the opcode is a write operation.
func isWriteOp(opcode uint32) bool {
	switch opcode {
	case proto.OpSetattr,
		proto.OpSymlink,
		proto.OpMknod,
		proto.OpMkdir,
		proto.OpUnlink,
		proto.OpRmdir,
		proto.OpRename,
		proto.OpLink,
		proto.OpWrite,
		proto.OpSetxattr,
		proto.OpRemovexattr,
		proto.OpCreate,
		proto.OpRename2,
		proto.OpFallocate,
		proto.OpCopyFileRange,
		proto.OpTmpfile:
		return true
	default:
		return false
	}
}
