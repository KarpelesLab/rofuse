package rofuse

import (
	"encoding/binary"
	"syscall"
	"unsafe"

	"github.com/KarpelesLab/rofuse/proto"
)

// handler is a function that handles a FUSE request.
type handler func(s *Server, req *request) error

// handlers maps opcodes to their handlers.
var handlers = map[uint32]handler{
	proto.OpInit:        handleInit,
	proto.OpDestroy:     handleDestroy,
	proto.OpLookup:      handleLookup,
	proto.OpForget:      handleForget,
	proto.OpBatchForget: handleBatchForget,
	proto.OpGetattr:     handleGetattr,
	proto.OpReadlink:    handleReadlink,
	proto.OpOpen:        handleOpen,
	proto.OpRead:        handleRead,
	proto.OpRelease:     handleRelease,
	proto.OpOpendir:     handleOpendir,
	proto.OpReaddir:     handleReaddir,
	proto.OpReaddirplus: handleReaddirplus,
	proto.OpReleasedir:  handleReleasedir,
	proto.OpStatfs:      handleStatfs,
	proto.OpAccess:      handleAccess,
	proto.OpFlush:       handleFlush,
	proto.OpInterrupt:   handleInterrupt,
}

// handleInit processes FUSE_INIT.
func handleInit(s *Server, req *request) error {
	in := (*proto.InitIn)(req.body())

	// Validate protocol version
	if in.Major != proto.FuseKernelVersion {
		// Major version mismatch - negotiate
		out := &proto.InitOut{
			Major: proto.FuseKernelVersion,
			Minor: proto.FuseKernelMinorVersion,
		}
		s.sendResponse(req, initOutBytes(out))
		return nil
	}

	if in.Minor < proto.MinSupportedMinor {
		return syscall.EPROTO
	}

	// Negotiate minor version
	minor := in.Minor
	if minor > proto.FuseKernelMinorVersion {
		minor = proto.FuseKernelMinorVersion
	}

	// Store negotiated version
	s.conn.protoMajor = in.Major
	s.conn.protoMinor = minor

	// Create config
	s.config = &Config{
		ProtoMajor:   in.Major,
		ProtoMinor:   minor,
		MaxReadahead: min(in.MaxReadahead, s.opts.MaxReadahead),
		MaxWrite:     s.opts.MaxWrite,
		MaxPages:     proto.DefaultMaxPages,
	}

	// Call filesystem Init
	ctx := s.newContext(req)
	if err := s.fs.Init(ctx, s.config); err != nil {
		return err
	}

	// Build response with capabilities we support
	var flags uint32 = 0

	// Read-only filesystem capabilities
	flags |= uint32(proto.CapAsyncRead)
	flags |= uint32(proto.CapParallelDirops)
	flags |= uint32(proto.CapAutoInvalData)
	flags |= uint32(proto.CapReaddirplus)
	flags |= uint32(proto.CapReaddirplusAuto)
	flags |= uint32(proto.CapCacheSymlinks)
	flags |= uint32(proto.CapExportSupport)
	flags |= uint32(proto.CapMaxPages)

	// Intersect with kernel capabilities
	flags &= in.Flags

	out := &proto.InitOut{
		Major:               proto.FuseKernelVersion,
		Minor:               minor,
		MaxReadahead:        s.config.MaxReadahead,
		Flags:               flags,
		MaxBackground:       s.opts.MaxBackground,
		CongestionThreshold: s.opts.MaxBackground * 3 / 4,
		MaxWrite:            s.opts.MaxWrite,
		TimeGran:            proto.DefaultTimeGran,
		MaxPages:            proto.DefaultMaxPages,
	}

	s.mu.Lock()
	s.initialized = true
	s.mu.Unlock()

	s.sendResponse(req, initOutBytes(out))
	return nil
}

// handleDestroy processes FUSE_DESTROY.
func handleDestroy(s *Server, req *request) error {
	ctx := s.newContext(req)
	s.fs.Destroy(ctx)

	s.mu.Lock()
	s.destroyed = true
	s.mu.Unlock()

	s.sendResponse(req, nil)
	return nil
}

// handleLookup processes FUSE_LOOKUP.
func handleLookup(s *Server, req *request) error {
	name := req.filename()

	ctx := s.newContext(req)
	entry, err := s.fs.Lookup(ctx, Inode(req.header.NodeID), name)
	if err != nil {
		return err
	}

	out := entryToProto(entry)
	s.sendResponse(req, entryOutBytes(out))
	return nil
}

// handleForget processes FUSE_FORGET (no reply).
func handleForget(s *Server, req *request) error {
	in := (*proto.ForgetIn)(req.body())

	ctx := s.newContext(req)
	s.fs.Forget(ctx, Inode(req.header.NodeID), in.Nlookup)

	// No reply for FORGET
	return nil
}

