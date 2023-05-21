package testdata

import (
	"context"
	"fmt"
)

type Service struct {
}

type HelloRequest struct {
	Msg string
}

type HelloReply struct {
	Msg string
}

func (s *Service) SayHello(ctx context.Context, req *HelloRequest) (*HelloReply, error) {
	rsp := &HelloReply{
		Msg: "world",
	}
	fmt.Println("receive client msg : ", req.Msg)
	return rsp, nil
}
