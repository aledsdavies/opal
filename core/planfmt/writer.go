package planfmt

import (
	"bytes"
	"encoding/binary"
	"io"
)

const (
	// Magic is the file magic number "OPAL" (4 bytes)
	Magic = "OPAL"

	// Version is the format version (uint16, little-endian)
	// Version scheme: major.minor encoded as single uint16
	// 0x0001 = version 1.0
	// Breaking changes increment major, additions increment minor
	Version uint16 = 0x0001
)

// Flags is a bitmask for optional features
type Flags uint16

const (
	// FlagCompressed indicates STEPS and VALUES sections are zstd-compressed
	FlagCompressed Flags = 1 << 0

	// FlagSigned indicates a detached Ed25519 signature is present
	FlagSigned Flags = 1 << 1

	// Bits 2-15 reserved for future use
)

// Write writes a plan to w and returns the 32-byte file hash (BLAKE3).
// The plan is canonicalized before writing to ensure deterministic output.
func Write(w io.Writer, p *Plan) ([32]byte, error) {
	wr := &Writer{w: w}
	return wr.WritePlan(p)
}

// Writer handles writing plans to binary format.
type Writer struct {
	w io.Writer
}

// WritePlan writes the plan to the underlying writer.
// Format: MAGIC(4) | VERSION(2) | FLAGS(2) | HEADER_LEN(4) | BODY_LEN(8) | HEADER | BODY
func (wr *Writer) WritePlan(p *Plan) ([32]byte, error) {
	// Use buffer-then-write pattern: build header and body first, then write preamble with correct lengths
	var headerBuf, bodyBuf bytes.Buffer

	// Build header in buffer
	if err := wr.writeHeader(&headerBuf, p); err != nil {
		return [32]byte{}, err
	}

	// Build body in buffer (TODO: implement sections)
	if err := wr.writeBody(&bodyBuf, p); err != nil {
		return [32]byte{}, err
	}

	// Now write preamble with actual lengths
	if err := wr.writePreamble(uint32(headerBuf.Len()), uint64(bodyBuf.Len())); err != nil {
		return [32]byte{}, err
	}

	// Write header
	if _, err := wr.w.Write(headerBuf.Bytes()); err != nil {
		return [32]byte{}, err
	}

	// Write body
	if _, err := wr.w.Write(bodyBuf.Bytes()); err != nil {
		return [32]byte{}, err
	}

	// TODO: Compute actual hash (BLAKE3)
	return [32]byte{}, nil
}

// writePreamble writes the fixed-size preamble (20 bytes)
func (wr *Writer) writePreamble(headerLen uint32, bodyLen uint64) error {
	// Magic number (4 bytes)
	if _, err := wr.w.Write([]byte(Magic)); err != nil {
		return err
	}

	// Version (2 bytes, little-endian)
	if err := binary.Write(wr.w, binary.LittleEndian, Version); err != nil {
		return err
	}

	// Flags (2 bytes, little-endian)
	flags := Flags(0) // No compression, no signature
	if err := binary.Write(wr.w, binary.LittleEndian, uint16(flags)); err != nil {
		return err
	}

	// Header length (4 bytes, uint32, little-endian)
	if err := binary.Write(wr.w, binary.LittleEndian, headerLen); err != nil {
		return err
	}

	// Body length (8 bytes, uint64, little-endian)
	if err := binary.Write(wr.w, binary.LittleEndian, bodyLen); err != nil {
		return err
	}

	return nil
}

// writeHeader writes the plan header to the buffer
func (wr *Writer) writeHeader(buf *bytes.Buffer, p *Plan) error {
	// Write PlanHeader struct (44 bytes fixed)
	// SchemaID (16 bytes)
	if _, err := buf.Write(p.Header.SchemaID[:]); err != nil {
		return err
	}

	// CreatedAt (8 bytes, uint64, little-endian)
	if err := binary.Write(buf, binary.LittleEndian, p.Header.CreatedAt); err != nil {
		return err
	}

	// Compiler (16 bytes)
	if _, err := buf.Write(p.Header.Compiler[:]); err != nil {
		return err
	}

	// PlanKind (1 byte)
	if err := buf.WriteByte(p.Header.PlanKind); err != nil {
		return err
	}

	// Reserved (3 bytes)
	if _, err := buf.Write([]byte{0, 0, 0}); err != nil {
		return err
	}

	// Target (variable length: 2-byte length prefix + string bytes)
	targetLen := uint16(len(p.Target))
	if err := binary.Write(buf, binary.LittleEndian, targetLen); err != nil {
		return err
	}
	if _, err := buf.WriteString(p.Target); err != nil {
		return err
	}

	return nil
}

// writeBody writes the plan body (TOC + sections) to the buffer
func (wr *Writer) writeBody(buf *bytes.Buffer, p *Plan) error {
	// TODO: Implement TOC and sections
	// For now, write nothing (empty body)
	return nil
}
