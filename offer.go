package pogoloniex

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

const (
	AskType int = iota
	BidType
	BuyType
	SellType
	InvalidType
)

var ordertypemap = map[string]int{
	"bid":  BidType,
	"ask":  AskType,
	"buy":  BuyType,
	"sell": SellType,
}

type Offerer interface {
	Pair() string
	Rate() decimal.Decimal
	Amount() decimal.Decimal
	OfferType() int
	In() decimal.Decimal
	Out(in decimal.Decimal) decimal.Decimal
	InSymbol() string
	OutSymbol() string
}

type Offer struct {
	pair      string
	rate      decimal.Decimal
	amount    decimal.Decimal
	offerType int
	exchange  Exchange
}

func (o Offer) Pair() string {
	return o.pair
}
func (o Offer) Rate() decimal.Decimal {
	return o.rate
}
func (o Offer) Amount() decimal.Decimal {
	return o.amount
}

// ####################
// TODO: Check if Poloniex rounds _exactly_ the same way!

type Ask struct {
	Offer
}

func (a Ask) OfferType() int {
	return AskType
}

func (a Ask) In() decimal.Decimal {
	return a.amount.Mul(a.rate)
}

// fee=f*in/rate
// out=in/rate-fee
func (a Ask) Out(in decimal.Decimal) decimal.Decimal {
	fee := a.exchange.GetFee().Mul(in).DivRound(a.rate, 8)
	return in.DivRound(a.rate, 8).Sub(fee)
}

func (a Ask) InSymbol() string {
	return strings.Split(a.pair, "_")[0]
}

func (a Ask) OutSymbol() string {
	return strings.Split(a.pair, "_")[1]
}

func (a Ask) String() string {
	s := "ask: "
	s += fmt.Sprintf("%12v: %20v", a.rate.StringFixed(8), a.amount.StringFixed(8))
	return s
}

// ####################

type Bid struct {
	Offer
}

func (b Bid) OfferType() int {
	return BidType
}

func (b Bid) In() decimal.Decimal {
	return b.amount
}

func (b Bid) Out(in decimal.Decimal) decimal.Decimal {
	out := in.Mul(b.rate)
	return out.Round(8).Sub(out.Mul(b.exchange.GetFee()).Round(8))
}

func (b Bid) InSymbol() string {
	return strings.Split(b.pair, "_")[1]
}

func (b Bid) OutSymbol() string {
	return strings.Split(b.pair, "_")[0]
}

func (b Bid) String() string {
	s := "bid: "
	s += fmt.Sprintf("%12v: %20v", b.rate.StringFixed(8), b.amount.StringFixed(8))
	return s
}

// ####################

type Invalid struct {
	Offer
}

func (i Invalid) OfferType() int {
	return InvalidType
}

func (i Invalid) In() decimal.Decimal {
	return decimal.Zero
}

func (i Invalid) Out(in decimal.Decimal) decimal.Decimal {
	return decimal.Zero
}

func (i Invalid) InSymbol() string {
	return ""
}

func (i Invalid) OutSymbol() string {
	return ""
}

// ####################

// fulfilling tree's Interface interface
func OfferCompare(ai, bi interface{}) int {
	a := ai.(Offerer)
	b := bi.(Offerer)
	if a.OfferType() != b.OfferType() {
		panic("comparing different offer types")
	}
	order := 1
	if a.OfferType() == BidType {
		order = -1
	}
	return order * a.Rate().Cmp(b.Rate())
}

func NewOffer(pair string, rate decimal.Decimal, amount decimal.Decimal, offerType int, exchange Exchange) Offerer {
	if offerType == AskType {
		return &Ask{Offer{pair: pair, rate: rate, amount: amount, offerType: AskType, exchange: exchange}}
	} else if offerType == BidType {
		return &Bid{Offer{pair: pair, rate: rate, amount: amount, offerType: BidType, exchange: exchange}}
	}
	return &Invalid{Offer{pair: pair, rate: decimal.Zero, amount: decimal.Zero, offerType: InvalidType, exchange: nil}}
}