// handleBatchForget processes FUSE_BATCH_FORGET (no reply).
func handleBatchForget(s *Server, req *request) error {
	in := (*proto.BatchForgetIn)(req.body())
	body := req.bodyBytes()

	if len(body) < proto.BatchForgetInSize {
		return syscall.EINVAL
	}

	// Parse forget entries
	entries := make([]ForgetEntry, in.Count)
	offset := proto.BatchForgetInSize
	for i := uint32(0); i < in.Count; i++ {
		if offset+proto.ForgetOneSize > len(body) {
			break
		}
		one := (*proto.ForgetOne)(unsafe.Pointer(&body[offset]))
		entries[i] = ForgetEntry{
			Ino:     Inode(one.NodeID),
			Nlookup: one.Nlookup,
		}
		offset += proto.ForgetOneSize
	}

	ctx := s.newContext(req)
	s.fs.BatchForget(ctx, entries)

	// No reply for BATCH_FORGET
	return nil
}

// handleGetattr processes FUSE_GETATTR.
func handleGetattr(s *Server, req *request) error {
	in := (*proto.GetAttrIn)(req.body())

	var fh *FileHandle
	if in.Flags&proto.GetattrFh != 0 {
		h := FileHandle(in.Fh)
		fh = &h
	}

	ctx := s.newContext(req)
	attr, err := s.fs.GetAttr(ctx, Inode(req.header.NodeID), fh)
	if err != nil {
		return err
	}

	out := &proto.AttrOut{
		AttrValid:     1, // 1 second default
		AttrValidNsec: 0,
		Attr:          attrToProto(attr),
	}

	s.sendResponse(req, attrOutBytes(out))
	return nil
}

// handleReadlink processes FUSE_READLINK.
func handleReadlink(s *Server, req *request) error {
	ctx := s.newContext(req)
	target, err := s.fs.ReadLink(ctx, Inode(req.header.NodeID))
	if err != nil {
		return err
	}

	s.sendResponse(req, []byte(target))
	return nil
}

// handleOpen processes FUSE_OPEN.
func handleOpen(s *Server, req *request) error {
	in := (*proto.OpenIn)(req.body())

	ctx := s.newContext(req)
	resp, err := s.fs.Open(ctx, Inode(req.header.NodeID), in.Flags)
	if err != nil {
		return err
	}

	out := &proto.OpenOut{
		Fh:        uint64(resp.Handle),
		OpenFlags: uint32(resp.Flags),
	}

	s.sendResponse(req, openOutBytes(out))
	return nil
}

// handleRead processes FUSE_READ.
func handleRead(s *Server, req *request) error {
	in := (*proto.ReadIn)(req.body())

	ctx := s.newContext(req)
	data, err := s.fs.Read(
		ctx,
		Inode(req.header.NodeID),
		FileHandle(in.Fh),
		int64(in.Offset),
		in.Size,
	)
	if err != nil {
		return err
	}

	s.sendResponse(req, data)
	return nil
}

// handleRelease processes FUSE_RELEASE.
func handleRelease(s *Server, req *request) error {
	in := (*proto.ReleaseIn)(req.body())

	ctx := s.newContext(req)
	err := s.fs.Release(ctx, Inode(req.header.NodeID), FileHandle(in.Fh))
	if err != nil {
		return err
	}

	s.sendResponse(req, nil)
	return nil
}

// handleOpendir processes FUSE_OPENDIR.
func handleOpendir(s *Server, req *request) error {
	in := (*proto.OpenIn)(req.body())

	ctx := s.newContext(req)
	resp, err := s.fs.OpenDir(ctx, Inode(req.header.NodeID), in.Flags)
	if err != nil {
		return err
	}

	out := &proto.OpenOut{
		Fh:        uint64(resp.Handle),
		OpenFlags: uint32(resp.Flags),
	}

	s.sendResponse(req, openOutBytes(out))
	return nil
}

// handleReaddir processes FUSE_READDIR.
func handleReaddir(s *Server, req *request) error {
	in := (*proto.ReadIn)(req.body())

	ctx := s.newContext(req)
	entries, err := s.fs.ReadDir(
		ctx,
		Inode(req.header.NodeID),
		FileHandle(in.Fh),
		int64(in.Offset),
		in.Size,
	)
	if err != nil {
		return err
	}

	// Serialize directory entries
	data := serializeDirents(entries, in.Size)
	s.sendResponse(req, data)
	return nil
}

// handleReaddirplus processes FUSE_READDIRPLUS.
func handleReaddirplus(s *Server, req *request) error {
	in := (*proto.ReadIn)(req.body())

	ctx := s.newContext(req)
	entries, err := s.fs.ReadDirPlus(
		ctx,
		Inode(req.header.NodeID),
		FileHandle(in.Fh),
		int64(in.Offset),
		in.Size,
	)
	if err != nil {
		return err
	}

	// Serialize directory entries with attributes
	data := serializeDirentsPlus(entries, in.Size)
	s.sendResponse(req, data)
	return nil
}

