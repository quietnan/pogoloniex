// Code that is too boring to put it anywhere else

package pogoloniex

import (
	"time"

	"github.com/shopspring/decimal"
)

type Balances map[string]decimal.Decimal

type Exchange interface {
	GetFee() decimal.Decimal
	String() string
}

// TODO: handle error gracefully
func parseDecimal(in interface{}) decimal.Decimal {
	switch i := in.(type) {
	case float64:
		return decimal.NewFromFloat(i)
	case string:
		ret, err := decimal.NewFromString(i)
		if err != nil {
			return decimal.Zero
		}
		return ret
	}
	return decimal.Zero
}

type Message struct {
	time    time.Time
	latest  bool
	payload interface{}
}

type TradeMessage struct {
	Pair      string
	OrderType int
	Rate      decimal.Decimal
	Amount    decimal.Decimal
}
