package am

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"time"

	ubt_am "github.com/ubtr/ubt/go/api/proto/services/am"

	"github.com/ubtr/ubt-go/blockchain"
	"github.com/ubtr/ubt-go/commons/rpcerrors"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Account struct {
	Name        *string `gorm:"uniqueIndex;unique"`
	NetworkType string
	Address     string `gorm:"primaryKey"`
	PK          []byte
	PublicKey   []byte
}

func migrate(db *gorm.DB) *gorm.DB {
	db.AutoMigrate(&Account{})
	return db
}

func GormOpenRetry(dsn string, opts ...gorm.Option) (*gorm.DB, error) {
	retryCount := 10
	var db *gorm.DB
	var err error
	for i := 0; i < retryCount; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			return migrate(db), nil
		}
		slog.Warn(fmt.Sprintf("[%v of %v] Failed to connect to DB, retrying...", i, retryCount))
		time.Sleep(2 * time.Second)
	}
	return nil, err
}

func InitAMServer(dsn string, encKey []byte) *AMServer {
	db, err := GormOpenRetry(dsn, &gorm.Config{})
	if err != nil {
		panic(err)
	}

	return internalInitAmServer(db, encKey)
}

func internalInitAmServer(db *gorm.DB, encKey []byte) *AMServer {
	var srv = AMServer{db: db, encryptionKey: GetEncryption(encKey)}
	return &srv
}

type AMServer struct {
	ubt_am.UnimplementedUbtAccountManagerServer
	db            *gorm.DB
	encryptionKey Encryption // key to encrypt necessary columns
}

func (s *AMServer) CreateAccount(ctx context.Context, req *ubt_am.CreateAccountRequest) (*ubt_am.CreateAccountResponse, error) {
	slog.Debug("CreateAccount", "chainType", req.ChainType, "name", req.Name, "pkLen", len(req.PrivateKey))
	bc := blockchain.GetBlockchain(req.ChainType)
	if bc == nil {
		return nil, rpcerrors.ErrInvalidChainId
	}

	var encryptedKey []byte
	var publicKey []byte
	var err error
	var address string

	if req.PrivateKey != nil {
		encryptedKey, err = s.encryptionKey.Encrypt(req.PrivateKey)
		if err != nil {
			return nil, err
		}
		address, err = bc.RecoverAddress(nil, req.PrivateKey)
		if err != nil {
			return nil, err
		}

		slog.Debug("AddressFromPK", "chainType", req.ChainType, "address", address)
		publicKey, err = bc.PublicFromPrivateKey(req.PrivateKey)
		if err != nil {
			return nil, err
		}
	} else {
		kp, err := bc.GenerateAccount(rand.Reader)
		if err != nil {
			return nil, err
		}

		encryptedKey, err = s.encryptionKey.Encrypt(kp.PrivateKey)
		if err != nil {
			return nil, err
		}
		publicKey = kp.PublicKey
		address = kp.Address
	}

	var name *string = nil
	if req.Name != "" {
		name = &req.Name
	}

	err = s.db.Save(&Account{
		Name:        name,
		NetworkType: req.ChainType,
		PK:          encryptedKey,
		PublicKey:   publicKey,
		Address:     address,
	}).Error
	if err != nil {
		return nil, err
	}
	return &ubt_am.CreateAccountResponse{
		Address:   address,
		Name:      req.Name,
		PublicKey: publicKey,
	}, nil
}

func (s *AMServer) GetAccount(ctx context.Context, req *ubt_am.GetStoredAccountRequest) (*ubt_am.GetStoredAccountResponse, error) {
	slog.Debug("GetAccount", "req", req)
	var acc Account
	if req.Name != "" {
		res := s.db.Where("name = ?", req.Name).First(&acc)
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return &ubt_am.GetStoredAccountResponse{}, nil
		} else if res.Error != nil {
			return nil, res.Error
		}
		name := ""
		if acc.Name != nil {
			name = *acc.Name
		}
		return &ubt_am.GetStoredAccountResponse{Address: acc.Address, Name: name, PublicKey: acc.PublicKey}, nil
	} else {
		res := s.db.First(&acc, "address = ?", req.Address)

		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return &ubt_am.GetStoredAccountResponse{}, nil
		} else if res.Error != nil {
			return nil, res.Error
		}
		name := ""
		if acc.Name != nil {
			name = *acc.Name
		}
		return &ubt_am.GetStoredAccountResponse{Address: acc.Address, Name: name, PublicKey: acc.PublicKey}, nil
	}
}

func (s *AMServer) ListAccounts(ctx context.Context, req *ubt_am.ListAccountsRequest) (*ubt_am.ListAccountsResponse, error) {
	var accounts []Account
	res := s.db.Where("name like ?", req.NameFilter+"%").Find(&accounts)
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
	bc := blockchain.GetBlockchain(req.ChainType)
	if bc == nil {
		return nil, rpcerrors.ErrInvalidChainId
	}
	var account Account

	if req.Name != "" {
		res := s.db.Where("name = ?", req.Name).First(&account)
		if res.Error != nil {
			return nil, res.Error
		}
	} else {
		res := s.db.First(&account, "address = ?", req.Address)
		if res.Error != nil {
			return nil, res.Error
		}
	}

	decryptedPk, err := s.encryptionKey.Decrypt(account.PK)
	if err != nil {
		return nil, err
	}

	signature, err := bc.Sign(req.Data, decryptedPk)

	return &ubt_am.SignPayloadResponse{Signature: signature}, err
}
