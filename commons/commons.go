package commons

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/ubtools/ubt/go/api/proto"
)

func Hex2Int(hexStr string) (uint64, error) {
	// remove 0x suffix if found in the input string
	cleaned := strings.Replace(hexStr, "0x", "", -1)

	// base 16 for hexadecimal
	result, err := strconv.ParseUint(cleaned, 16, 64)
	if err != nil {
		return 0, err
	}
	return uint64(result), nil
}

func Hex2Uint64OrZero(hexStr string) uint64 {
	result, err := Hex2Int(hexStr)
	if err != nil {
		return 0
	}
	return result
}

type UInt64HexString uint64

func (v UInt64HexString) AsNumber() uint64 {
	return uint64(v)
}

func (v UInt64HexString) MarshalJSON() ([]byte, error) {
	return json.Marshal(fmt.Sprintf("0x%x", v))
}

func (v *UInt64HexString) UnmarshalJSON(data []byte) error {
	var hexStr string
	if err := json.Unmarshal(data, &hexStr); err != nil {
		return err
	}

	//log.Printf("UnmarshalJSON: %s", hexStr)

	result, err := Hex2Int(hexStr)
	if err != nil {
		return err
	}
	*v = UInt64HexString(result)
	return nil
}

func ChainIdToString(chainId *proto.ChainId) string {
	return fmt.Sprintf("%s:%s", chainId.Type, chainId.Network)
}

var MAINNET = "MAINNET"

func StringToChainId(chainId string) *proto.ChainId {
	parts := strings.Split(chainId, ":")
	network := MAINNET
	if len(parts) > 1 {
		network = parts[1]
	}
	return &proto.ChainId{
		Type:    parts[0],
		Network: network,
	}
}

type Config struct {
	// The port for the server to listen on
	Port int `json:"port"`

	// The host for the server to listen on
	Host string `json:"host"`
}
