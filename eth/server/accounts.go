package server

import (
	"context"

	"github.com/ubtools/ubt/go/api/proto"
	"github.com/ubtools/ubt/go/api/proto/services"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (srv *EthServer) GetAccount(ctx context.Context, req *services.GetAccountRequest) (*proto.Account, error) {
	return &proto.Account{
		Id:         req.Address,
		IsContract: false, //FIXME
	}, nil
}
func (srv *EthServer) DeriveAccount(ctx context.Context, req *services.DeriveAccountRequest) (*proto.Account, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeriveAccount not implemented")
}
