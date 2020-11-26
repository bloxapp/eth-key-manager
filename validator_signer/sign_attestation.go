package validator_signer

import (
	"encoding/hex"

	"github.com/pkg/errors"
	pb "github.com/wealdtech/eth2-signer-api/pb/v1"

	"github.com/bloxapp/eth2-key-manager/core"
)

func (signer *SimpleSigner) SignBeaconAttestation(req *pb.SignBeaconAttestationRequest) (*pb.SignResponse, error) {
	// 1. get the account
	if req.GetPublicKey() == nil {
		return nil, errors.New("account was not supplied")
	}
	account, err := signer.wallet.AccountByPublicKey(hex.EncodeToString(req.GetPublicKey()))
	if err != nil {
		return nil, err
	}

	// 2. lock for current account
	signer.lock(account.ID(), "attestation")
	defer func() {
		signer.unlock(account.ID(), "attestation")
	}()

	// 3. far future check
	if !IsValidFarFutureEpoch(signer.network, req.Data.Target.Epoch) {
		return nil, errors.Errorf("target epoch too far into the future")
	}
	if !IsValidFarFutureEpoch(signer.network, req.Data.Source.Epoch) {
		return nil, errors.Errorf("source epoch too far into the future")
	}

	// 4. check we can even sign this
	if val, err := signer.slashingProtector.IsSlashableAttestation(account.ValidatorPublicKey(), req); err != nil || val != nil {
		if err != nil {
			return nil, err
		}
		return nil, errors.Errorf("slashable attestation (%s), not signing", val.Status)
	}

	// 5. add to protection storage
	if err := signer.slashingProtector.UpdateLatestAttestation(account.ValidatorPublicKey(), req); err != nil {
		return nil, err
	}

	// 6. Prepare and sign data
	forSig, err := PrepareAttestationReqForSigning(req)
	if err != nil {
		return nil, err
	}
	sig, err := account.ValidationKeySign(forSig)
	if err != nil {
		return nil, err
	}
	res := &pb.SignResponse{
		State:     pb.ResponseState_SUCCEEDED,
		Signature: sig.Marshal(),
	}

	return res, nil
}

// PrepareAttestationReqForSigning prepares the given attestation request for signing.
// This is exported to allow use it by custom signing mechanism.
func PrepareAttestationReqForSigning(req *pb.SignBeaconAttestationRequest) ([]byte, error) {
	data := core.ToCoreAttestationData(req)
	forSig, err := prepareForSig(data, req.Domain)
	if err != nil {
		return nil, err
	}
	return forSig[:], nil
}
