package main

import (
	pb "BeerShop/proto"
	"context"
	"io"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/grpclog"
)

const (
	Address = "127.0.0.1:8080"
	OpenTLS = true
)

type clientCredential struct{}

func (cc clientCredential) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		"appid":  "101010",
		"appkey": "i am key",
	}, nil
}

func (c clientCredential) RequireTransportSecurity() bool {
	return OpenTLS
}

func main() {
	var err error
	var opts []grpc.DialOption

	if OpenTLS {
		creds, err := credentials.NewClientTLSFromFile("../keys/ca.crt", "server name")
		if err != nil {
			log.Fatalf("failed to create TLS credentials %v", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	opts = append(opts, grpc.WithPerRPCCredentials(new(clientCredential)))
	opts = append(opts, grpc.WithUnaryInterceptor(interceptor))

	conn, err := grpc.NewClient(Address, opts...)
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()
	c := pb.NewBeerShopClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	menuStream, err := c.GetMenu(ctx, &pb.MenuRequest{})
	if err != nil {
		log.Fatal("Failed to get menu:", err)
	}
	done := make(chan bool)
	var items []*pb.Item
	newItem := new(pb.Item)

	go func() {
		for {
			resp, err := menuStream.Recv()
			if err == io.EOF {
				done <- true
				return
			}
			if err != nil {
				log.Fatalf("cannot receive %v", err)
			}
			items = resp.Items
			log.Printf("Resp received %v", resp.Items)
		}
	}()

	<-done

	newItem.Id = 1
	newItem.Cost = "100"
	newItem.Name = "TaiPi"
	item, err := c.Create(ctx, newItem)
	if err != nil {
		log.Fatal(err)
	}
	items = append(items, item)

	receipt, err := c.PlaceOrder(ctx, &pb.Order{Items: items, Name: "cat"})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Received recipt %v", receipt)

	status, err := c.GetOrderStatus(ctx, receipt)
	if err != nil {
		log.Fatal("Failed to get order status:", err)
	}
	log.Printf("Order status:%v", status)
}

func interceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	start := time.Now()
	err := invoker(ctx, method, req, reply, cc, opts...)
	grpclog.Infof("method=%s req=%v rep=%v duration=%s error=%v\n", method, req, reply, time.Since(start), err)
	return err
}
