package psst

//go:generate stringer -type=connState

import (
	"container/list"
	"fmt"
	"math/rand"
	"time"
)

// Connection states
type connState int

const (
	closed connState = iota
	listen
	synSent
	synReceived
	open
	closeWait
)

type connConfig struct {
	MaxOutstandingSegmentsSelf uint16
	MaxOutstandingSegmentsPeer uint16
	RetransmissionTimeout      uint16
	CumulativeAckTimeout       uint16
	NulTimeout                 uint16
	MaxRetransmissions         uint8
	MaxCumulativeAck           uint8
	MaxOutOfSeq                uint8
}

type txBufferEntry struct {
	SeqNumber uint16
	txCount   uint8
	Data      []byte
}

type rxBufferEntry struct {
	SeqNumber uint16
	Data      []byte
}

type conn struct {
	state connState
	// Connection config
	config *connConfig
	// Transmitter state variables
	txNextSeq       uint16
	txOldestUnacked uint16
	txBuffer        *list.List
	// Receiver state variables
	rxLastInSeq uint16
	rxBuffer    *list.List
	// Timers
	retransmissionTimer *time.Timer
	cumulativeAckTimer  *time.Timer
	nulTimer            *time.Timer
}

func NewConn() *conn {
	initialSeqNumber := uint16(rand.Int())
	return &conn{
		state:           closed,
		txNextSeq:       initialSeqNumber + 1,
		txOldestUnacked: initialSeqNumber,
		txBuffer:        list.New(),
		rxBuffer:        list.New(),
	}
}

func (self *conn) handleSegment(segment *segment) error {
	if err := self.validateSegment(segment); err != nil {
		return err
	}

	switch self.state {
	case open:
		// Handle RST & break
		if segment.RST {
			// TODO Update state
			break
		}

		// Handle NUL & break
		if segment.NUL {
			// TODO validate SeqNumber and send ACK
			break
		}

		// Handle ACK
		if segment.ACK {
			// Check for positive unsigned diff AckNumber > txOldestUnacked
			if diff := segment.AckNumber - self.txOldestUnacked; diff != 0 && diff < self.config.MaxOutstandingSegmentsPeer {
				self.txOldestUnacked = segment.AckNumber
				self.clearAckedTxBuffer()
			}
		}

		// Handle EAK
		if segment.EAK {
			eakHeader := segment.VarHeader.(*eakVarHeader)
			for _, eak := range eakHeader.EakNumbers {
				self.removeFromTxBuffer(eak)
			}
		}

		// Handle data payload
		if len(segment.Data) > 0 {
			if segment.SeqNumber-self.rxLastInSeq == 1 {
				self.receivedData(segment.Data)
				self.rxLastInSeq++
				self.flushInSeqRxBuffer()
			}
		}
	default:
		return fmt.Errorf("State %s not implemented", self.state)
	}

	return nil
}

func (self *conn) validateSegment(segment *segment) error {
	// TODO Check for unexpected segment header
	return nil
}

func (self *conn) removeFromTxBuffer(SeqNumber uint16) {
	for element := self.txBuffer.Front(); element != nil; element = element.Next() {
		entry := element.Value.(*txBufferEntry)

		if entry.SeqNumber == SeqNumber {
			self.txBuffer.Remove(element)
			break
		}

		// Check for positive unsigned diff: buffer SeqNumber > input SeqNumber
		if diff := entry.SeqNumber - SeqNumber; diff != 0 && diff < self.config.MaxOutstandingSegmentsPeer {
			break
		}
	}
}

func (self *conn) clearAckedTxBuffer() {
	var next *list.Element
	for element := self.txBuffer.Front(); element != nil; element = next {
		entry := element.Value.(*txBufferEntry)

		// Check for positive unsigned diff: SeqNumber > txOldestUnacked
		if diff := entry.SeqNumber - self.txOldestUnacked; diff != 0 && diff < self.config.MaxOutstandingSegmentsPeer {
			break
		}

		next = element.Next()
		self.txBuffer.Remove(element)
	}
}

func (self *conn) receivedData(data []byte) {
	// TODO Forward new data to application layer
}

func (self *conn) flushInSeqRxBuffer() {
	var next *list.Element
	for element := self.rxBuffer.Front(); element != nil; element = next {
		entry := element.Value.(*rxBufferEntry)

		if entry.SeqNumber-self.rxLastInSeq != 1 {
			break
		}

		next = element.Next()
		self.rxBuffer.Remove(element)
		self.receivedData(entry.Data)
		self.rxLastInSeq++
	}
}
