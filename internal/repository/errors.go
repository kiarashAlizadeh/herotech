package repository

import "errors"

var (
	ErrAuctionNotFound      = errors.New("auction not found")
	ErrMaxActiveAuctions    = errors.New("guild has reached maximum concurrent active auctions limit")
	ErrAuctionAlreadyExists = errors.New("an active auction already exists for this item")
	ErrAuctionNotActive     = errors.New("auction is no longer accepting bids")
	ErrBidOnOwnAuction      = errors.New("cannot bid on your own auction item")
	ErrAlreadyHighestBidder = errors.New("you are already holding the leading position in this auction")
	ErrInsufficientBalance  = errors.New("insufficient available gold balance to back reservation")
	ErrRetractLeadingBid    = errors.New("cannot retract a bid while holding the leading position")
	ErrActiveBidNotFound    = errors.New("no active bid reservation found to cancel")
	ErrGuildNotFound        = errors.New("guild not found")
	ErrItemNotFound         = errors.New("item not found")
	ErrItemNotAvailable     = errors.New("item is no longer available for direct purchase")
	ErrPurchaseOwnItem      = errors.New("cannot purchase your own listed item")
	ErrInsufficientGold     = errors.New("insufficient available gold balance")
	ErrDailyLimitExceeded   = errors.New("purchase violates daily transaction limit")
)
