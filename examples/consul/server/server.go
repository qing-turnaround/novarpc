package main

import (
	"time"

	"github.com/xing-you-ji/novarpc"
	"github.com/xing-you-ji/novarpc/plugin/consul"
	"github.com/xing-you-ji/novarpc/testdata"
)

func main() {
	opts := []novarpc.ServerOption{
		novarpc.WithAddress("127.0.0.1:8000"),
		novarpc.WithNetwork("tcp"),
		novarpc.WithSerializationType("msgpack"),
		novarpc.WithTimeout(time.Millisecond * 2000),
		novarpc.WithSelectorSvrAddr("43.139.192.217:8500"),
		novarpc.WithPlugin(consul.Name),
	}
	s := novarpc.NewServer(opts...)
	if err := s.RegisterService("hello.Greeter", new(testdata.Service)); err != nil {
		panic(err)
	}
	s.Serve()

}
