package psst

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestBaseSerialization(t *testing.T) {
	segment := segment{
		SYN:       true,
		ACK:       true,
		SeqNumber: 0x1234,
		AckNumber: 0x5678,
	}

	expected := []byte{
		0xC0, 0x04,
		0x12, 0x34,
		0x56, 0x78,
		0x00, 0x00,
	}

	checkSegment(segment, expected, t)
}

func TestDataSerialization(t *testing.T) {
	segment := segment{
		ACK:       true,
		SeqNumber: 0x1234,
		AckNumber: 0x5678,
		Data:      []byte{0xba, 0xad, 0xbe, 0xef, 0x15, 0xba, 0xad},
	}

	expected := []byte{
		0x40, 0x04,
		0x12, 0x34,
		0x56, 0x78,
		0x00, 0x07,
		0xba, 0xad,
		0xbe, 0xef,
		0x15, 0xba,
		0xad,
	}

	checkSegment(segment, expected, t)
}

func TestLargeDataSerialization(t *testing.T) {
	data := make([]byte, 40000)
	segment := segment{
		ACK:       true,
		SeqNumber: 0x1234,
		AckNumber: 0x5678,
		Data:      data,
	}

	header := []byte{
		0x40, 0x04,
		0x12, 0x34,
		0x56, 0x78,
		0x9c, 0x40,
	}

	expected := make([]byte, 40008)
	copy(expected, header)

	checkSegment(segment, expected, t)
}

func TestSynSerialization(t *testing.T) {
	segment := segment{
		SYN:       true,
		SeqNumber: 0x1234,
		AckNumber: 0x5678,
		VarHeader: &synVarHeader{
			Version:                1,
			MaxSegmentSize:         16384,
			MaxOutstandingSegments: 16,
			RetransmissionTimeout:  4096,
			CumulativeAckTimeout:   2048,
			NulTimeout:             16384,
			MaxRetransmissions:     4,
			MaxCumulativeAck:       16,
			MaxOutOfSeq:            16,
			MaxAutoReset:           4,
		},
	}

	expected := []byte{
		0x80, 0x0C,
		0x12, 0x34,
		0x56, 0x78,
		0x00, 0x00,
		0x01, 0x00,
		0x40, 0x00,
		0x00, 0x10,
		0x10, 0x00,
		0x08, 0x00,
		0x40, 0x00,
		0x04, 0x10,
		0x10, 0x04,
	}

	checkSegment(segment, expected, t)
}

func TestEakSerialization(t *testing.T) {
	segment := segment{
		ACK:       true,
		EAK:       true,
		SeqNumber: 0x1234,
		AckNumber: 0x5678,
		VarHeader: &eakVarHeader{
			EakNumbers: []uint16{0x123a, 0x123b, 0x123c},
		},
	}

	expected := []byte{
		0x60, 0x07,
		0x12, 0x34,
		0x56, 0x78,
		0x00, 0x00,
		0x12, 0x3a,
		0x12, 0x3b,
		0x12, 0x3c,
	}

	checkSegment(segment, expected, t)
}

func checkSegment(seg segment, expected []byte, t *testing.T) {
	serialized, err := seg.MarshalBinary()

	if err != nil {
		t.Fatalf("Failed to serialze segment: %v", err)
	}

	if bytes.Compare(serialized, expected) != 0 {
		t.Fatalf("Serialzed segment didn't match expected: %v", hex.Dump(serialized))
	}
}
