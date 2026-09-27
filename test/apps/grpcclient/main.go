// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package main provides a minimal gRPC client for integration testing.
// This client is designed to be instrumented with the otelc compile-time tool.
package main

import (
	"context"
	"flag"
	"io"
	"log"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"go.opentelemetry.io/otelc/test/shared/grpcpb/pb"
)

var (
	addr        = flag.String("addr", "localhost:50051", "The server address")
	name        = flag.String("name", "world", "The name to greet")
	stream      = flag.Bool("stream", false, "Use streaming RPC")
	count       = flag.Int("count", 1, "Number of requests to make (for streaming)")
	dialContext = flag.Bool("dial-context", false, "Use grpc.DialContext instead of NewClient")
)

func main() {
	flag.Parse()

	creds := grpc.WithTransportCredentials(insecure.NewCredentials())
	var (
		conn *grpc.ClientConn
		err  error
	)
	if *dialContext {
		conn, err = grpc.DialContext(context.Background(), *addr, creds)
	} else {
		conn, err = grpc.NewClient(*addr, creds)
	}
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewGreeterClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if *stream {
		doStreaming(ctx, client)
	} else {
		doUnary(ctx, client)
	}
}

func doUnary(ctx context.Context, client pb.GreeterClient) {
	resp, err := client.SayHello(ctx, &pb.HelloRequest{Name: *name})
	if err != nil {
		log.Fatalf("failed to call SayHello: %v", err)
	}
	slog.Info("greeting", "message", resp.GetMessage())
}

func doStreaming(ctx context.Context, client pb.GreeterClient) {
	stream, err := client.SayHelloStream(ctx)
	if err != nil {
		log.Fatalf("failed to call SayHelloStream: %v", err)
	}

	// Send requests
	for i := 0; i < *count; i++ {
		if err := stream.Send(&pb.HelloRequest{Name: *name}); err != nil {
			log.Fatalf("failed to send: %v", err)
		}
	}
	if err := stream.CloseSend(); err != nil {
		log.Fatalf("failed to close send: %v", err)
	}

	// Receive responses
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("failed to receive: %v", err)
		}
		slog.Info("stream response", "message", resp.GetMessage())
	}
}
