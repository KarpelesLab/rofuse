package rofuse

import (
	"encoding/binary"
	"io"
	"sync"
	"syscall"
	"unsafe"

	"github.com/KarpelesLab/rofuse/proto"
)

// connection manages /dev/fuse I/O.
type connection struct {
	fd      int
	mounted bool

	// Serialized writes
	writeMu sync.Mutex

	// Protocol version negotiated during INIT
	protoMajor uint32
	protoMinor uint32
}

// newConnection creates a new FUSE connection.
func newConnection(fd int) *connection {
	return &connection{
		fd:      fd,
		mounted: true,
	}
}

// readRequest reads the next FUSE request from the kernel.
func (c *connection) readRequest(pool *bufferPool) (*request, error) {
	buf := pool.get()

	n, err := syscall.Read(c.fd, buf)
	if err != nil {
		pool.put(buf)
		if err == syscall.ENODEV {
			return nil, ErrNotMounted
		}
		if err == syscall.EINTR {
			// Interrupted, try again
			return nil, err
		}
		return nil, err
	}

	if n < proto.InHeaderSize {
		pool.put(buf)
		return nil, io.ErrUnexpectedEOF
	}

	return newRequest(buf[:n], pool), nil
}

// writeResponse writes a FUSE response to the kernel.
func (c *connection) writeResponse(data []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	_, err := syscall.Write(c.fd, data)
	if err == syscall.ENODEV {
		return ErrNotMounted
	}
	return err
}

// close closes the connection.
func (c *connection) close() error {
	if c.fd >= 0 {
		err := syscall.Close(c.fd)
		c.fd = -1
		return err
	}
	return nil
}

// fd returns the file descriptor for the connection.
func (c *connection) Fd() int {
	return c.fd
}

// request represents a FUSE request from the kernel.
type request struct {
	header *proto.InHeader
	data   []byte // Full request data including header
	pool   *bufferPool
}

// newRequest parses a FUSE request from raw data.
func newRequest(data []byte, pool *bufferPool) *request {
	return &request{
		header: (*proto.InHeader)(unsafe.Pointer(&data[0])),
		data:   data,
		pool:   pool,
	}
}

// body returns the request body (data after the header).
func (r *request) body() unsafe.Pointer {
	if len(r.data) <= proto.InHeaderSize {
		return nil
	}
	return unsafe.Pointer(&r.data[proto.InHeaderSize])
}

// bodyBytes returns the request body as a byte slice.
func (r *request) bodyBytes() []byte {
	if len(r.data) <= proto.InHeaderSize {
		return nil
	}
	return r.data[proto.InHeaderSize:]
}

// filename extracts a null-terminated filename from the request body.
func (r *request) filename() string {
	body := r.bodyBytes()
	if body == nil {
		return ""
	}
	// Find null terminator
	for i, b := range body {
		if b == 0 {
			return string(body[:i])
		}
	}
	return string(body)
}

// release returns the request buffer to the pool.
func (r *request) release() {
	if r.pool != nil && r.data != nil {
		r.pool.put(r.data[:cap(r.data)])
		r.data = nil
	}
}

// response builds a FUSE response.
type response struct {
	data []byte
}

// newResponse creates a new response for the given request.
func newResponse(req *request, payloadSize int) *response {
	size := proto.OutHeaderSize + payloadSize
	data := make([]byte, size)

	// Write header
	binary.LittleEndian.PutUint32(data[0:4], uint32(size))
	binary.LittleEndian.PutUint32(data[4:8], 0) // Error = 0 (success)
	binary.LittleEndian.PutUint64(data[8:16], req.header.Unique)

	return &response{data: data}
}

// newErrorResponse creates an error response.
func newErrorResponse(req *request, errno int32) *response {
	data := make([]byte, proto.OutHeaderSize)

	binary.LittleEndian.PutUint32(data[0:4], uint32(proto.OutHeaderSize))
	binary.LittleEndian.PutUint32(data[4:8], uint32(errno))
	binary.LittleEndian.PutUint64(data[8:16], req.header.Unique)

	return &response{data: data}
}

// payload returns the response payload area (after the header).
func (r *response) payload() []byte {
	return r.data[proto.OutHeaderSize:]
}

// setPayload sets the response payload directly.
func (r *response) setPayload(payload []byte) {
	r.data = make([]byte, proto.OutHeaderSize+len(payload))
	binary.LittleEndian.PutUint32(r.data[0:4], uint32(len(r.data)))
	copy(r.data[proto.OutHeaderSize:], payload)
}

// bytes returns the full response data.
func (r *response) bytes() []byte {
	return r.data
}

// Helper to read little-endian int32
func init() {
	// Verify we're on a little-endian system or handle byte order
	// For now, assume little-endian (Linux on x86/ARM)
}
