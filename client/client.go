package client

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/xing-you-ji/novarpc/codec"
	"github.com/xing-you-ji/novarpc/codes"
	"github.com/xing-you-ji/novarpc/interceptor"
	"github.com/xing-you-ji/novarpc/metadata"
	"github.com/xing-you-ji/novarpc/pool/connpool"
	"github.com/xing-you-ji/novarpc/protocol"
	"github.com/xing-you-ji/novarpc/selector"
	"github.com/xing-you-ji/novarpc/stream"
	"github.com/xing-you-ji/novarpc/transport"
	"github.com/xing-you-ji/novarpc/utils"
)

type Client interface {
	// 调用下游服务
	Invoke(ctx context.Context, req, rsp interface{}, path string, opts ...Option) error
}

// DefaultClient 是一个全局的 Client（为了减少创建/销毁 客户端的损耗）
var DefaultClient = NewDefaultClient()

// New 创建一个默认的 Client
var NewDefaultClient = func() *defaultClient {
	return &defaultClient{
		opts: &Options{
			protocol: "proto",
		},
	}
}

type defaultClient struct {
	opts *Options
}

// Call 通过Invoke 反射捕捉参数调用下游服务
func (c *defaultClient) Call(ctx context.Context, servicePath string, req interface{}, rsp interface{},
	opts ...Option) error {

	callOpts := make([]Option, 0, len(opts)+1)
	callOpts = append(callOpts, opts...)
	callOpts = append(callOpts, WithSerializationType(codec.MsgPack))

	// servicePath example : /user.Greeter/SayHello
	err := c.Invoke(ctx, req, rsp, servicePath, callOpts...)
	if err != nil {
		return err
	}

	return nil
}

func (c *defaultClient) Invoke(ctx context.Context, req, rsp interface{}, path string, opts ...Option) error {

	// 选项模式执行 opts
	for _, o := range opts {
		o(c.opts)
	}

	// 如果设置了超时时间，那么就使用 context.WithTimeout
	if c.opts.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.opts.timeout)
		defer cancel()
	}

	newCtx, clientStream := stream.NewClientStream(ctx)

	// 解析 path 得到 serviceName, method
	serviceName, method, err := utils.ParseServicePath(path)
	if err != nil {
		return err
	}

	// set serviceName, method
	c.opts.serviceName = serviceName
	c.opts.method = method

	// TODO : delete or not
	clientStream.WithServiceName(serviceName)
	clientStream.WithMethod(method)

	// 在执行 Invoke 之前，先执行拦截器对应的函数
	return interceptor.ClientIntercept(newCtx, req, rsp, c.opts.interceptors, c.invoke)
}

// invoke 真正调用下游服务
func (c *defaultClient) invoke(ctx context.Context, req, rsp interface{}) error {

	// 获取序列化器
	serialization := codec.GetSerialization(c.opts.serializationType)
	// 序列化请求参数
	payload, err := serialization.Marshal(req)
	if err != nil {
		return codes.NewFrameworkError(codes.ClientMsgErrorCode, "request marshal failed ...")
	}

	// 按照协议进行编码(默认是自定义协议)
	clientCodec := codec.GetCodec(c.opts.protocol)

	// 增加请求头：例如：metadata
	request := addReqHeader(ctx, c, payload)

	// 请求头进行 序列化
	reqBuf, err := proto.Marshal(request)
	if err != nil {
		return err
	}

	// 编码请求（得到一个完整 的 二进制请求包）
	reqBody, err := clientCodec.Encode(reqBuf)
	if err != nil {
		return err
	}

	clientTransport := c.NewClientTransport()
	clientTransportOpts := []transport.ClientTransportOption{
		transport.WithServiceName(c.opts.serviceName),
		transport.WithClientTarget(c.opts.target),
		transport.WithClientNetwork(c.opts.network),
		transport.WithClientPool(connpool.GetPool("default")),
		transport.WithSelector(selector.GetSelector(c.opts.selectorName)),
		transport.WithTimeout(c.opts.timeout),
	}

	// send request
	frame, err := clientTransport.Send(ctx, reqBody, clientTransportOpts...)
	if err != nil {
		return err
	}

	// 将响应解码
	rspBuf, err := clientCodec.Decode(frame)
	if err != nil {
		return err
	}

	// 反序列化响应头
	response := &protocol.Response{}
	if err = proto.Unmarshal(rspBuf, response); err != nil {
		return err
	}

	if response.RetCode != 0 {
		return codes.New(response.RetCode, response.RetMsg)
	}

	// 反序列化响应
	return serialization.Unmarshal(response.Payload, rsp)

}

func (c *defaultClient) NewClientTransport() transport.ClientTransport {
	return transport.GetClientTransport(c.opts.protocol)
}

func addReqHeader(ctx context.Context, client *defaultClient, payload []byte) *protocol.Request {
	clientStream := stream.GetClientStream(ctx)

	servicePath := fmt.Sprintf("/%s/%s", clientStream.ServiceName, clientStream.Method)
	// 元数据：例如超时时间
	md := metadata.ClientMetadata(ctx)

	// fill the authentication information
	for _, pra := range client.opts.perRPCAuth {
		authMd, _ := pra.GetMetadata(ctx)
		for k, v := range authMd {
			md[k] = []byte(v)
		}
	}

	request := &protocol.Request{
		ServicePath: servicePath,
		Payload:     payload,
		Metadata:    md,
	}

	return request
}