// handleReleasedir processes FUSE_RELEASEDIR.
func handleReleasedir(s *Server, req *request) error {
	in := (*proto.ReleaseIn)(req.body())

	ctx := s.newContext(req)
	err := s.fs.ReleaseDir(ctx, Inode(req.header.NodeID), FileHandle(in.Fh))
	if err != nil {
		return err
	}

	s.sendResponse(req, nil)
	return nil
}

// handleStatfs processes FUSE_STATFS.
func handleStatfs(s *Server, req *request) error {
	ctx := s.newContext(req)
	st, err := s.fs.StatFS(ctx, Inode(req.header.NodeID))
	if err != nil {
		return err
	}

	out := &proto.StatfsOut{
		St: proto.Kstatfs{
			Blocks:  st.Blocks,
			Bfree:   st.Bfree,
			Bavail:  st.Bavail,
			Files:   st.Files,
			Ffree:   st.Ffree,
			Bsize:   st.Bsize,
			Namelen: st.Namelen,
			Frsize:  st.Frsize,
		},
	}

	s.sendResponse(req, statfsOutBytes(out))
	return nil
}

// handleAccess processes FUSE_ACCESS.
func handleAccess(s *Server, req *request) error {
	in := (*proto.AccessIn)(req.body())

	ctx := s.newContext(req)
	err := s.fs.Access(ctx, Inode(req.header.NodeID), in.Mask)
	if err != nil {
		return err
	}

	s.sendResponse(req, nil)
	return nil
}

// handleFlush processes FUSE_FLUSH.
func handleFlush(s *Server, req *request) error {
	// Read-only filesystem, nothing to flush
	s.sendResponse(req, nil)
	return nil
}

// handleInterrupt processes FUSE_INTERRUPT.
func handleInterrupt(s *Server, req *request) error {
	// We don't track interruptible operations currently
	// Just acknowledge the interrupt
	return nil
}

// Helper functions for serializing responses

func initOutBytes(out *proto.InitOut) []byte {
	data := make([]byte, proto.InitOutSize)
	binary.LittleEndian.PutUint32(data[0:], out.Major)
	binary.LittleEndian.PutUint32(data[4:], out.Minor)
	binary.LittleEndian.PutUint32(data[8:], out.MaxReadahead)
	binary.LittleEndian.PutUint32(data[12:], out.Flags)
	binary.LittleEndian.PutUint16(data[16:], out.MaxBackground)
	binary.LittleEndian.PutUint16(data[18:], out.CongestionThreshold)
	binary.LittleEndian.PutUint32(data[20:], out.MaxWrite)
	binary.LittleEndian.PutUint32(data[24:], out.TimeGran)
	binary.LittleEndian.PutUint16(data[28:], out.MaxPages)
	binary.LittleEndian.PutUint16(data[30:], out.MapAlignment)
	binary.LittleEndian.PutUint32(data[32:], out.Flags2)
	binary.LittleEndian.PutUint32(data[36:], out.MaxStackDepth)
	return data
}

func entryOutBytes(out *proto.EntryOut) []byte {
	data := make([]byte, proto.EntryOutSize)
	binary.LittleEndian.PutUint64(data[0:], out.NodeID)
	binary.LittleEndian.PutUint64(data[8:], out.Generation)
	binary.LittleEndian.PutUint64(data[16:], out.EntryValid)
	binary.LittleEndian.PutUint64(data[24:], out.AttrValid)
	binary.LittleEndian.PutUint32(data[32:], out.EntryValidNsec)
	binary.LittleEndian.PutUint32(data[36:], out.AttrValidNsec)
	writeAttr(data[40:], &out.Attr)
	return data
}

func attrOutBytes(out *proto.AttrOut) []byte {
	data := make([]byte, proto.AttrOutSize)
	binary.LittleEndian.PutUint64(data[0:], out.AttrValid)
	binary.LittleEndian.PutUint32(data[8:], out.AttrValidNsec)
	binary.LittleEndian.PutUint32(data[12:], out.Dummy)
	writeAttr(data[16:], &out.Attr)
	return data
}

func openOutBytes(out *proto.OpenOut) []byte {
	data := make([]byte, proto.OpenOutSize)
	binary.LittleEndian.PutUint64(data[0:], out.Fh)
	binary.LittleEndian.PutUint32(data[8:], out.OpenFlags)
	binary.LittleEndian.PutUint32(data[12:], out.Padding)
	return data
}

