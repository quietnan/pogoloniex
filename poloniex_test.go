package pogoloniex

import (
	"fmt"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestGetOrderBook(t *testing.T) {
	p := New()

	book := p.NewOrderBook("XMR_NXT")
	book.Get()
	//	fmt.Println(book)
}

func TestGetPairs(t *testing.T) {
	p := New()
	pairs := p.GetPairs()
	fmt.Println(pairs)
}

func TestMaintainOrderBook(t *testing.T) {
	p := New()

	book := p.NewOrderBook("BTC_ETH")
	go func() {
		BC := book.BookChange.Listen()
		Trades := book.Trades.Listen()
		for {
			select {
			case <-BC.C:
			case <-Trades.C:
			}
		}
	}()
	go book.Maintain()

	if testing.Short() {
		time.Sleep(10 * time.Second)
	} else {
		time.Sleep(1 * time.Minute)
	}
}

func TestOrderBookFunctions(t *testing.T) {
	p := New()

	book := p.NewOrderBook("BTC_ETH")
	book.Get()

	lowestAskRate := book.GetBestRate(AskType)
	fmt.Println(book.GetVolumeBetterThan(AskType, lowestAskRate, lowestAskRate.Mul(decimal.NewFromFloat(1.01)), lowestAskRate.Mul(decimal.NewFromFloat(1.02))))

	highestBidRate := book.GetBestRate(BidType)
	fmt.Println(book.GetVolumeBetterThan(BidType, highestBidRate, highestBidRate.Mul(decimal.NewFromFloat(0.99))))
}

// func TestBuySellCancel(t *testing.T) {
// 	o := NewOrderBook("BTC_XMR")
// 	p.GetOrderBook(o)
// 	iter := o.Trees[BidType].Iterator()
// 	iter.Last()
// 	worstBid := iter.Value().(Offerer)
// 	//worstBid := o.Trees[BidType].At(o.Trees[BidType].Len() - 1).(*Offer)
// 	fmt.Println(worstBid)
// 	fmt.Println("Attempting to add to ", worstBid, "by buying")
// 	testTotal := new(big.Rat)
// 	testTotal.SetString("0.0011")
// 	testAmount := new(big.Rat)
// 	testAmount.Quo(testTotal, worstBid.Rate())
// 	orderNumber, _, _ := p.Buy("BTC_XMR", worstBid.Rate(), testAmount)
// 	if err := p.CancelOrder(orderNumber); err != nil {
// 		log.Fatal(err)
// 	}

// 	iter = o.Trees[AskType].Iterator()
// 	iter.Last()
// 	worstAsk := iter.Value().(Offerer)
// 	//worstAsk := o.Trees[AskType].At(o.Trees[AskType].Len() - 1).(*Offer)
// 	fmt.Println("Attempting to add to ", worstAsk, "by selling")

// 	testAmount.Quo(testTotal, worstAsk.Rate())

// 	orderNumber, _, _ = p.Sell("BTC_XMR", worstAsk.Rate(), testAmount)
// 	if err := p.CancelOrder(orderNumber); err != nil {
// 		log.Fatal(err)
// 	}
// }

// func TestUpdateBalance(t *testing.T) {
// 	fmt.Println(p.GetFee().FloatString(4))
// 	fmt.Println(p.UpdateBalances())
// }
