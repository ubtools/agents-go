package tests

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ubtr/ubt-go/blockchain"
	_ "github.com/ubtr/ubt-go/blockchain/bnb"
	_ "github.com/ubtr/ubt-go/blockchain/eth"
	_ "github.com/ubtr/ubt-go/blockchain/trx"
)

var staticRandom = &staticRandomReader{}

type staticRandomReader struct{}

func (r *staticRandomReader) Read(p []byte) (n int, err error) {
	for i := range p {
		p[i] = 0x01
	}
	return len(p), nil
}

var testBlockchainAddresses = map[string]string{
	"ETH": "0x1a642f0E3c3aF545E7AcBD38b07251B3990914F1",
	"BNB": "0x1a642f0E3c3aF545E7AcBD38b07251B3990914F1",
	"TRX": "TCNkawTmcQgYSU8nP8cHswT1QPjharxJr7",
}

func runTestAddressFromPublicKey(b blockchain.Blockchain, t *testing.T) {
	k, err := b.GenerateAccount(staticRandom)
	if err != nil {
		t.Errorf("expected no error, got %s", err)
	}

	t.Logf("Address: %s\n", k.Address)
	t.Logf("Public Key: %s\n", fmt.Sprintf("%x", k.PublicKey))
	t.Logf("Private Key: %s\n", fmt.Sprintf("%x", k.PrivateKey))

	if b.ValidateAddress(k.Address) != true {
		t.Errorf("expected address to be valid, got invalid")
	}
	if k.Address != testBlockchainAddresses[b.Type] {
		t.Errorf("expected address of %s, got %s", testBlockchainAddresses[b.Type], k.Address)
	}

}

func TestBlockchainsRegistered(t *testing.T) {
	if len(blockchain.Blockchains) != 3 {
		t.Errorf("expected 3 blockchains, got %d", len(blockchain.Blockchains))
	}
}

func TestKeyGeneration(t *testing.T) {
	for _, b := range blockchain.Blockchains {
		t.Run(b.Type, func(t *testing.T) {
			runTestAddressFromPublicKey(b, t)
		})
	}
}

func runTestSignVerify(b blockchain.Blockchain, t *testing.T) {
	k, err := b.GenerateAccount(staticRandom)
	if err != nil {
		t.Errorf("expected no error, got %s", err)
	}

	data := []byte("hello world")
	dataHash := crypto.Keccak256Hash(data).Bytes()
	sig, err := b.Sign(dataHash, k.PrivateKey)
	if err != nil {
		t.Errorf("expected no error, got %s", err)
	}

	if len(sig) != 65 {
		t.Errorf("expected signature length of 65, got %d", len(sig))
	}

	ok := b.Verify(dataHash, sig, k.PublicKey)
	if !ok {
		t.Errorf("expected signature to be valid, got invalid")
	}
}

func TestSignVerify(t *testing.T) {
	for _, b := range blockchain.Blockchains {
		t.Run(b.Type, func(t *testing.T) {
			runTestSignVerify(b, t)
		})
	}
}
