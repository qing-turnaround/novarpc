package novarpc

import (
	"context"
	"fmt"
	"github.com/xing-you-ji/novarpc/interceptor"
	"github.com/xing-you-ji/novarpc/log"
	"github.com/xing-you-ji/novarpc/plugin"
	"github.com/xing-you-ji/novarpc/plugin/jaeger"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"reflect"
	"syscall"
)

// Server
type Server struct {
	opts    *ServerOptions  // 服务参数选项
	service Service         // 一个 Server 可以有一个或多个 Service（暂时只弄一个）
	plugins []plugin.Plugin // 插件
	closing bool            // 服务是否正在关闭
}

// NewServer creates a Server, Support to pass in ServerOption parameters
func NewServer(opt ...ServerOption) *Server {
	log.Init()
	s := &Server{
		opts: &ServerOptions{},
	}

	// 遍历 ServerOption 参数
	for _, o := range opt {
		o(s.opts)
	}

	// 根据参数 创建一个 Service
	s.service = NewService(s.opts)

	for pluginName, pluginVal := range plugin.PluginMap {
		if !containPlugin(pluginName, s.opts.pluginNames) {
			continue
		}
		s.plugins = append(s.plugins, pluginVal)
	}

	return s
}

func NewService(opts *ServerOptions) Service {
	return &service{
		opts: opts,
	}
}

func containPlugin(pluginName string, plugins []string) bool {
	for _, pluginVal := range plugins {
		if pluginName == pluginVal {
			return true
		}
	}
	return false
}

type emptyInterface interface{}

// RegisterService 服务注册
func (s *Server) RegisterService(serviceName string, svr interface{}) error {
	// 通过反射获取 svr 的类型和值
	svrType := reflect.TypeOf(svr)
	svrValue := reflect.ValueOf(svr)

	// 将 svr 封装成 ServiceDesc
	sd := &ServiceDesc{
		ServiceName: serviceName,
		// 这种写法是为了兼容代码生成
		HandlerType: (*emptyInterface)(nil),
		Svr:         svr,
	}

	// 通过反射获取 svr 的所有方法
	methods, err := getServiceMethods(svrType, svrValue)
	if err != nil {
		return err
	}

	// 记录方法
	sd.Methods = methods

	// 注册
	s.Register(sd, svr)

	return nil
}

func getServiceMethods(serviceType reflect.Type, serviceValue reflect.Value) ([]*MethodDesc, error) {

	var methods []*MethodDesc

	// 检查类型 的方法个数
	for i := 0; i < serviceType.NumMethod(); i++ {
		method := serviceType.Method(i)

		// 检查方法的参数个数
		if err := checkMethod(method.Type); err != nil {
			return nil, err
		}

		methodHandler := func(ctx context.Context, svr interface{}, dec func(interface{}) error, ceps []interceptor.ServerInterceptor) (interface{}, error) {

			reqType := method.Type.In(2)

			// 创建一个 reqType 类型的指针
			req := reflect.New(reqType.Elem()).Interface()

			if err := dec(req); err != nil {
				return nil, err
			}

			if len(ceps) == 0 {
				values := method.Func.Call([]reflect.Value{serviceValue, reflect.ValueOf(ctx), reflect.ValueOf(req)})
				// determine error
				return values[0].Interface(), nil
			}

			handler := func(ctx context.Context, reqbody interface{}) (interface{}, error) {

				values := method.Func.Call([]reflect.Value{serviceValue, reflect.ValueOf(ctx), reflect.ValueOf(req)})

				return values[0].Interface(), nil
			}

			return interceptor.ServerIntercept(ctx, req, ceps, handler)
		}

		methods = append(methods, &MethodDesc{
			MethodName: method.Name,
			Handler:    methodHandler,
		})
	}

	return methods, nil
}

