package wallet_hd

import (
	"encoding/hex"
	"fmt"

	"github.com/bloxapp/KeyVault/core"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// according to https://github.com/ethereum/EIPs/blob/master/EIPS/eip-2334.md
const (
	BaseAccountPath   = "/%d"
	WithdrawalKeyPath = BaseAccountPath + "/0"
	ValidatorKeyPath  = WithdrawalKeyPath + "/0"
)

// Predefined errors
var (
	// ErrAccountNotFound is the error when account not found
	ErrAccountNotFound = errors.New("account not found")
)

// an hierarchical deterministic wallet
type HDWallet struct {
	id          uuid.UUID
	walletType  core.WalletType
	indexMapper map[string]uuid.UUID
	context     *core.WalletContext
}

func NewHDWallet(context *core.WalletContext) *HDWallet {
	return &HDWallet{
		id:          uuid.New(),
		walletType:  core.HDWallet,
		indexMapper: make(map[string]uuid.UUID),
		context:     context,
	}
}

// ID provides the ID for the wallet.
func (wallet *HDWallet) ID() uuid.UUID {
	return wallet.id
}

// Type provides the type of the wallet.
func (wallet *HDWallet) Type() core.WalletType {
	return wallet.walletType
}

// CreateValidatorKey creates a new validation (validator) key pair in the wallet.
// This will error if an account with the name already exists.
func (wallet *HDWallet) CreateValidatorAccount(seed []byte, name string) (core.ValidatorAccount, error) {
	if len(name) == 0 {
		name = fmt.Sprintf("account-%d", len(wallet.indexMapper))
	}

	// create the master key
	key, err := core.MasterKeyFromSeed(seed)
	if err != nil {
		return nil, err
	}

	baseAccountPath := fmt.Sprintf(BaseAccountPath, len(wallet.indexMapper))
	// validator key
	validatorPath := fmt.Sprintf(ValidatorKeyPath, len(wallet.indexMapper))
	validatorKey, err := key.Derive(validatorPath)
	if err != nil {
		return nil, err
	}
	// withdrawal key
	withdrawalPath := fmt.Sprintf(WithdrawalKeyPath, len(wallet.indexMapper))
	withdrawalKey, err := key.Derive(withdrawalPath)
	if err != nil {
		return nil, err
	}

	// create ret account
	ret, err := NewValidatorAccount(
		name,
		validatorKey,
		withdrawalKey.PublicKey(),
		baseAccountPath,
		wallet.context,
	)
	if err != nil {
		return nil, err
	}

	validatorPublicKey := hex.EncodeToString(ret.ValidatorPublicKey().Marshal())
	// register new wallet and save portfolio
	reset := func() {
		delete(wallet.indexMapper, validatorPublicKey)
	}
	wallet.indexMapper[validatorPublicKey] = ret.ID()
	err = wallet.context.Storage.SaveAccount(ret)
	if err != nil {
		reset()
		return nil, err
	}
	err = wallet.context.Storage.SaveWallet(wallet)
	if err != nil {
		reset()
		return nil, err
	}

	return ret, nil
}

func (wallet *HDWallet) DeleteAccountByPublicKey(pubKey string) error {
	account, err := wallet.AccountByPublicKey(pubKey)
	if err != nil {
		return errors.Wrap(err, "failed to get account by public key")
	}

	err = wallet.context.Storage.DeleteAccount(account.ID())
	if err != nil {
		return errors.Wrap(err, "failed to delete account from store")
	}
	delete(wallet.indexMapper, pubKey)
	err = wallet.context.Storage.SaveWallet(wallet)
	if err != nil {
		return errors.Wrap(err, "failed to save wallet")
	}
	return nil
}

// Accounts provides all accounts in the wallet.
func (wallet *HDWallet) Accounts() <-chan core.ValidatorAccount {
	ch := make(chan core.ValidatorAccount, 1024) // TODO - handle more? change from chan?
	go func() {
		for pubKey := range wallet.indexMapper {
			id := wallet.indexMapper[pubKey]
			account, err := wallet.AccountByID(id)
			if err != nil {
				continue
			}
			ch <- account
		}
		close(ch)
	}()

	return ch
}

// AccountByID provides a single account from the wallet given its ID.
// This will error if the account is not found.
func (wallet *HDWallet) AccountByID(id uuid.UUID) (core.ValidatorAccount, error) {
	ret, err := wallet.context.Storage.OpenAccount(id)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return nil, nil
	}
	ret.SetContext(wallet.context)
	return ret, nil
}

func (wallet *HDWallet) SetContext(ctx *core.WalletContext) {
	wallet.context = ctx
}

// AccountByPublicKey provides a single account from the wallet given its public key.
// This will error if the account is not found.
func (wallet *HDWallet) AccountByPublicKey(pubKey string) (core.ValidatorAccount, error) {
	id, exists := wallet.indexMapper[pubKey]
	if !exists {
		return nil, ErrAccountNotFound
	}
	return wallet.AccountByID(id)
}
