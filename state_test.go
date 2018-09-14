package psst

import (
	"testing"
)

func TestSimpleAckHandling(t *testing.T) {
	conn := NewConn()

	conn.state = stateOpen
	conn.config = defaultConfig()
	conn.txNextSeq = 1
	conn.txOldestUnacked = conn.txNextSeq - 1

	enqueueTxSegments(conn, 4)

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

	conn.state = stateOpen
	conn.config = defaultConfig()
	conn.txNextSeq = 0xFFFE
	conn.txOldestUnacked = conn.txNextSeq - 1

	enqueueTxSegments(conn, 6)

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

func TestIntWrappingAckHandling(t *testing.T) {
	conn := NewConn()

	conn.state = stateOpen
	conn.config = defaultConfig()
	conn.txNextSeq = 0x7FFE
	conn.txOldestUnacked = conn.txNextSeq - 1

	enqueueTxSegments(conn, 6)

	inputSegment := &segment{
		ACK:       true,
		SeqNumber: conn.rxLastInSeq + 1,
		AckNumber: 0x7FFE,
	}

	conn.handleSegment(inputSegment)
	validateTxBuffer(conn, []uint16{0x7FFF, 0x8000, 0x8001, 0x8002, 0x8003}, t)

	inputSegment = &segment{
		ACK:       true,
		SeqNumber: conn.rxLastInSeq + 1,
		AckNumber: 0x8001,
	}

	conn.handleSegment(inputSegment)
	validateTxBuffer(conn, []uint16{0x8002, 0x8003}, t)
}

func TestEakHandling(t *testing.T) {
	conn := NewConn()

	conn.state = stateOpen
	conn.config = defaultConfig()
	conn.txNextSeq = 1
	conn.txOldestUnacked = conn.txNextSeq - 1

	enqueueTxSegments(conn, 4)

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

func TestOutOfSeqRxBuffer(t *testing.T) {
	conn := NewConn()

	conn.state = stateOpen
	conn.config = defaultConfig()
	conn.rxLastInSeq = 0

	inputSegment := &segment{
		SeqNumber: 6,
		Data:      []byte{0},
	}

	conn.handleSegment(inputSegment)
	validateRxBuffer(conn, []uint16{6}, t)

	inputSegment = &segment{
		SeqNumber: 2,
		Data:      []byte{0},
	}

	conn.handleSegment(inputSegment)
	validateRxBuffer(conn, []uint16{2, 6}, t)

	inputSegment = &segment{
		SeqNumber: 3,
		Data:      []byte{0},
	}

	conn.handleSegment(inputSegment)
	validateRxBuffer(conn, []uint16{2, 3, 6}, t)

	inputSegment = &segment{
		SeqNumber: 7,
		Data:      []byte{0},
	}

	conn.handleSegment(inputSegment)
	validateRxBuffer(conn, []uint16{2, 3, 6, 7}, t)

	inputSegment = &segment{
		SeqNumber: 1,
		Data:      []byte{0},
	}

	conn.handleSegment(inputSegment)
	validateRxBuffer(conn, []uint16{6, 7}, t)
}

func TestClosedStateSegmentValidation(t *testing.T) {
	conn := NewConn()

	conn.state = stateClosed
	conn.config = defaultConfig()

	input := []struct {
		*segment
		action
	}{
		{&segment{SYN: true, VarHeader: &synVarHeader{}}, actionDiscard},
		{&segment{ACK: true, AckNumber: conn.txNextSeq - 1}, actionDiscard},
		{&segment{RST: true}, actionDiscard},
	}

	for _, i := range input {
		if action, _ := conn.validateSegment(i.segment); action != i.action {
			t.Fatalf("Segment validation action %v for segment %v doesn't match expected %v", action, i.segment, i.action)
		}
	}
}

func TestListenStateSegmentValidation(t *testing.T) {
	conn := NewConn()

	conn.state = stateListen
	conn.config = defaultConfig()

	input := []struct {
		*segment
		action
	}{
		{&segment{SYN: true, VarHeader: &synVarHeader{}}, actionContinue},
		{&segment{ACK: true, AckNumber: conn.txNextSeq - 1}, actionDiscard},
		{&segment{RST: true}, actionDiscard},
		{&segment{SYN: true, ACK: true, AckNumber: conn.txNextSeq - 1, VarHeader: &synVarHeader{}}, actionDiscard},
		{&segment{SYN: true}, actionReset},
	}

	for _, i := range input {
		if action, _ := conn.validateSegment(i.segment); action != i.action {
			t.Fatalf("Segment validation action %v for segment %v doesn't match expected %v", action, i.segment, i.action)
		}
	}
}

func TestSynSentStateSegmentValidation(t *testing.T) {
	conn := NewConn()

	conn.state = stateSynSent
	conn.config = defaultConfig()

	input := []struct {
		*segment
		action
	}{
		{&segment{RST: true}, actionContinue},
		{&segment{SYN: true, VarHeader: &synVarHeader{}}, actionContinue},
		{&segment{SYN: true, ACK: true, AckNumber: conn.txNextSeq - 1, VarHeader: &synVarHeader{}}, actionContinue},
		{&segment{ACK: true, AckNumber: conn.txNextSeq - 1}, actionDiscard},
		{&segment{NUL: true}, actionDiscard},
		{&segment{SYN: true}, actionReset},
	}

	for _, i := range input {
		if action, _ := conn.validateSegment(i.segment); action != i.action {
			t.Fatalf("Segment validation action %v for segment %v doesn't match expected %v", action, i.segment, i.action)
		}
	}
}

func TestSynReceivedStateSegmentValidation(t *testing.T) {
	conn := NewConn()

	conn.state = stateSynReceived
	conn.config = defaultConfig()
	conn.txNextSeq = 0x1234

	input := []struct {
		*segment
		action
	}{
		{&segment{RST: true}, actionContinue},
		{&segment{ACK: true, SeqNumber: conn.rxLastInSeq + 1, AckNumber: conn.txNextSeq - 1}, actionContinue},
		{&segment{ACK: true, SeqNumber: conn.rxLastInSeq + 1, AckNumber: conn.txNextSeq - 1, Data: []byte{0}}, actionContinue},
		{&segment{ACK: true, SeqNumber: conn.rxLastInSeq + conn.config.MaxOutstandingSegmentsSelf, AckNumber: conn.txNextSeq - 1, Data: []byte{0}}, actionContinue},
		{&segment{ACK: true, NUL: true, SeqNumber: conn.rxLastInSeq + 1, AckNumber: conn.txNextSeq - 1}, actionContinue},
		{&segment{NUL: true, SeqNumber: conn.rxLastInSeq + 1}, actionDiscard},
		{&segment{SeqNumber: conn.rxLastInSeq + 1, Data: []byte{0}}, actionDiscard},
		{&segment{SYN: true, VarHeader: &synVarHeader{}}, actionReset},
		{&segment{SYN: true, ACK: true, AckNumber: conn.txNextSeq - 1, VarHeader: &synVarHeader{}}, actionReset},
		{&segment{ACK: true, SeqNumber: conn.rxLastInSeq + 1, AckNumber: conn.txNextSeq}, actionReset},
		{&segment{ACK: true, SeqNumber: conn.rxLastInSeq + 1, AckNumber: conn.txNextSeq - 2}, actionReset},
		{&segment{ACK: true, SeqNumber: conn.rxLastInSeq + (3 * conn.config.MaxOutstandingSegmentsSelf), AckNumber: conn.txNextSeq - 1, Data: []byte{0}}, actionAck},
	}

	for _, i := range input {
		if action, _ := conn.validateSegment(i.segment); action != i.action {
			t.Fatalf("Segment validation action %v for segment %v doesn't match expected %v", action, i.segment, i.action)
		}
	}
}

func TestOpenStateSegmentValidation(t *testing.T) {
	conn := NewConn()

	conn.state = stateOpen
	conn.config = defaultConfig()
	conn.txNextSeq = 0x1234
	conn.txOldestUnacked = conn.txNextSeq - 10

	input := []struct {
		*segment
		action
	}{
		{&segment{RST: true}, actionContinue},
		{&segment{ACK: true, SeqNumber: conn.rxLastInSeq + 1, AckNumber: conn.txOldestUnacked}, actionContinue},
		{&segment{ACK: true, SeqNumber: conn.rxLastInSeq + 1, AckNumber: conn.txOldestUnacked, Data: []byte{0}}, actionContinue},
		{&segment{ACK: true, SeqNumber: conn.rxLastInSeq + conn.config.MaxOutstandingSegmentsSelf, AckNumber: conn.txOldestUnacked, Data: []byte{0}}, actionContinue},
		{&segment{ACK: true, EAK: true, SeqNumber: conn.rxLastInSeq + 1, AckNumber: conn.txOldestUnacked, VarHeader: &eakVarHeader{EakNumbers: []uint16{conn.txOldestUnacked + 2}}}, actionContinue},
		{&segment{ACK: true, NUL: true, SeqNumber: conn.rxLastInSeq + 1, AckNumber: conn.txOldestUnacked}, actionContinue},
		{&segment{NUL: true, SeqNumber: conn.rxLastInSeq + 1}, actionContinue},
		{&segment{SeqNumber: conn.rxLastInSeq + 1, Data: []byte{0}}, actionContinue},
		{&segment{NUL: true, SeqNumber: conn.rxLastInSeq + 1, Data: []byte{0}}, actionDiscard},
		{&segment{ACK: true, SeqNumber: conn.rxLastInSeq + 1, AckNumber: conn.txNextSeq}, actionDiscard},
		{&segment{EAK: true, SeqNumber: conn.rxLastInSeq + 1, AckNumber: conn.txOldestUnacked, VarHeader: &eakVarHeader{EakNumbers: []uint16{conn.txOldestUnacked + 2}}}, actionDiscard},
		{&segment{ACK: true, EAK: true, SeqNumber: conn.rxLastInSeq + 1, AckNumber: conn.txOldestUnacked, VarHeader: &eakVarHeader{EakNumbers: []uint16{conn.txOldestUnacked + 20}}}, actionDiscard},
		{&segment{ACK: true, EAK: true, SeqNumber: conn.rxLastInSeq + 1, AckNumber: conn.txOldestUnacked + 5, VarHeader: &eakVarHeader{EakNumbers: []uint16{conn.txOldestUnacked + 2}}}, actionDiscard},
		{&segment{SYN: true, SeqNumber: conn.rxLastInSeq + 1, VarHeader: &synVarHeader{}}, actionReset},
		{&segment{ACK: true, EAK: true, SeqNumber: conn.rxLastInSeq + 1, AckNumber: conn.txOldestUnacked}, actionReset},
		{&segment{SYN: true, SeqNumber: conn.rxLastInSeq, VarHeader: &synVarHeader{}}, actionAck},
		{&segment{ACK: true, SeqNumber: conn.rxLastInSeq + (3 * conn.config.MaxOutstandingSegmentsSelf), AckNumber: conn.txNextSeq - 1, Data: []byte{0}}, actionAck},
	}

	for _, i := range input {
		if action, _ := conn.validateSegment(i.segment); action != i.action {
			t.Fatalf("Segment validation action %v for segment %v doesn't match expected %v", action, i.segment, i.action)
		}
	}
}

func TestCloseWaitStateSegmentValidation(t *testing.T) {
	conn := NewConn()

	conn.state = stateCloseWait
	conn.config = defaultConfig()
	conn.txNextSeq = 0x1234

	input := []struct {
		*segment
		action
	}{
		{&segment{RST: true}, actionContinue},
		{&segment{ACK: true, SeqNumber: conn.rxLastInSeq + 1, AckNumber: conn.txNextSeq - 1}, actionDiscard},
		{&segment{NUL: true, SeqNumber: conn.rxLastInSeq + 1}, actionDiscard},
		{&segment{SeqNumber: conn.rxLastInSeq + 1, Data: []byte{0}}, actionDiscard},
		{&segment{SYN: true, VarHeader: &synVarHeader{}}, actionDiscard},
	}

	for _, i := range input {
		if action, _ := conn.validateSegment(i.segment); action != i.action {
			t.Fatalf("Segment validation action %v for segment %v doesn't match expected %v", action, i.segment, i.action)
		}
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

func enqueueTxSegments(conn *conn, count int) {
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

func validateRxBuffer(conn *conn, seqNumbers []uint16, t *testing.T) {
	if len(seqNumbers) != conn.rxBuffer.Len() {
		t.Fatalf("rxBuffer length %d doesn't match expected length %d", conn.rxBuffer.Len(), len(seqNumbers))
	}

	element := conn.rxBuffer.Front()
	for i, seq := range seqNumbers {
		if seq != element.Value.(*rxBufferEntry).SeqNumber {
			t.Fatalf("rxBuffer SeqNumber %d at position %d doesn't match expected %d", element.Value.(*rxBufferEntry).SeqNumber, i, seq)
		}
		element = element.Next()
	}
}