func statfsOutBytes(out *proto.StatfsOut) []byte {
	data := make([]byte, proto.StatfsOutSize)
	binary.LittleEndian.PutUint64(data[0:], out.St.Blocks)
	binary.LittleEndian.PutUint64(data[8:], out.St.Bfree)
	binary.LittleEndian.PutUint64(data[16:], out.St.Bavail)
	binary.LittleEndian.PutUint64(data[24:], out.St.Files)
	binary.LittleEndian.PutUint64(data[32:], out.St.Ffree)
	binary.LittleEndian.PutUint32(data[40:], out.St.Bsize)
	binary.LittleEndian.PutUint32(data[44:], out.St.Namelen)
	binary.LittleEndian.PutUint32(data[48:], out.St.Frsize)
	return data
}

func writeAttr(data []byte, attr *proto.Attr) {
	binary.LittleEndian.PutUint64(data[0:], attr.Ino)
	binary.LittleEndian.PutUint64(data[8:], attr.Size)
	binary.LittleEndian.PutUint64(data[16:], attr.Blocks)
	binary.LittleEndian.PutUint64(data[24:], attr.Atime)
	binary.LittleEndian.PutUint64(data[32:], attr.Mtime)
	binary.LittleEndian.PutUint64(data[40:], attr.Ctime)
	binary.LittleEndian.PutUint32(data[48:], attr.AtimeNsec)
	binary.LittleEndian.PutUint32(data[52:], attr.MtimeNsec)
	binary.LittleEndian.PutUint32(data[56:], attr.CtimeNsec)
	binary.LittleEndian.PutUint32(data[60:], attr.Mode)
	binary.LittleEndian.PutUint32(data[64:], attr.Nlink)
	binary.LittleEndian.PutUint32(data[68:], attr.Uid)
	binary.LittleEndian.PutUint32(data[72:], attr.Gid)
	binary.LittleEndian.PutUint32(data[76:], attr.Rdev)
	binary.LittleEndian.PutUint32(data[80:], attr.Blksize)
	binary.LittleEndian.PutUint32(data[84:], attr.Flags)
}

func entryToProto(entry *Entry) *proto.EntryOut {
	entrySec, entryNsec := durationToTimespec(entry.EntryTimeout)
	attrSec, attrNsec := durationToTimespec(entry.AttrTimeout)

	return &proto.EntryOut{
		NodeID:         uint64(entry.Ino),
		Generation:     entry.Generation,
		EntryValid:     entrySec,
		EntryValidNsec: entryNsec,
		AttrValid:      attrSec,
		AttrValidNsec:  attrNsec,
		Attr:           attrToProto(&entry.Attr),
	}
}

func serializeDirents(entries []DirEntry, maxSize uint32) []byte {
	buf := make([]byte, 0, maxSize)

	for _, entry := range entries {
		// Calculate entry size (padded to 8 bytes)
		nameLen := len(entry.Name)
		entrySize := proto.DirentSize + nameLen
		paddedSize := (entrySize + 7) &^ 7

		if uint32(len(buf)+paddedSize) > maxSize {
			break
		}

		// Write dirent header
		dirent := make([]byte, paddedSize)
		binary.LittleEndian.PutUint64(dirent[0:], uint64(entry.Ino))
		binary.LittleEndian.PutUint64(dirent[8:], entry.Offset)
		binary.LittleEndian.PutUint32(dirent[16:], uint32(nameLen))
		binary.LittleEndian.PutUint32(dirent[20:], entry.Type)
		copy(dirent[proto.DirentSize:], entry.Name)

		buf = append(buf, dirent...)
	}

	return buf
}

func serializeDirentsPlus(entries []DirEntryPlus, maxSize uint32) []byte {
	buf := make([]byte, 0, maxSize)

	for _, entry := range entries {
		// Calculate entry size (padded to 8 bytes)
		nameLen := len(entry.Name)
		entrySize := proto.DirentPlusSize + nameLen
		paddedSize := (entrySize + 7) &^ 7

		if uint32(len(buf)+paddedSize) > maxSize {
			break
		}

		// Write EntryOut + Dirent
		entryOut := entryToProto(&entry.Entry)
		entryOutData := entryOutBytes(entryOut)

		direntData := make([]byte, paddedSize-proto.EntryOutSize)
		binary.LittleEndian.PutUint64(direntData[0:], uint64(entry.Entry.Ino))
		binary.LittleEndian.PutUint64(direntData[8:], entry.Entry.Generation) // Use generation as offset
		binary.LittleEndian.PutUint32(direntData[16:], uint32(nameLen))
		binary.LittleEndian.PutUint32(direntData[20:], fileModeToType(entry.Entry.Attr.Mode))
		copy(direntData[proto.DirentSize:], entry.Name)

		buf = append(buf, entryOutData...)
		buf = append(buf, direntData...)
	}

	return buf
}
