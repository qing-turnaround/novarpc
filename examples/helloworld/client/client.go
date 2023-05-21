package main

import (
	"context"
	"fmt"
	"time"

	"github.com/xing-you-ji/novarpc/client"
	"github.com/xing-you-ji/novarpc/testdata"
)

func main() {
	opts := []client.Option{
		client.WithTarget("127.0.0.1:8000"),
		client.WithNetwork("tcp"),
		client.WithTimeout(2000 * time.Millisecond),
		client.WithSerializationType("msgpack"),
	}
	c := client.DefaultClient
	req := &testdata.HelloRequest{
		Msg: "hello",
	}
	rsp := &testdata.HelloReply{}
	err := c.Call(context.Background(), "/user.Greeter/SayHello", req, rsp, opts...)
	fmt.Println(rsp.Msg, err)
}
