package am

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ubtr/ubt-go/blockchain"
	_ "github.com/ubtr/ubt-go/blockchain/eth"
	ubt_am "github.com/ubtr/ubt/go/api/proto/services/am"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func createTestDb() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	return migrate(db)
}

func TestAMAccountCreateGet(t *testing.T) {
	db := createTestDb()
	srv := internalInitAmServer(db, []byte("test"))

	res, err := srv.CreateAccount(context.TODO(), &ubt_am.CreateAccountRequest{ChainType: "ETH"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Address == "" {
		t.Fatal("Empty address")
	}

	res, err = srv.CreateAccount(context.TODO(), &ubt_am.CreateAccountRequest{ChainType: "ETH", Name: "test1"})
	if err != nil {
		t.Fatal(err)
	}

	if res.Address == "" {
		t.Fatal("Empty address")
	}

	createdAcc, err := srv.GetAccount(context.TODO(), &ubt_am.GetStoredAccountRequest{Address: res.Address})
	if err != nil {
		t.Fatal(err)
	}

	if createdAcc.Name != "test1" {
		t.Fatal("Wrong name")
	}

	if createdAcc.Address != res.Address {
		t.Fatal("Wrong address")
	}

	acccountsList, err := srv.ListAccounts(context.TODO(), &ubt_am.ListAccountsRequest{
		NameFilter: "test",
	})

	if err != nil {
		t.Fatal(err)
	}

	if len(acccountsList.Accounts) != 1 {
		t.Fatalf("Wrong account count: %d", len(acccountsList.Accounts))
	}

	if acccountsList.Accounts[0].Name != "test1" {
		t.Fatal("Wrong name")
	}
}

func TestAMSign(t *testing.T) {
	db := createTestDb()
	srv := internalInitAmServer(db, []byte("test"))

	res, err := srv.CreateAccount(context.TODO(), &ubt_am.CreateAccountRequest{ChainType: "ETH", Name: "test1"})
	if err != nil {
		t.Fatal(err)
	}

	signRes, err := srv.SignPayload(context.TODO(), &ubt_am.SignPayloadRequest{
		ChainType: "ETH",
		Name:      "test1",
		Data:      crypto.Keccak256([]byte("testPayload")),
	})

	if err != nil {
		t.Fatal(err)
	}

	if len(signRes.Signature) == 0 {
		t.Fatal("Empty signature")
	}
	bc := blockchain.GetBlockchain("ETH")
	if !bc.Verify(crypto.Keccak256([]byte("testPayload")), signRes.Signature, res.PublicKey) {
		t.Fatal("Wrong signature")
	}
}
