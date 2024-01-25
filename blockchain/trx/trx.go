package trx

import (
	"crypto/sha256"
	"errors"
	"io"

	b "github.com/ubtr/ubt-go/blockchain"
	"github.com/ubtr/ubt-go/blockchain/eth"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/shengdoushi/base58"
)

const CODE_STR = "TRX"
const CODE_NUM = 195
const DECIMALS = 6
const TronBytePrefix = byte(0x41)

type Address []byte

func (a Address) String() string {
	h256h0 := sha256.New()
	h256h0.Write(a)
	h0 := h256h0.Sum(nil)

	h256h1 := sha256.New()
	h256h1.Write(h0)
	h1 := h256h1.Sum(nil)

	inputCheck := a
	inputCheck = append(inputCheck, h1[:4]...)

	return base58.Encode(inputCheck, base58.BitcoinAlphabet)
}

func RecoverAddress(publicKey []byte, privateKey []byte) (address string, err error) {
	if publicKey == nil {
		if privateKey == nil {
			return "", errors.New("publicKey and privateKey cannot both be nil")
		}
		publicKey, err = eth.PublicKeyFromPrivateKey(privateKey)
		if err != nil {
			return
		}
	}
	return AddressFromPublicKey(publicKey).String(), nil
}

func AddressFromPublicKey(publicKey []byte) Address {
	address := crypto.Keccak256Hash(publicKey[1:]).Bytes()[12:]
	addressTron := make([]byte, 0)
	addressTron = append(addressTron, TronBytePrefix)
	addressTron = append(addressTron, address...)
	return addressTron
}

func TronRandomKey(rnd io.Reader) (*b.KeyPair, error) {
	kp, err := eth.RandomKey(rnd)
	if err != nil {
		return nil, err
	}
	kp.Address = AddressFromPublicKey(kp.PublicKey).String()
	return kp, nil
}

func ValidateAddress(address string) bool {
	binAddr, err := base58.Decode(address, base58.BitcoinAlphabet)
	if err != nil {
		return false
	}
	if len(binAddr) != 25 {
		return false
	}
	if binAddr[0] != TronBytePrefix {
		return false
	}

	// unnecessary check
	h256h0 := sha256.New()
	h256h0.Write(binAddr[:21])
	h0 := h256h0.Sum(nil)

	h256h1 := sha256.New()
	h256h1.Write(h0)
	h1 := h256h1.Sum(nil)

	if h1[0] != binAddr[21] || h1[1] != binAddr[22] || h1[2] != binAddr[23] || h1[3] != binAddr[24] {
		return false
	}

	return true
}

var Instance = b.Blockchain{
	Type:            CODE_STR,
	TypeNum:         CODE_NUM,
	Decimals:        DECIMALS,
	SignatureType:   eth.Instance.SignatureType,
	GenerateAccount: TronRandomKey,
	ValidateAddress: ValidateAddress,
	RecoverAddress:  RecoverAddress,
	Sign:            eth.SignData,
	Verify:          eth.VerifyData,
}

func init() {
	b.Blockchains[CODE_STR] = Instance
}
