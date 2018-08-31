package psst

import (
	"testing"
)

func TestSimpleAckHandling(t *testing.T) {
	conn := NewConn()

	conn.state = open
	conn.config = defaultConfig()
	conn.txNextSeq = 1
	conn.txOldestUnacked = 0

	enqueueSegments(conn, 4)

	inputSegment := &segment{
		ACK:       true,
		SeqNumber: conn.rxLastInSeq,
		AckNumber: 2,
	}

	conn.handleSegment(inputSegment)

	if conn.txOldestUnacked != 2 {
		t.Fatalf("Failed to update AckNumber")
	}

	if conn.txBuffer.Len() != 2 || conn.txBuffer.Front().Value.(*txBufferEntry).SeqNumber != 3 {
		t.Fatalf("Failed to clear acked txBuffer")
	}
}

func TestWrappingAckHandling(t *testing.T) {
	conn := NewConn()

	conn.state = open
	conn.config = defaultConfig()
	conn.txNextSeq = 0xFFFE
	conn.txOldestUnacked = 0xFFFD

	enqueueSegments(conn, 6)

	inputSegment := &segment{
		ACK:       true,
		SeqNumber: conn.rxLastInSeq,
		AckNumber: 0xFFFE,
	}

	conn.handleSegment(inputSegment)

	if conn.txOldestUnacked != 0xFFFE {
		t.Fatalf("Failed to update AckNumber")
	}

	if conn.txBuffer.Len() != 5 || conn.txBuffer.Front().Value.(*txBufferEntry).SeqNumber != 0xFFFF {
		t.Fatalf("Failed to clear acked txBuffer")
	}

	inputSegment = &segment{
		ACK:       true,
		SeqNumber: conn.rxLastInSeq,
		AckNumber: 1,
	}

	conn.handleSegment(inputSegment)

	if conn.txOldestUnacked != 1 {
		t.Fatalf("Failed to update AckNumber")
	}

	if conn.txBuffer.Len() != 2 || conn.txBuffer.Front().Value.(*txBufferEntry).SeqNumber != 2 {
		t.Fatalf("Failed to clear acked txBuffer")
	}
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
