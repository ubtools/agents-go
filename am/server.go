package am

import (
	"context"
	"crypto/rand"
	"errors"

	ubt_am "github.com/ubtools/ubt/go/api/proto/services/am"

	"github.com/ubtools/ubt/go/blockchain"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type Account struct {
	Name        *string `gorm:"index"`
	NetworkType string
	Address     string `gorm:"primaryKey"`
	PK          []byte
}

func InitAMServier(dsn string) *AMServer {
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	db.AutoMigrate(&Account{})

	var srv = AMServer{db: db}
	return &srv
}

type AMServer struct {
	ubt_am.UnimplementedUbtAccountManagerServer
	db *gorm.DB
}

func (s *AMServer) CreateAccount(ctx context.Context, req *ubt_am.CreateAccountRequest) (*ubt_am.CreateAccountResponse, error) {
	bc := blockchain.GetBlockchain(req.NetworkType)
	if bc == nil {
		return nil, errors.New("NO SUCH NETWORK")
	}
	kp, err := bc.KeyGenerator(rand.Reader)
	if err != nil {
		return nil, err
	}
	s.db.Save(&Account{
		Name:        &req.Name,
		NetworkType: req.NetworkType,
		PK:          kp.PrivateKey,
		Address:     kp.Address,
	})
	return &ubt_am.CreateAccountResponse{
		Address: kp.Address,
	}, nil
}

func (s *AMServer) HasAccount(ctx context.Context, req *ubt_am.HasAccountRequest) (*ubt_am.HasAccountResponse, error) {
	var exist bool
	if req.Name != "" {
		res := s.db.Where("name = ?", req.Name).First(&Account{})
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			exist = false
		} else if res.Error != nil {
			return nil, res.Error
		}
	} else {
		res := s.db.First(&Account{}, req.Address)

		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			exist = false
		} else if res.Error != nil {
			return nil, res.Error
		}
	}

	return &ubt_am.HasAccountResponse{
		Exists: exist,
	}, nil
}

func (s *AMServer) ListAccounts(context.Context, *ubt_am.ListAccountsRequest) (*ubt_am.ListAccountsResponse, error) {
	var accounts []Account
	res := s.db.Find(&accounts)
	if res.Error != nil {
		return nil, res.Error
	}

	var r []*ubt_am.ListAccountsResponse_Account
	for _, a := range accounts {
		r = append(r, &ubt_am.ListAccountsResponse_Account{Name: *a.Name, Address: a.Address})
	}

	return &ubt_am.ListAccountsResponse{
		Accounts: r,
	}, nil
}

func (s *AMServer) SignPayload(ctx context.Context, req *ubt_am.SignPayloadRequest) (*ubt_am.SignPayloadResponse, error) {
	bc := blockchain.GetBlockchain(req.NetworkType)
	if bc == nil {
		return nil, errors.New("NO SUCH NETWORK")
	}
	var account Account

	if req.Name != "" {
		res := s.db.Where("name = ?", req.Name).First(&account)
		if res.Error != nil {
			return nil, res.Error
		}
	} else {
		res := s.db.First(&account, req.Address)
		if res.Error != nil {
			return nil, res.Error
		}
	}

	signature, err := bc.Signer(req.Data, account.PK)

	return &ubt_am.SignPayloadResponse{Signature: signature}, err
}
