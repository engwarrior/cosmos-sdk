package types

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/fee_grant/exported"
)

// BasicFeeAllowance implements FeeAllowance with a one-time grant of tokens
// that optionally expires. The delegatee can use up to SpendLimit to cover fees.
type BasicFeeAllowance struct {
	// SpendLimit is the maximum amount of tokens to be spent
	SpendLimit sdk.Coins

	// Expiration specifies an optional time or height when this allowance expires.
	// If Expiration.IsZero() then it never expires
	Expiration ExpiresAt
}

var _ exported.FeeAllowance = (*BasicFeeAllowance)(nil)

// Accept can use fee payment requested as well as timestamp/height of the current block
// to determine whether or not to process this. This is checked in
// Keeper.UseGrantedFees and the return values should match how it is handled there.
//
// If it returns an error, the fee payment is rejected, otherwise it is accepted.
// The FeeAllowance implementation is expected to update it's internal state
// and will be saved again after an acceptance.
//
// If remove is true (regardless of the error), the FeeAllowance will be deleted from storage
// (eg. when it is used up). (See call to RevokeFeeAllowance in Keeper.UseGrantedFees)
func (a *BasicFeeAllowance) Accept(fee sdk.Coins, blockTime time.Time, blockHeight int64) (bool, error) {
	if a.Expiration.IsExpired(blockTime, blockHeight) {
		return true, sdkerrors.Wrap(ErrFeeLimitExpired, "basic allowance")
	}

	left, invalid := a.SpendLimit.SafeSub(fee)
	if invalid {
		return false, sdkerrors.Wrap(ErrFeeLimitExceeded, "basic allowance")
	}

	a.SpendLimit = left
	return left.IsZero(), nil
}

// PrepareForExport will adjust the expiration based on export time. In particular,
// it will subtract the dumpHeight from any height-based expiration to ensure that
// the elapsed number of blocks this allowance is valid for is fixed.
func (a *BasicFeeAllowance) PrepareForExport(dumpTime time.Time, dumpHeight int64) exported.FeeAllowance {
	return &BasicFeeAllowance{
		SpendLimit: a.SpendLimit,
		Expiration: a.Expiration.PrepareForExport(dumpTime, dumpHeight),
	}
}

// ValidateBasic implements FeeAllowance and enforces basic sanity checks
func (a BasicFeeAllowance) ValidateBasic() error {
	if !a.SpendLimit.IsValid() {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidCoins, "send amount is invalid: %s", a.SpendLimit)
	}
	if !a.SpendLimit.IsAllPositive() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidCoins, "spend limit must be positive")
	}
	return a.Expiration.ValidateBasic()
}