package main

import (
	"context"
	"fmt"
	"time"

	"github.com/xing-you-ji/novarpc/client"
	"github.com/xing-you-ji/novarpc/plugin/consul"
	"github.com/xing-you-ji/novarpc/testdata"
)

func main() {
	opts := []client.Option{
		client.WithNetwork("tcp"),
		client.WithTimeout(2000 * time.Millisecond),
		client.WithSelectorName(consul.Name),
	}
	c := client.DefaultClient

	now := time.Now()
	consul.Init("43.139.192.217:8500")
	for {
		req := &testdata.HelloRequest{
			Msg: fmt.Sprintf("hello %v", time.Now().Sub(now).Seconds()),
		}
		rsp := &testdata.HelloReply{}

		err := c.Call(context.Background(), "/helloworld.Greeter/SayHello", req, rsp, opts...)
		if err != nil {
			return
		}
		fmt.Println("receive server response: ", rsp.Msg)
		time.Sleep(3 * time.Second)
	}
}
