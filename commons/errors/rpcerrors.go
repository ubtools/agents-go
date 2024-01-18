package rpcerrors

import "errors"

var ErrInvalidCurrency = errors.New("invalid currency")
var ErrInvalidChainId = errors.New("invalid chain id")
var ErrUnknown = errors.New("unknown error")
