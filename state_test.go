package psst

import (
	"testing"
)

func TestSimpleAckHandling(t *testing.T) {
	conn := NewConn()

	conn.state = open
	conn.config = defaultConfig()
	conn.txNextSeq = 1
	conn.txOldestUnacked = conn.txNextSeq - 1

	enqueueSegments(conn, 4)

	inputSegment := &segment{
		ACK:       true,
		SeqNumber: conn.rxLastInSeq + 1,
		AckNumber: 2,
	}

	conn.handleSegment(inputSegment)
	validateTxBuffer(conn, []uint16{3, 4}, t)
}

func TestUintWrappingAckHandling(t *testing.T) {
	conn := NewConn()

	conn.state = open
	conn.config = defaultConfig()
	conn.txNextSeq = 0xFFFE
	conn.txOldestUnacked = conn.txNextSeq - 1

	enqueueSegments(conn, 6)

	inputSegment := &segment{
		ACK:       true,
		SeqNumber: conn.rxLastInSeq + 1,
		AckNumber: 0xFFFE,
	}

	conn.handleSegment(inputSegment)
	validateTxBuffer(conn, []uint16{0xFFFF, 0, 1, 2, 3}, t)

	inputSegment = &segment{
		ACK:       true,
		SeqNumber: conn.rxLastInSeq + 1,
		AckNumber: 1,
	}

	conn.handleSegment(inputSegment)
	validateTxBuffer(conn, []uint16{2, 3}, t)
}

func TestEakHandling(t *testing.T) {
	conn := NewConn()

	conn.state = open
	conn.config = defaultConfig()
	conn.txNextSeq = 1
	conn.txOldestUnacked = conn.txNextSeq - 1

	enqueueSegments(conn, 4)

	inputSegment := &segment{
		ACK:       true,
		EAK:       true,
		SeqNumber: conn.rxLastInSeq + 1,
		AckNumber: 1,
		VarHeader: &eakVarHeader{
			EakNumbers: []uint16{3, 4},
		},
	}

	conn.handleSegment(inputSegment)
	validateTxBuffer(conn, []uint16{2}, t)
}

func defaultConfig() *connConfig {
	return &connConfig{
		MaxOutstandingSegmentsSelf: 10,
		MaxOutstandingSegmentsPeer: 10,
		RetransmissionTimeout:      10,
		CumulativeAckTimeout:       10,
		NulTimeout:                 10,
		MaxRetransmissions:         10,
		MaxCumulativeAck:           10,
		MaxOutOfSeq:                10,
	}
}

func enqueueSegments(conn *conn, count int) {
	for i := 0; i < count; i++ {
		entry := &txBufferEntry{
			SeqNumber: conn.txNextSeq,
		}

		conn.txBuffer.PushBack(entry)
		conn.txNextSeq++
	}
}

func validateTxBuffer(conn *conn, seqNumbers []uint16, t *testing.T) {
	if len(seqNumbers) != conn.txBuffer.Len() {
		t.Fatalf("txBuffer length %d doesn't match expected length %d", conn.txBuffer.Len(), len(seqNumbers))
	}

	element := conn.txBuffer.Front()
	for i, seq := range seqNumbers {
		if seq != element.Value.(*txBufferEntry).SeqNumber {
			t.Fatalf("txBuffer SeqNumber %d at position %d doesn't match expected %d", element.Value.(*txBufferEntry).SeqNumber, i, seq)
		}
		element = element.Next()
	}
}
