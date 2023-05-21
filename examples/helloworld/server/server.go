package main

import (
	"github.com/xing-you-ji/novarpc"
	"time"

	"github.com/xing-you-ji/novarpc/testdata"
)

func main() {
	opts := []novarpc.ServerOption{
		novarpc.WithAddress("127.0.0.1:8000"),
		novarpc.WithNetwork("tcp"),
		novarpc.WithSerializationType("msgpack"),
		novarpc.WithTimeout(time.Millisecond * 2000),
	}
	s := novarpc.NewServer(opts...)
	if err := s.RegisterService("/goods.Greeter", new(testdata.Service)); err != nil {
		panic(err)
	}

	// 启动服务
	s.Serve()
}
