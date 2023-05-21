package novarpc

import (
	"context"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/xing-you-ji/novarpc/codec"
	"github.com/xing-you-ji/novarpc/codes"
	"github.com/xing-you-ji/novarpc/interceptor"
	"github.com/xing-you-ji/novarpc/metadata"
	"github.com/xing-you-ji/novarpc/protocol"
	"github.com/xing-you-ji/novarpc/transport"
	"github.com/xing-you-ji/novarpc/utils"
	"go.uber.org/zap"
)

// Service 定义了一个具体 Service 的通用实现接口
type Service interface {
	Register(string, Handler) // 注册方法
	Serve(*ServerOptions)     // 启动服务
	Close()                   // 关闭服务
	Name() string             // 获取服务名称
}

// service 是 Service 接口的具体实现
type service struct {
	svr         interface{}        // server
	ctx         context.Context    // 上下文
	cancel      context.CancelFunc // 上下文控制器（取消函数）
	serviceName string             // 服务名称
	handlers    map[string]Handler // 方法对应处理函数
	opts        *ServerOptions     // 参数选项

	closing bool // 服务是否正在关闭
}

// ServiceDesc is a detailed description of a service
type ServiceDesc struct {
	Svr         interface{}   // server
	ServiceName string        // 服务名称
	Methods     []*MethodDesc // 方法描述
	HandlerType interface{}
}

// Method 具体方法的描述(包含方法名和方法处理函数)
type MethodDesc struct {
	MethodName string
	Handler    Handler
}

// Handler is the handler of a method
type Handler func(context.Context, interface{}, func(interface{}) error, []interceptor.ServerInterceptor) (interface{}, error)

// Register 注册方法
func (s *service) Register(handlerName string, handler Handler) {
	if s.handlers == nil {
		s.handlers = make(map[string]Handler)
	}
	s.handlers[handlerName] = handler
}

// Serve 启动服务
func (s *service) Serve(opts *ServerOptions) {

	s.opts = opts

	transportOpts := []transport.ServerTransportOption{
		transport.WithServerAddress(s.opts.address),
		transport.WithServerNetwork(s.opts.network),
		transport.WithHandler(s),
		transport.WithServerTimeout(s.opts.timeout),
		transport.WithSerializationType(s.opts.serializationType),
		transport.WithProtocol(s.opts.protocol),
	}

	serverTransport := transport.GetServerTransport(s.opts.protocol)

	s.ctx, s.cancel = context.WithCancel(context.Background())

	if err := serverTransport.ListenAndServe(s.ctx, transportOpts...); err != nil {
		zap.L().Error("server transport listen and serve error", zap.Error(err))
		return
	}
	zap.L().Info("server transport listen and serve success", zap.String("address", s.opts.address))
	<-s.ctx.Done()
}

func (s *service) Close() {
	s.closing = true
	if s.cancel != nil {
		s.cancel()
	}
	fmt.Println("service closed")
}

func (s *service) Name() string {
	return s.serviceName
}

func (s *service) Handle(ctx context.Context, reqbuf []byte) ([]byte, error) {
	// parse protocol header
	request := &protocol.Request{}
	if err := proto.Unmarshal(reqbuf, request); err != nil {
		return nil, err
	}
	// 创建一个新的上下文（里面包含request的 元数据）
	ctx = metadata.WithServerMetadata(ctx, request.Metadata)
	// 请求体反序列化
	serverSerialization := codec.GetSerialization(s.opts.serializationType)
	dec := func(req interface{}) error {
		// 反序列化请求体（请求体默认使用 msgpack进行 序列化 与 反序列化）
		if err := serverSerialization.Unmarshal(request.Payload, req); err != nil {
			return err
		}
		return nil
	}

	// 如果设置了超时时间，则使用超时上下文
	if s.opts.timeout != 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.opts.timeout)
		defer cancel()
	}

	// 解析服务路径
	_, method, err := utils.ParseServicePath(request.ServicePath)
	if err != nil {
		return nil, codes.New(codes.ClientMsgErrorCode, "method is invalid")
	}

	// 如果方法不存在，则返回错误
	handler := s.handlers[method]
	if handler == nil {
		return nil, errors.New("handlers is nil")
	}

	// 处理
	rsp, err := handler(ctx, s.svr, dec, s.opts.interceptors)
	if err != nil {
		return nil, err
	}

	responseBuf, err := serverSerialization.Marshal(rsp)
	if err != nil {
		return nil, err
	}

	return responseBuf, nil
}
