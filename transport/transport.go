// Network communication layer, responsible for the bottom layer of network communication,
// mainly including tcp && udp two protocol implementation
package transport

import (
	"context"
	"encoding/binary"
	"io"
	"net"

	"github.com/xing-you-ji/novarpc/codec"
	"github.com/xing-you-ji/novarpc/codes"
)

const DefaultPayloadLength = 1024

// 定义了一个最大的数据包长度，4M
const MaxPayloadLength = 4 * 1024 * 1024

// ServerTransport 提供一种监听和处理请求的机制，实现成接口，主要是为了实现可插拔，支持业务自定义（比如支持HTTP协议）
type ServerTransport interface {
	// monitoring and processing of requests
	ListenAndServe(context.Context, ...ServerTransportOption) error
}

// Send 这个方法主要是用来发起请求调用，传参除了上下文 context 之外，还有二进制的请求包 request，返回是一个二进制的完整数据帧。这里设计成 interface 接口的形式，同样是为了可插拔、支持业务自定义
type ClientTransport interface {
	// send requests
	Send(context.Context, []byte, ...ClientTransportOption) ([]byte, error)
}

// Framer defines the reading of data frames from a data stream
type Framer interface {
	// read a full frame
	ReadFrame(net.Conn) ([]byte, error)
}

type framer struct {
	buffer  []byte // 1024 byte, 不够的话会扩容
	counter int    // to prevent the dead loop
}

// Create a Framer
func NewFramer() Framer {
	return &framer{
		buffer: make([]byte, DefaultPayloadLength),
	}
}

func (f *framer) Resize() {
	f.buffer = make([]byte, len(f.buffer)*2)
}

func (f *framer) ReadFrame(conn net.Conn) ([]byte, error) {

	// 读取帧头
	frameHeader := make([]byte, codec.FrameHeadLen)
	if num, err := io.ReadFull(conn, frameHeader); num != codec.FrameHeadLen || err != nil {
		return nil, err
	}

	// 验证魔数是否正确
	if magic := frameHeader[0]; magic != codec.Magic {
		return nil, codes.NewFrameworkError(codes.ClientMsgErrorCode, "invalid magic...")
	}
	// 读取请求包长度(正好4个字节)
	length := binary.BigEndian.Uint32(frameHeader[7:11])

	// 如果请求包长度大于最大的数据包长度，那么就返回一个错误
	if length > MaxPayloadLength {
		return nil, codes.NewFrameworkError(codes.ClientMsgErrorCode, "payload too large...")
	}

	// 如果请求包长度大于当前缓冲区的长度，那么就扩容
	for uint32(len(f.buffer)) < length && f.counter <= 12 {
		f.buffer = make([]byte, len(f.buffer)*2)
		f.counter++
	}

	// 读取请求包
	if num, err := io.ReadFull(conn, f.buffer[:length]); uint32(num) != length || err != nil {
		return nil, err
	}

	return append(frameHeader, f.buffer[:length]...), nil
}
