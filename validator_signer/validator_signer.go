package validator_signer

import (
	"sync"

	"github.com/google/uuid"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	pb "github.com/wealdtech/eth2-signer-api/pb/v1"

	"github.com/bloxapp/eth2-key-manager/core"
)

// ValidatorSigner represents the behavior of the validator signer
type ValidatorSigner interface {
	ListAccounts() (*pb.ListAccountsResponse, error)
	SignBeaconBlock(block *eth.BeaconBlock, domain []byte, pubKey []byte) ([]byte, error)
	SignBeaconAttestation(attestation *eth.AttestationData, domain []byte, pubKey []byte) ([]byte, error)
	SignSlot(slot uint64, domain []byte, pubKey []byte) ([]byte, error)
}

type signingRoot struct {
	Hash   [32]byte `ssz-size:"32"`
	Domain []byte   `ssz-size:"32"`
}

// SimpleSigner implements ValidatorSigner interface
type SimpleSigner struct {
	wallet            core.Wallet
	slashingProtector core.SlashingProtector
	network           core.Network
	signLocks         map[string]*sync.RWMutex
}

// NewSimpleSigner is the constructor of SimpleSigner
func NewSimpleSigner(wallet core.Wallet, slashingProtector core.SlashingProtector, network core.Network) *SimpleSigner {
	return &SimpleSigner{
		wallet:            wallet,
		slashingProtector: slashingProtector,
		network:           network,
		signLocks:         map[string]*sync.RWMutex{},
	}
}

// lock locks signer
func (signer *SimpleSigner) lock(accountId uuid.UUID, operation string) {
	k := accountId.String() + "_" + operation
	if val, ok := signer.signLocks[k]; ok {
		val.Lock()
	} else {
		signer.signLocks[k] = &sync.RWMutex{}
		signer.signLocks[k].Lock()
	}
}

func (signer *SimpleSigner) unlock(accountId uuid.UUID, operation string) {
	k := accountId.String() + "_" + operation
	if val, ok := signer.signLocks[k]; ok {
		val.Unlock()
	}
}

func prepareForSig(data interface{}, domain []byte) ([32]byte, error) {
	root, err := ssz.HashTreeRoot(data)
	if err != nil {
		return [32]byte{}, err
	}

	forSig, err := ssz.HashTreeRoot(&signingRoot{
		Hash:   root,
		Domain: domain,
	})
	if err != nil {
		return [32]byte{}, err
	}

	return forSig, nil
}
