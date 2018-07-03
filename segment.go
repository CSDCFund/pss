package psst

import (
	"encoding"
	"encoding/binary"
)

// Segment format
//
//  0             0 0   1         1
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5
// +-+-+-+-+-+-----+---------------+
// |S|A|E|R|N|Reser|    Header     |
// |Y|C|A|S|U|-ved |    Length     |
// |N|K|K|T|L| (0) |(16-bit words) |
// +-+-+-+-+-+-----+---------------+
// |        Sequence Number        |
// +---------------+---------------+
// |    Acknowledgement Number     |
// +---------------+---------------+
// |     Data Length (octets)      |
// +---------------+---------------+
// |   Variable Header Section     |
// .                               .
// .                               .
// |                               |
// +---------------+---------------+
// |          Data Section         |
// .                               .
// .                               .
// |                               |
// +---------------+---------------+

type segment struct {
	SYN, ACK, EAK, RST, NUL bool
	SeqNumber               uint16
	AckNumber               uint16
	VarHeader
	Data []byte
}

type VarHeader encoding.BinaryMarshaler

func (self *segment) MarshalBinary() ([]byte, error) {
	var mashaledVarHeader []byte
	if self.VarHeader != nil {
		var err error
		mashaledVarHeader, err = self.VarHeader.MarshalBinary()

		if err != nil {
			return nil, err
		}
	}

	headerLength := 8 + len(mashaledVarHeader)
	dataLength := len(self.Data)
	buffer := make([]byte, headerLength+dataLength)

	// Pack variables
	buffer[0] = self.encodeFlags()
	buffer[1] = byte(headerLength >> 1)
	binary.BigEndian.PutUint16(buffer[2:], self.SeqNumber)
	binary.BigEndian.PutUint16(buffer[4:], self.AckNumber)
	binary.BigEndian.PutUint16(buffer[6:], uint16(dataLength))
	copy(buffer[8:], mashaledVarHeader)
	copy(buffer[headerLength:], self.Data)

	return buffer, nil
}

func (self *segment) encodeFlags() uint8 {
	var flags uint8

	if self.SYN {
		flags |= 1 << 7
	}
	if self.ACK {
		flags |= 1 << 6
	}
	if self.EAK {
		flags |= 1 << 5
	}
	if self.RST {
		flags |= 1 << 4
	}
	if self.NUL {
		flags |= 1 << 3
	}

	return flags
}

// Variable header fields

// SYN header format
//
//  0             0 0   1         1
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5
// +-+-+-+-+-+-----+---------------+
// | |A| | | | | | |               |
// |1|C|0|0|0|0|0|0|      12       |
// | |K| | | | | | |               |
// +-+-+-+-+-+-----+---------------+
// |        Sequence Number        |
// +---------------+---------------+
// |    Acknowledgement Number     |
// +---------------+---------------+
// |               0               |
// +---------------+---------------+
// |  Version = 1  |     Spare     |
// +---------------+---------------+
// |      Maximum Segment Size     |
// +---------------+---------------+
// | Max # of Outstanding Segments |
// +---------------+---------------+
// | Retransmission Timeout Value  |
// +---------------+---------------+
// | Cumulative Ack Timeout Value  |
// +---------------+---------------+
// |   Null Segment Timeout Value  |
// +---------------+---------------+
// |  Max Retrans  | Max Cum Ack   |
// +---------------+---------------+
// | Max Out of Seq| Max Auto Reset|
// +---------------+---------------+

type synVarHeader struct {
	Version                uint8
	MaxSegmentSize         uint16
	MaxOutstandingSegments uint16
	RetransmissionTimeout  uint16
	CumulativeAckTimeout   uint16
	NulTimeout             uint16
	MaxRetransmissions     uint8
	MaxCumulativeAck       uint8
	MaxOutOfSeq            uint8
	MaxAutoReset           uint8
}

func (self *synVarHeader) MarshalBinary() ([]byte, error) {
	buffer := make([]byte, 16)

	buffer[0] = self.Version
	binary.BigEndian.PutUint16(buffer[2:], self.MaxSegmentSize)
	binary.BigEndian.PutUint16(buffer[4:], self.MaxOutstandingSegments)
	binary.BigEndian.PutUint16(buffer[6:], self.RetransmissionTimeout)
	binary.BigEndian.PutUint16(buffer[8:], self.CumulativeAckTimeout)
	binary.BigEndian.PutUint16(buffer[10:], self.NulTimeout)
	buffer[12] = self.MaxRetransmissions
	buffer[13] = self.MaxCumulativeAck
	buffer[14] = self.MaxOutOfSeq
	buffer[15] = self.MaxAutoReset

	return buffer, nil
}

type eakVarHeader struct {
	EakNumbers []uint16
}

func (self *eakVarHeader) MarshalBinary() ([]byte, error) {
	buffer := make([]byte, 2*len(self.EakNumbers))

	for i := 0; i < len(self.EakNumbers); i++ {
		binary.BigEndian.PutUint16(buffer[i*2:], self.EakNumbers[i])
	}

	return buffer, nil
}
