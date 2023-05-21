package transport

import (
	"context"
	"go.uber.org/zap"
	"io"
	"net"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/xing-you-ji/novarpc/codec"
	"github.com/xing-you-ji/novarpc/codes"
	"github.com/xing-you-ji/novarpc/protocol"
	"github.com/xing-you-ji/novarpc/stream"
	"github.com/xing-you-ji/novarpc/utils"
)

type serverTransport struct {
	opts *ServerTransportOptions
}

var serverTransportMap = make(map[string]ServerTransport)

func init() {
	serverTransportMap["default"] = DefaultServerTransport
}

// RegisterServerTransport supports business custom registered ServerTransport
func RegisterServerTransport(name string, serverTransport ServerTransport) {
	if serverTransportMap == nil {
		serverTransportMap = make(map[string]ServerTransport)
	}
	serverTransportMap[name] = serverTransport
}

// Get the ServerTransport
func GetServerTransport(transport string) ServerTransport {

	if v, ok := serverTransportMap[transport]; ok {
		return v
	}

	return DefaultServerTransport
}

// The default server transport
var DefaultServerTransport = NewServerTransport()

// Use the singleton pattern to create a server transport
var NewServerTransport = func() ServerTransport {
	return &serverTransport{
		opts: &ServerTransportOptions{},
	}
}

func (s *serverTransport) ListenAndServe(ctx context.Context, opts ...ServerTransportOption) error {

	for _, o := range opts {
		o(s.opts)
	}

	switch s.opts.Network {
	case "tcp", "tcp4", "tcp6":
		return s.ListenAndServeTcp(ctx, opts...)
	case "udp", "udp4", "udp6":
		return s.ListenAndServeUdp(ctx, opts...)
	default:
		return codes.NetworkNotSupportedError
	}
}

func (s *serverTransport) ListenAndServeTcp(ctx context.Context, opts ...ServerTransportOption) error {

	listener, err := net.Listen(s.opts.Network, s.opts.Address)
	if err != nil {
		return err
	}

	go func() {
		if err = s.serve(ctx, listener); err != nil {
			zap.L().Error("transport serve error", zap.Error(err))
		}
	}()

	return nil
}

func (s *serverTransport) serve(ctx context.Context, listener net.Listener) error {

	var tempDelay time.Duration

	tcpListener, ok := listener.(*net.TCPListener)
	if !ok {
		return codes.NetworkNotSupportedError
	}

	for {

		// 检查是否有关闭信号
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		// 监听客户端连接
		conn, err := tcpListener.AcceptTCP()
		if err != nil {
			// 检查错误是否是暂时性的
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				// 最大延迟 1 秒
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				// 随机延迟一段时间后重试
				time.Sleep(tempDelay)
				continue
			}
			return err
		}

		// 开启 keepalive
		if err = conn.SetKeepAlive(true); err != nil {
			return err
		}

		// 设置 keepalive 时间间隔
		if s.opts.KeepAlivePeriod != 0 {
			conn.SetKeepAlivePeriod(s.opts.KeepAlivePeriod)
		}

		go func() {

			// 为每个连接创建一个上下文
			ctx, _ := stream.NewServerStream(ctx)

			if err := s.handleConn(ctx, wrapConn(conn)); err != nil {
				zap.L().Error("novaRPC handle tcp conn error", zap.Error(err))
			}

		}()
	}
}

// handleConn 处理客户端连接
func (s *serverTransport) handleConn(ctx context.Context, conn *connWrapper) error {
	// 关闭连接
	defer conn.Close()

	// 读取客户端请求
	for {
		// 监听关闭信号
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 读取客户端请求
		frame, err := s.read(ctx, conn)

		if err == io.EOF {
			// read completed
			return nil
		}
		// 如果是其他错误，直接返回
		if err != nil {
			return err
		}

		// 处理客户端请求
		rsp, err := s.handle(ctx, frame)
		if err != nil {
			zap.L().Error("novaRPC handle error", zap.Error(err))
		}

		// 响应客户端请求
		if err = s.write(ctx, conn, rsp); err != nil {
			return err
		}
	}

}

func (s *serverTransport) read(ctx context.Context, conn *connWrapper) ([]byte, error) {

	// 这个时候已经读取到了一个完整的 包
	request, err := conn.framer.ReadFrame(conn)
	if err != nil {
		return nil, err
	}

	return request, nil
}

func (s *serverTransport) handle(ctx context.Context, requestBuf []byte) ([]byte, error) {

	// 解码客户端请求
	serverCodec := codec.GetCodec(s.opts.Protocol)

	// 解码客户端请求
	request, err := serverCodec.Decode(requestBuf)
	if err != nil {
		zap.L().Error("novaRPC decode error", zap.Error(err))
		return nil, err
	}

	// 得到响应体
	responseBuf, err := s.opts.Handler.Handle(ctx, request)
	if err != nil {
		zap.L().Error("novaRPC handle error", zap.Error(err))
	}

	// 添加响应头
	response := addRspHeader(responseBuf, err)

	// 使用protobuf 序列化response
	rspPb, err := proto.Marshal(response)
	if err != nil {
		zap.L().Error("novaRPC proto marshal error", zap.Error(err))
		return nil, err
	}

	// 编码响应体(加入帧头)
	responseBody, err := serverCodec.Encode(rspPb)
	if err != nil {
		zap.L().Error("novaRPC encode error", zap.Error(err))
		return nil, err
	}

	return responseBody, nil
}

func addRspHeader(payload []byte, err error) *protocol.Response {
	response := &protocol.Response{
		Payload: payload,
		RetCode: codes.OK,
		RetMsg:  "success",
	}

	if err != nil {
		if e, ok := err.(*codes.Error); ok {
			response.RetCode = e.Code
			response.RetMsg = e.Message
		} else {
			response.RetCode = codes.ServerInternalErrorCode
			response.RetMsg = codes.ServerInternalError.Message
		}
	}

	return response
}

func (s *serverTransport) write(ctx context.Context, conn net.Conn, rsp []byte) error {
	if _, err := conn.Write(rsp); err != nil {
		zap.L().Error("novaRPC write error", zap.Error(err))
	}

	return nil
}

type connWrapper struct {
	net.Conn
	framer Framer
}

func wrapConn(rawConn net.Conn) *connWrapper {
	return &connWrapper{
		Conn:   rawConn,
		framer: NewFramer(),
	}
}

func (s *serverTransport) getServerStream(ctx context.Context, request *protocol.Request) (*stream.ServerStream, error) {
	serverStream := stream.GetServerStream(ctx)

	_, method, err := utils.ParseServicePath(string(request.ServicePath))
	if err != nil {
		return nil, codes.New(codes.ClientMsgErrorCode, "method is invalid")
	}

	serverStream.WithMethod(method)

	return serverStream, nil
}
