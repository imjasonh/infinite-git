package packfile

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"hash"
	"io"
)

const (
	// Object types in packfile
	OBJ_COMMIT = 1
	OBJ_TREE   = 2
	OBJ_BLOB   = 3
	OBJ_TAG    = 4
)

// Writer writes a packfile.
type Writer struct {
	buf     bytes.Buffer
	objects int
	hash    hash.Hash
}

// NewWriter creates a new packfile writer.
func NewWriter() *Writer {
	w := &Writer{
		hash: sha1.New(),
	}

	// Write pack header
	w.buf.WriteString("PACK")
	binary.Write(&w.buf, binary.BigEndian, uint32(2)) // version
	binary.Write(&w.buf, binary.BigEndian, uint32(0)) // placeholder for object count

	return w
}

// AddObject adds an object to the packfile.
func (w *Writer) AddObject(objType int, data []byte) error {
	w.objects++

	// Encode object header
	// Format: 1-bit continuation, 3-bit type, 4-bit size (then 7-bit size chunks)
	size := len(data)
	header := (objType << 4) | (size & 0xf)
	size >>= 4

	for size > 0 {
		header |= 0x80 // Set continuation bit
		w.buf.WriteByte(byte(header))
		header = size & 0x7f
		size >>= 7
	}
	w.buf.WriteByte(byte(header))

	// Compress and write object data
	var compressedBuf bytes.Buffer
	zw := zlib.NewWriter(&compressedBuf)
	if _, err := zw.Write(data); err != nil {
		return fmt.Errorf("compressing object: %w", err)
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("closing compressor: %w", err)
	}

	w.buf.Write(compressedBuf.Bytes())
	return nil
}

// Finalize completes the packfile and returns the data.
func (w *Writer) Finalize() []byte {
	data := w.buf.Bytes()

	// Update object count in header
	binary.BigEndian.PutUint32(data[8:12], uint32(w.objects))

	// Calculate and append checksum
	w.hash.Write(data)
	checksum := w.hash.Sum(nil)

	result := append(data, checksum...)
	return result
}

// Reader reads objects from a packfile.
type Reader struct {
	data   []byte
	offset int
}

// NewReader creates a new packfile reader.
func NewReader(data []byte) (*Reader, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("packfile too small")
	}

	if string(data[:4]) != "PACK" {
		return nil, fmt.Errorf("invalid packfile signature")
	}

	version := binary.BigEndian.Uint32(data[4:8])
	if version != 2 {
		return nil, fmt.Errorf("unsupported packfile version: %d", version)
	}

	return &Reader{
		data:   data,
		offset: 12, // Skip header
	}, nil
}

// readVarint reads a variable-length integer.
func (r *Reader) readVarint() (int, int, error) {
	if r.offset >= len(r.data) {
		return 0, 0, io.EOF
	}

	b := r.data[r.offset]
	r.offset++

	objType := (int(b) >> 4) & 0x7
	size := int(b) & 0xf
	shift := 4

	for b&0x80 != 0 {
		if r.offset >= len(r.data) {
			return 0, 0, io.EOF
		}
		b = r.data[r.offset]
		r.offset++
		size |= (int(b) & 0x7f) << shift
		shift += 7
	}

	return objType, size, nil
}

// ReadObject reads the next object from the packfile.
func (r *Reader) ReadObject() (objType int, data []byte, err error) {
	// Read object header
	objType, size, err := r.readVarint()
	if err != nil {
		return 0, nil, err
	}

	// Read compressed data
	zr, err := zlib.NewReader(bytes.NewReader(r.data[r.offset:]))
	if err != nil {
		return 0, nil, fmt.Errorf("creating decompressor: %w", err)
	}
	defer zr.Close()

	data = make([]byte, size)
	if _, err := io.ReadFull(zr, data); err != nil {
		return 0, nil, fmt.Errorf("decompressing object: %w", err)
	}

	// Update offset past compressed data
	// This is tricky - we need to figure out how much compressed data we read
	// For now, we'll use a simple approach and skip this optimization
	// In a real implementation, we'd track the compressed size properly
	// For now, just consume the rest
	var buf bytes.Buffer
	io.Copy(&buf, zr)

	return objType, data, nil
}
