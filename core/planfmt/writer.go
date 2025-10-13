package planfmt

import (
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
// Format: MAGIC(4) | VERSION(2) | FLAGS(2) | HEADER_LEN(4) | BODY_LEN(8) | ...
func (wr *Writer) WritePlan(p *Plan) ([32]byte, error) {
	// Step 1: Write magic number (4 bytes)
	if _, err := wr.w.Write([]byte(Magic)); err != nil {
		return [32]byte{}, err
	}

	// Step 2: Write version (2 bytes, little-endian)
	if err := binary.Write(wr.w, binary.LittleEndian, Version); err != nil {
		return [32]byte{}, err
	}

	// Step 3: Write flags (2 bytes, little-endian)
	// For now, no flags set (no compression, no signature)
	flags := Flags(0)
	if err := binary.Write(wr.w, binary.LittleEndian, uint16(flags)); err != nil {
		return [32]byte{}, err
	}

	// Step 4: Write header length (4 bytes, uint32, little-endian)
	// Header contains: SchemaID(16) + CreatedAt(8) + Compiler(16) + PlanKind(1) + Reserved(3) + Target(variable)
	// For now, write placeholder (will compute actual size later)
	headerLen := wr.computeHeaderLen(p)
	if err := binary.Write(wr.w, binary.LittleEndian, headerLen); err != nil {
		return [32]byte{}, err
	}

	// Step 5: Write body length (8 bytes, uint64, little-endian)
	// Body contains: TOC + all sections
	// For now, write placeholder (will compute actual size later)
	bodyLen := wr.computeBodyLen(p)
	if err := binary.Write(wr.w, binary.LittleEndian, bodyLen); err != nil {
		return [32]byte{}, err
	}

	// TODO: Write actual header, TOC, sections, hash

	// Return dummy hash for now (will implement BLAKE3 later)
	return [32]byte{}, nil
}

// computeHeaderLen computes the size of the header in bytes
func (wr *Writer) computeHeaderLen(p *Plan) uint32 {
	// SchemaID(16) + CreatedAt(8) + Compiler(16) + PlanKind(1) + Reserved(3) + TargetLen(2) + Target(variable)
	baseSize := 16 + 8 + 16 + 1 + 3 + 2
	targetSize := len(p.Target)
	return uint32(baseSize + targetSize)
}

// computeBodyLen computes the size of the body (TOC + sections) in bytes
func (wr *Writer) computeBodyLen(p *Plan) uint64 {
	// For now, return 0 (no sections yet)
	// Will implement when we add TOC and sections
	return 0
}
