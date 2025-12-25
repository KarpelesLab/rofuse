package sharing

import (
	"encoding/gob"
	"fmt"
	"net"
	"sync"
	"syscall"
)

// Coordinator manages multi-process FUSE serving.
// It holds the master FUSE FD and distributes cloned FDs to worker processes.
type Coordinator struct {
	sockPath string
	masterFd int
	listener *net.UnixListener

	workers   map[int]*Worker
	workersMu sync.RWMutex

	closed bool
	closeMu sync.Mutex
}

// Worker represents a connected worker process.
type Worker struct {
	PID    int
	conn   *net.UnixConn
	fd     int // The cloned FD for this worker
	closed bool
}

// RegisterMessage is sent by workers to register with the coordinator.
type RegisterMessage struct {
	PID int
}

// ResponseMessage is sent by the coordinator after registration.
type ResponseMessage struct {
	Success bool
	Error   string
}

// NewCoordinator creates a coordinator for multi-process serving.
// The masterFd should be the FUSE file descriptor from mounting.
// The sockPath is where workers will connect to receive their FDs.
func NewCoordinator(sockPath string, masterFd int) (*Coordinator, error) {
	passer, err := NewFDPasser(sockPath)
	if err != nil {
		return nil, err
	}

	return &Coordinator{
		sockPath: sockPath,
		masterFd: masterFd,
		listener: passer.listener,
		workers:  make(map[int]*Worker),
	}, nil
}

// AcceptWorker waits for a new worker process to connect and sends it a cloned FD.
// Returns the Worker on success. The worker's FD is automatically closed when
// the worker disconnects or when RemoveWorker is called.
func (c *Coordinator) AcceptWorker() (*Worker, error) {
	c.closeMu.Lock()
	if c.closed {
		c.closeMu.Unlock()
		return nil, fmt.Errorf("coordinator closed")
	}
	c.closeMu.Unlock()

	conn, err := c.listener.AcceptUnix()
	if err != nil {
		return nil, fmt.Errorf("accept: %w", err)
	}

	// Read registration message
	dec := gob.NewDecoder(conn)
	var reg RegisterMessage
	if err := dec.Decode(&reg); err != nil {
		conn.Close()
		return nil, fmt.Errorf("decode registration: %w", err)
	}

	// Clone the FUSE FD for this worker
	clonedFd, err := CloneFuseFD(c.masterFd)
	if err != nil {
		// Send error response
		enc := gob.NewEncoder(conn)
		enc.Encode(ResponseMessage{Success: false, Error: err.Error()})
		conn.Close()
		return nil, fmt.Errorf("clone fd: %w", err)
	}

	// Send success response
	enc := gob.NewEncoder(conn)
	if err := enc.Encode(ResponseMessage{Success: true}); err != nil {
		syscall.Close(clonedFd)
		conn.Close()
		return nil, fmt.Errorf("encode response: %w", err)
	}

	// Send the cloned FD to the worker
	if err := SendFD(conn, clonedFd); err != nil {
		syscall.Close(clonedFd)
		conn.Close()
		return nil, fmt.Errorf("send fd: %w", err)
	}

	worker := &Worker{
		PID:  reg.PID,
		conn: conn,
		fd:   clonedFd,
	}

	c.workersMu.Lock()
	c.workers[reg.PID] = worker
	c.workersMu.Unlock()

	return worker, nil
}

// RemoveWorker removes a worker and closes its resources.
func (c *Coordinator) RemoveWorker(pid int) {
	c.workersMu.Lock()
	worker, ok := c.workers[pid]
	if ok {
		delete(c.workers, pid)
	}
	c.workersMu.Unlock()

	if ok && worker != nil {
		worker.Close()
	}
}

// Workers returns a copy of the current worker list.
func (c *Coordinator) Workers() []*Worker {
	c.workersMu.RLock()
	defer c.workersMu.RUnlock()

	workers := make([]*Worker, 0, len(c.workers))
	for _, w := range c.workers {
		workers = append(workers, w)
	}
	return workers
}

// WorkerCount returns the number of active workers.
func (c *Coordinator) WorkerCount() int {
	c.workersMu.RLock()
	defer c.workersMu.RUnlock()
	return len(c.workers)
}

// Close closes the coordinator and all worker connections.
func (c *Coordinator) Close() error {
	c.closeMu.Lock()
	c.closed = true
	c.closeMu.Unlock()

	// Close all workers
	c.workersMu.Lock()
	for _, worker := range c.workers {
		worker.Close()
	}
	c.workers = make(map[int]*Worker)
	c.workersMu.Unlock()

	// Close listener
	return c.listener.Close()
}

// SockPath returns the socket path.
func (c *Coordinator) SockPath() string {
	return c.sockPath
}

// Close closes the worker's connection and FD.
func (w *Worker) Close() {
	if w.closed {
		return
	}
	w.closed = true

	if w.conn != nil {
		w.conn.Close()
	}
	if w.fd >= 0 {
		syscall.Close(w.fd)
		w.fd = -1
	}
}

// WorkerClient is used by worker processes to connect to a coordinator.
type WorkerClient struct {
	sockPath string
	conn     *net.UnixConn
	fd       int
}

// ConnectToCoordinator connects to a coordinator and receives a FUSE FD.
func ConnectToCoordinator(sockPath string, pid int) (*WorkerClient, error) {
	addr := &net.UnixAddr{Name: sockPath, Net: "unix"}
	conn, err := net.DialUnix("unix", nil, addr)
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}

	// Send registration
	enc := gob.NewEncoder(conn)
	if err := enc.Encode(RegisterMessage{PID: pid}); err != nil {
		conn.Close()
		return nil, fmt.Errorf("encode registration: %w", err)
	}

	// Read response
	dec := gob.NewDecoder(conn)
	var resp ResponseMessage
	if err := dec.Decode(&resp); err != nil {
		conn.Close()
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if !resp.Success {
		conn.Close()
		return nil, fmt.Errorf("coordinator error: %s", resp.Error)
	}

	// Receive the FUSE FD
	fd, err := ReceiveFD(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("receive fd: %w", err)
	}

	return &WorkerClient{
		sockPath: sockPath,
		conn:     conn,
		fd:       fd,
	}, nil
}

// Fd returns the FUSE file descriptor.
func (w *WorkerClient) Fd() int {
	return w.fd
}

// Close closes the worker client connection.
// Note: The FUSE FD should be closed separately after the server is done.
func (w *WorkerClient) Close() error {
	if w.conn != nil {
		return w.conn.Close()
	}
	return nil
}

// CloseFd closes the FUSE file descriptor.
func (w *WorkerClient) CloseFd() error {
	if w.fd >= 0 {
		err := syscall.Close(w.fd)
		w.fd = -1
		return err
	}
	return nil
}
