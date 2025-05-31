package main

import (
	pb "BeerShop/proto"
	"context"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const Address = "127.0.0.1:8080"

type beerShopService struct {
	pb.UnimplementedBeerShopServer
}

var BeerShopService beerShopService
var itemsMap map[int32]string
var items []*pb.Item

func (bss *beerShopService) GetMenu(menuRequest *pb.MenuRequest, srv grpc.ServerStreamingServer[pb.Menu]) error {
	for i := range items {
		err := srv.Send(&pb.Menu{
			Items: items[i : i+1],
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (bss *beerShopService) PlaceOrder(ctx context.Context, in *pb.Order) (*pb.Receipt, error) {
	newOrder := &pb.Receipt{}
	if in.Items == nil || in.Name == "" {
		return nil, fmt.Errorf("order input Items wrong or Name wrong")
	}
	newOrder.Id = "023090823" + in.Name //之後再想
	newOrder.Itesm = in.Items
	return newOrder, nil
}

func (bss *beerShopService) GetOrderStatus(ctx context.Context, in *pb.Receipt) (*pb.OrderStatus, error) {
	return &pb.OrderStatus{
		OrderId: in.Id,
		Status:  "IN_PROGRESS",
	}, nil
}

func (bss *beerShopService) Create(ctx context.Context, in *pb.Item) (*pb.Item, error) {
	newBeer := new(pb.Item)
	if in.Id > int32(len(items)) {
		fmt.Printf("Id 非順號 重新編號為:%v", int32(len(items)))
		newBeer.Id = int32(len(items))
	} else if _, ok := itemsMap[in.Id]; ok {
		fmt.Printf("Id 已存在 重新編號為:%v", int32(len(items)))
		newBeer.Id = int32(len(items))
	} else {
		newBeer.Id = in.Id
	}
	newBeer.Name = in.Name
	newBeer.Cost = in.Cost
	// itemsMap[newBeer.Id] = newBeer.Name
	items = append(items, newBeer)
	return newBeer, nil
}

func main() {
	lis, err := net.Listen("tcp", Address)
	if err != nil {
		panic(err)
	}

	var opts []grpc.ServerOption

	creds, err := credentials.NewServerTLSFromFile("../keys/server.crt", "../keys/server.key")
	if err != nil {
		log.Fatalf("Failed to generate credentials %v", err)
	}
	opts = append(opts, grpc.Creds(creds))

	opts = append(opts, grpc.UnaryInterceptor(interceptor))

	s := grpc.NewServer(opts...)
	pb.RegisterBeerShopServer(s, &BeerShopService)

	log.Printf("Listen on %s", Address)
	s.Serve(lis)
}

func auth(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Errorf(codes.Unauthenticated, "無Token認證信息")
	}

	var appid, appkey string
	if val, ok := md["appid"]; ok {
		appid = val[0]
	}
	if val, ok := md["appkey"]; ok {
		appkey = val[0]
	}
	if appid != "101010" || appkey != "i am key" {
		return status.Errorf(codes.Unauthenticated, "Token認證信息無效:appid = %s, appkey= %s", appid, appkey)
	}
	return nil
}

func interceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	err := auth(ctx)
	if err != nil {
		return nil, err
	}
	return handler(ctx, req)
}
