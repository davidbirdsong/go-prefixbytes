package msgio

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"sync"

	msgio "github.com/jbenet/go-msgio"
	mpool "github.com/jbenet/go-msgio/mpool"
)

var PrefixBytes int = 2

func readInt(b []byte) uint32 {
	// equivalnt of return int32(binary.LittleEndian.Uint32(b))
	switch cap(b) {
	case 8:
		return uint32(b[0])
	case 16:
		return uint32(binary.LittleEndian.Uint16(b))
	case 32:
		return uint32(binary.LittleEndian.Uint32(b))
	case 64:
		return uint32(binary.LittleEndian.Uint64(b))
	}
	return 0

}

// fixedintReader is the underlying type that implements the Reader interface.
type fixedintReader struct {
	R    io.Reader
	lbuf []byte
	next int
	pool *mpool.Pool
	lock sync.Locker
}

// NewFixedintReader wraps an io.Reader with a fixedint framed reader.
// The msgio.Reader will read whole messages at a time (using the length).
func NewFixedintReader(r io.Reader, prefbytes int) msgio.ReadCloser {
	return NewFixedintReaderWithPool(
		bufio.NewReader(r),
		prefbytes,
		&mpool.ByteSlicePool,
	)
}

// NewFixedintReaderWithPool wraps an io.Reader with a fixedint msgio framed reader.
// The msgio.Reader will read whole messages at a time (using the length).
func NewFixedintReaderWithPool(r io.Reader, prefbytes int, p *mpool.Pool) msgio.ReadCloser {
	if p == nil {
		panic("nil pool")
	}
	return &fixedintReader{
		R:    r,
		lbuf: make([]byte, prefbytes),
		next: -1,
		pool: p,
		lock: new(sync.Mutex),
	}
}

// NextMsgLen reads the length of the next msg into s.lbuf, and returns it.
// WARNING: like Read, NextMsgLen is destructive. It reads from the internal
// reader.
func (s *fixedintReader) NextMsgLen() (int, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.nextMsgLen()
}

func (s *fixedintReader) nextMsgLen() (int, error) {
	if s.next == -1 {
		if n, err := s.R.Read(s.lbuf); err != nil {
			return 0, err
		} else if n != cap(s.lbuf) {
			return 0, fmt.Errorf("prefix buffer resized")
		}

		length := readInt(s.lbuf)
		s.next = int(length)
	}
	return s.next, nil
}

func (s *fixedintReader) Read(msg []byte) (int, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	length, err := s.nextMsgLen()
	if err != nil {
		return 0, err
	}

	if length > len(msg) {
		return 0, io.ErrShortBuffer
	}
	_, err = io.ReadFull(s.R, msg[:length])
	s.next = -1 // signal we've consumed this msg
	return length, err
}

func (s *fixedintReader) ReadMsg() ([]byte, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	length, err := s.nextMsgLen()
	if err != nil {
		return nil, err
	}

	msgb := s.pool.Get(uint32(length))
	if msgb == nil {
		return nil, io.ErrShortBuffer
	}
	msg := msgb.([]byte)[:length]
	_, err = io.ReadFull(s.R, msg)
	s.next = -1 // signal we've consumed this msg
	return msg, err
}

func (s *fixedintReader) ReleaseMsg(msg []byte) {
	s.pool.Put(uint32(cap(msg)), msg)
}

func (s *fixedintReader) Close() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if c, ok := s.R.(io.Closer); ok {
		return c.Close()
	}
	return nil
}
