package planfmt_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/aledsdavies/opal/core/planfmt"
)

// TestWriteEmptyPlan verifies we can write a minimal plan with correct magic and version
func TestWriteEmptyPlan(t *testing.T) {
	// Given: empty plan
	plan := &planfmt.Plan{}

	// When: write to buffer
	var buf bytes.Buffer
	hash, err := planfmt.Write(&buf, plan)

	// Then: no error, valid hash, valid magic number
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if len(hash) != 32 {
		t.Errorf("Expected 32-byte hash, got %d", len(hash))
	}

	// Verify magic number "OPAL"
	data := buf.Bytes()
	if len(data) < 4 {
		t.Fatalf("Output too short: %d bytes", len(data))
	}

	magic := string(data[0:4])
	if magic != "OPAL" {
		t.Errorf("Expected magic 'OPAL', got %q", magic)
	}

	// Verify version is present (bytes 4-5, little-endian uint16)
	if len(data) < 6 {
		t.Fatalf("Output missing version: %d bytes", len(data))
	}
}

// TestWriteFlags verifies flags field is written correctly
func TestWriteFlags(t *testing.T) {
	// Given: empty plan (no compression, no signature)
	plan := &planfmt.Plan{}

	// When: write to buffer
	var buf bytes.Buffer
	_, err := planfmt.Write(&buf, plan)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Then: flags field should be 0x0000 (no flags set)
	data := buf.Bytes()
	if len(data) < 8 {
		t.Fatalf("Output too short for flags: %d bytes", len(data))
	}

	// Flags are at offset 6-7 (after magic + version)
	flags := binary.LittleEndian.Uint16(data[6:8])
	if flags != 0 {
		t.Errorf("Expected flags 0x0000, got 0x%04x", flags)
	}
}

// TestWriteHeaderLengths verifies header and body length fields
func TestWriteHeaderLengths(t *testing.T) {
	// Given: empty plan
	plan := &planfmt.Plan{
		Target: "deploy",
	}

	// When: write to buffer
	var buf bytes.Buffer
	_, err := planfmt.Write(&buf, plan)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Then: verify we have header length and body length fields
	data := buf.Bytes()

	// Format: MAGIC(4) | VERSION(2) | FLAGS(2) | HEADER_LEN(4) | BODY_LEN(8)
	// Minimum size: 4 + 2 + 2 + 4 + 8 = 20 bytes
	if len(data) < 20 {
		t.Fatalf("Output too short for header lengths: %d bytes", len(data))
	}

	// HEADER_LEN at offset 8-11 (uint32, little-endian)
	headerLen := binary.LittleEndian.Uint32(data[8:12])
	if headerLen == 0 {
		t.Error("Expected non-zero header length")
	}

	// BODY_LEN at offset 12-19 (uint64, little-endian)
	bodyLen := binary.LittleEndian.Uint64(data[12:20])
	// Body length can be 0 for empty plan, just verify field exists
	_ = bodyLen
}
