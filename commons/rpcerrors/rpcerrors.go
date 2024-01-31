package rpcerrors

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var ErrInvalidCurrency = status.Error(codes.InvalidArgument, "invalid currencyId")
var ErrInvalidChainId = status.Error(codes.InvalidArgument, "invalid chainId")
var ErrInvalidAddress = status.Error(codes.InvalidArgument, "invalid address")
var ErrInvalidAmount = status.Error(codes.InvalidArgument, "invalid amount")
var ErrBlockOutOfRange = status.Error(codes.OutOfRange, "no more blocks")
var ErrUnknown = errors.New("unknown error")

func ArgError(argName string, err error) error {
	return status.Errorf(codes.InvalidArgument, "invalid arg '%s': %s", argName, err.Error())
}