func checkMethod(method reflect.Type) error {

	// 入参个数必须是两个
	// TODO 看实际情况再修改
	if method.NumIn() != 3 {
		return fmt.Errorf("method %s invalid, the number of parameters != 2", method.Name())
	}

	// 返回值个数必须是两个
	if method.NumOut() != 2 {
		return fmt.Errorf("method %s invalid, the number of return values != 2", method.Name())
	}

	// 第一个参数必须是 context
	ctxType := method.In(1)
	var contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
	if !ctxType.Implements(contextType) {
		return fmt.Errorf("method %s invalid, first param is not context", method.Name())
	}

	// 第二个参数必须是一个指针
	argType := method.In(2)
	if argType.Kind() != reflect.Ptr {
		return fmt.Errorf("method %s invalid, req type is not a pointer", method.Name())
	}

	// 第一个返回值必须是一个指针
	replyType := method.Out(0)
	if replyType.Kind() != reflect.Ptr {
		return fmt.Errorf("method %s invalid, reply type is not a pointer", method.Name())
	}

	// 第二个返回值必须是 error
	errType := method.Out(1)
	var errorType = reflect.TypeOf((*error)(nil)).Elem()
	if !errType.Implements(errorType) {
		return fmt.Errorf("method %s invalid, returns %s , not error", method.Name(), errType.Name())
	}

	return nil
}

func (s *Server) Register(sd *ServiceDesc, svr interface{}) {
	// 不可能为空
	if sd == nil || svr == nil {
		return
	}
	ht := reflect.TypeOf(sd.HandlerType).Elem()
	st := reflect.TypeOf(svr)
	// 判断 svr 是否实现了 HandlerType 接口类型
	if !st.Implements(ht) {
		zap.L().Error("server.Register found the handlerType is not implemented by the svr", zap.String("handlerType", ht.Name()), zap.String("svr", st.Name()))
	}

	service := &service{
		svr:         svr,
		serviceName: sd.ServiceName,
		handlers:    make(map[string]Handler),
	}

	// 记录方法
	for _, method := range sd.Methods {
		service.handlers[method.MethodName] = method.Handler
	}

	s.service = service
}

// Serve 启动服务
func (s *Server) Serve() {
	// tips：可以加上优雅关闭
	err := s.InitPlugins()
	if err != nil {
		panic(err)
	}

	// 启动服务
	go s.service.Serve(s.opts)
	// 等待关闭信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGSEGV)
	<-quit
	s.Close()
	if err = s.DeRegisterPlugin(); err != nil {
		zap.L().Warn("deregister plugin failed", zap.Error(err))
	} else {
		zap.L().Info("deregister plugin success")
	}

}

type emptyService struct{}

func (s *Server) ServeHttp() {

	if err := s.RegisterService("/http", new(emptyService)); err != nil {
		panic(err)
	}

	s.Serve()
}

func (s *Server) Close() {
	s.closing = false
	s.service.Close()
}

func (s *Server) InitPlugins() error {
	// 初始化插件
	for _, p := range s.plugins {

		switch val := p.(type) {

		case plugin.ResolverPlugin:
			var services []string
			services = append(services, s.service.Name())

			pluginOpts := []plugin.Option{
				plugin.WithSelectorSvrAddr(s.opts.selectorSvrAddr),
				plugin.WithSvrAddr(s.opts.address),
				plugin.WithServices(services),
			}
			if err := val.Register(pluginOpts...); err != nil {
				zap.L().Error("resolver init error", zap.Error(err))
				return err
			}

		case plugin.TracingPlugin:

			pluginOpts := []plugin.Option{
				plugin.WithTracingSvrAddr(s.opts.tracingSvrAddr),
			}

			tracer, err := val.Init(pluginOpts...)
			if err != nil {
				zap.L().Error("tracing init error", zap.Error(err))
				return err
			}

			s.opts.interceptors = append(s.opts.interceptors, jaeger.OpenTracingServerInterceptor(tracer, s.opts.tracingSpanName))

		default:

		}

	}

	return nil
}

func (s *Server) DeRegisterPlugin() error {
	for _, p := range s.plugins {
		switch val := p.(type) {
		case plugin.ResolverPlugin:
			err := val.DeRegister()
			if err != nil {
				return err
			}
		}
	}
	return nil
}
