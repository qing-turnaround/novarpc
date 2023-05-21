package novarpc

import (
	"github.com/xing-you-ji/novarpc/interceptor"
	"time"
)

// Server Options
type ServerOptions struct {
	address           string        // 监听地址
	network           string        // 网络类型
	protocol          string        // 协议类型
	timeout           time.Duration // timeout
	serializationType string        // 请求体序列化协议

	selectorSvrAddr string   // service discovery server address, required when using the third-party service discovery plugin
	tracingSvrAddr  string   // tracing plugin server address, required when using the third-party tracing plugin
	tracingSpanName string   // tracing span name, required when using the third-party tracing plugin
	pluginNames     []string // plugin name
	interceptors    []interceptor.ServerInterceptor
}

// option function
type ServerOption func(*ServerOptions)

// WithAddress set server listening address
func WithAddress(address string) ServerOption {
	return func(o *ServerOptions) {
		o.address = address
	}
}

// WithNetwork set server network type
func WithNetwork(network string) ServerOption {
	return func(o *ServerOptions) {
		o.network = network
	}
}

// WithProtocol set server protocol type
func WithProtocol(protocol string) ServerOption {
	return func(o *ServerOptions) {
		o.protocol = protocol
	}
}

// WithTimeout set server timeout
func WithTimeout(timeout time.Duration) ServerOption {
	return func(o *ServerOptions) {
		o.timeout = timeout
	}
}

// WithSerializationType set server serialization type
func WithSerializationType(serializationType string) ServerOption {
	return func(o *ServerOptions) {
		o.serializationType = serializationType
	}
}

// WithSelectorSvrAddr set service discovery server address
func WithSelectorSvrAddr(addr string) ServerOption {
	return func(o *ServerOptions) {
		o.selectorSvrAddr = addr
	}
}

// WithPlugin set plugin name
func WithPlugin(pluginName ...string) ServerOption {
	return func(o *ServerOptions) {
		o.pluginNames = append(o.pluginNames, pluginName...)
	}
}

// WithInterceptor set interceptor
func WithInterceptor(interceptors ...interceptor.ServerInterceptor) ServerOption {
	return func(o *ServerOptions) {
		o.interceptors = append(o.interceptors, interceptors...)
	}
}

// WithTracingSvrAddr set tracing plugin server address
func WithTracingSvrAddr(addr string) ServerOption {
	return func(o *ServerOptions) {
		o.tracingSvrAddr = addr
	}
}

// WithTracingSpanName set tracing span name
func WithTracingSpanName(name string) ServerOption {
	return func(o *ServerOptions) {
		o.tracingSpanName = name
	}
}
