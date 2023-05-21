package main

import (
	"context"
	"errors"

	"time"

	"github.com/xing-you-ji/novarpc"
	"github.com/xing-you-ji/novarpc/auth"
	"github.com/xing-you-ji/novarpc/log"
	"github.com/xing-you-ji/novarpc/metadata"
	"github.com/xing-you-ji/novarpc/testdata"
)

func main() {

	af := func(ctx context.Context) (context.Context, error) {
		md := metadata.ServerMetadata(ctx)

		if len(md) == 0 {
			return ctx, errors.New("token nil")
		}
		v := md["authorization"]
		log.Debug("token : ", string(v))
		if string(v) != "Bearer testToken" {
			return ctx, errors.New("token invalid")
		}
		return ctx, nil
	}

	opts := []gorpc.ServerOption{
		gorpc.WithAddress("127.0.0.1:8003"),
		gorpc.WithNetwork("tcp"),
		gorpc.WithSerializationType("msgpack"),
		gorpc.WithTimeout(time.Millisecond * 2000),
		gorpc.WithInterceptor(auth.BuildAuthInterceptor(af)),
	}
	s := gorpc.NewServer(opts...)
	if err := s.RegisterService("/helloworld.Greeter", new(testdata.Service)); err != nil {
		panic(err)
	}
	s.Serve()
}
