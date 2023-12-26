package rpcerrors

import "errors"

var INVALID_CURRENCY = errors.New("invalid currency")
var INVALID_CHAIN_ID = errors.New("invalid chain id")
var UNKNOWN_ERROR = errors.New("unknown error")
