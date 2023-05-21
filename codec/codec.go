package codec

import (
	"bytes"
	"encoding/binary"
	"math"
	"sync"

	"github.com/golang/protobuf/proto"
)

// Codec defines the codec specification for data
type Codec interface {
	Encode([]byte) ([]byte, error)
	Decode([]byte) ([]byte, error)
}

const FrameHeadLen = 15 // frame header length
const Magic = 0x11      // magic
const Version = 0       // version

// FrameHeader describes the header structure of a data frame
type FrameHeader struct {
	Magic        uint8  // 魔数
	Version      uint8  // version
	MsgType      uint8  // msg type e.g. :   0x0: general req,  0x1: heartbeat
	ReqType      uint8  // request type e.g.	 :   0x0: send and receive,   0x1: send but not receive,  0x2: client stream request, 0x3: server stream request, 0x4: bidirectional streaming request
	CompressType uint8  // compression or not :  0x0: not compression,  0x1: compression
	StreamID     uint16 // stream ID
	Length       uint32 // total packet length
	Reserved     uint32 // 4 bytes reserved
}

// GetCodec get a Codec by a codec name
func GetCodec(name string) Codec {
	if codec, ok := codecMap[name]; ok {
		return codec
	}
	return DefaultCodec
}

var codecMap = make(map[string]Codec)

// DefaultCodec defines the default codec
var DefaultCodec = NewCodec()

// NewCodec returns a globally unique codec
var NewCodec = func() Codec {
	return &defaultCodec{}
}

func init() {
	RegisterCodec("proto", DefaultCodec)
}

// RegisterCodec registers a codec, which will be added to codecMap
func RegisterCodec(name string, codec Codec) {
	if codecMap == nil {
		codecMap = make(map[string]Codec)
	}
	codecMap[name] = codec
}

func (c *defaultCodec) Encode(data []byte) ([]byte, error) {
	// 计算总长度
	totalLen := FrameHeadLen + len(data)
	buffer := bytes.NewBuffer(make([]byte, 0, totalLen))

	// 帧头（帧头目前固定 15 byte）
	frame := FrameHeader{
		Magic:        Magic,
		Version:      Version,
		MsgType:      0x0,
		ReqType:      0x0,
		CompressType: 0x0,
		Length:       uint32(len(data)),
	}

	// 写入魔数
	if err := binary.Write(buffer, binary.BigEndian, frame.Magic); err != nil {
		return nil, err
	}

	// 写入版本号
	if err := binary.Write(buffer, binary.BigEndian, frame.Version); err != nil {
		return nil, err
	}

	// 写入消息类型
	if err := binary.Write(buffer, binary.BigEndian, frame.MsgType); err != nil {
		return nil, err
	}

	// 写入请求类型
	if err := binary.Write(buffer, binary.BigEndian, frame.ReqType); err != nil {
		return nil, err
	}

	// 写入压缩类型
	if err := binary.Write(buffer, binary.BigEndian, frame.CompressType); err != nil {
		return nil, err
	}

	// 写入streamID
	if err := binary.Write(buffer, binary.BigEndian, frame.StreamID); err != nil {
		return nil, err
	}

	// 写入数据长度
	if err := binary.Write(buffer, binary.BigEndian, frame.Length); err != nil {
		return nil, err
	}

	// 写入保留字段
	if err := binary.Write(buffer, binary.BigEndian, frame.Reserved); err != nil {
		return nil, err
	}

	// 写入数据
	if err := binary.Write(buffer, binary.BigEndian, data); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (c *defaultCodec) Decode(requestBuf []byte) ([]byte, error) {
	// 我们只需要帧头 后面的数据
	return requestBuf[FrameHeadLen:], nil
}

type defaultCodec struct{}

func upperLimit(val int) uint32 {
	if val > math.MaxInt32 {
		return uint32(math.MaxInt32)
	}
	return uint32(val)
}

// bufferPool 用于缓存编码后的数据
var bufferPool = &sync.Pool{
	New: func() interface{} {
		return &cachedBuffer{
			Buffer:            proto.Buffer{},
			lastMarshaledSize: 16,
		}
	},
}

type cachedBuffer struct {
	proto.Buffer
	lastMarshaledSize uint32
}
