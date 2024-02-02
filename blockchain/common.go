package blockchain

import "io"

type Signer func(data []byte, privateKey []byte) ([]byte, error)
type Verifier func(data []byte, signature []byte, publicKey []byte) bool
type KeyGenerator func(rand io.Reader) (*KeyPair, error)
type AddressValidator func(addr string) bool
type AddressFromKeys func(publicKey []byte, privateKey []byte) (string, error)
type PublicFromPrivateKey func(privateKey []byte) ([]byte, error)

type KeyPair struct {
	Address    string
	PublicKey  []byte
	PrivateKey []byte
}

func (k KeyPair) String() string {
	return k.Address
}

var Blockchains map[string]Blockchain = make(map[string]Blockchain)

func GetBlockchain(t string) *Blockchain {
	if b, found := Blockchains[t]; !found {
		return nil
	} else {
		return &b
	}
}

type Blockchain struct {
	Type                 string               // slip-044 coin name
	TypeNum              uint                 // slip-044 coin number
	Decimals             uint                 // native currency decimals
	SignatureType        string               // e.g. secp256k1
	Sign                 Signer               // sign any arbitrary data
	Verify               Verifier             // verify any arbitrary data
	GenerateAccount      KeyGenerator         // offline generate a new account/address
	ValidateAddress      AddressValidator     // validate address
	PublicFromPrivateKey PublicFromPrivateKey // get public key from private
	RecoverAddress       AddressFromKeys      // recover address from public and/or private key
}

func (b *Blockchain) String() string {
	return b.Type
}
