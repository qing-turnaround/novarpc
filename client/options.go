package client

import (
	"time"

	"github.com/xing-you-ji/novarpc/auth"
	"github.com/xing-you-ji/novarpc/interceptor"
	"github.com/xing-you-ji/novarpc/transport"
)

// Options defines the client call parameters
type Options struct {
	serviceName       string        // 服务名
	method            string        // 方法名
	target            string        // target
	timeout           time.Duration // timeout
	network           string        // 网络类型
	protocol          string        // RPC协议类型，比如grpc，自定义协议
	serializationType string        // 序列化协议 msgpack、json、proto
	transportOpts     transport.ClientTransportOptions
	interceptors      []interceptor.ClientInterceptor
	selectorName      string            // service discovery name, e.g. : consul、zookeeper、etcd
	perRPCAuth        []auth.PerRPCAuth // authentication information required for each RPC call
	transportAuth     auth.TransportAuth
}

type Option func(*Options)

// WithServiceName set service name
func WithServiceName(serviceName string) Option {
	return func(o *Options) {
		o.serviceName = serviceName
	}
}

// WithMethod set method name
func WithMethod(method string) Option {
	return func(o *Options) {
		o.method = method
	}
}

// WithTarget set target
func WithTarget(target string) Option {
	return func(o *Options) {
		o.target = target
	}
}

// WithTimeout set timeout
func WithTimeout(timeout time.Duration) Option {
	return func(o *Options) {
		o.timeout = timeout
	}
}

// WithNetwork set network
func WithNetwork(network string) Option {
	return func(o *Options) {
		o.network = network
	}
}

// WithProtocol set protocol
func WithProtocol(protocol string) Option {
	return func(o *Options) {
		o.protocol = protocol
	}
}

// WithTransportOpts set transport options
func WithSerializationType(serializationType string) Option {
	return func(o *Options) {
		o.serializationType = serializationType
	}
}

// WithTransportOpts set transport options
func WithSelectorName(selectorName string) Option {
	return func(o *Options) {
		o.selectorName = selectorName
	}
}

// WithTransportOpts set transport options
func WithInterceptor(interceptors ...interceptor.ClientInterceptor) Option {
	return func(o *Options) {
		o.interceptors = append(o.interceptors, interceptors...)
	}
}

// WithTransportOpts set transport options
func WithPerRPCAuth(rpcAuth auth.PerRPCAuth) Option {
	return func(o *Options) {
		o.perRPCAuth = append(o.perRPCAuth, rpcAuth)
	}
}

// WithTransportOpts set transport options
func WithTransportAuth(transportAuth auth.TransportAuth) Option {
	return func(o *Options) {
		o.transportAuth = transportAuth
	}
}
