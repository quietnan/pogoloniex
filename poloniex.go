package pogoloniex

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
	"gopkg.in/jcelliott/turnpike.v2"
)

type Poloniex struct {
	fee                   decimal.Decimal
	url, publicurl, wsurl string
	key, secret           string
	balances              Balances
	orderPairs            map[string]string
	wsClient              *turnpike.Client
	trading               bool
	httpclient            *http.Client
	httplog               *log.Logger
}

func New(funcs ...func(*Poloniex)) *Poloniex {
	p := &Poloniex{
		fee:        decimal.NewFromFloat(0.0025),
		url:        "https://poloniex.com/tradingApi",
		publicurl:  "http://poloniex.com/public",
		wsurl:      "wss://api.poloniex.com:443",
		trading:    false,
		httpclient: &http.Client{},
		balances:   make(Balances),
		orderPairs: make(map[string]string)}

	for _, f := range funcs {
		f(p)
	}
	var err error
	p.wsClient, err = turnpike.NewWebsocketClient(turnpike.JSON, p.wsurl, nil)
	if err != nil {
		log.Fatalln("creatingerror:", err)
	}
	_, err = p.wsClient.JoinRealm("realm1", nil)
	if err != nil {
		log.Fatalln("joiningerror:", err)
	}
	return p
}

func (p *Poloniex) GetPairs() []string {
	url := p.publicurl + "?command=return24hVolume"
	start := time.Now()
	res, err := http.Get(url)
	if p.httplog != nil {
		p.httplog.Println("Getting pairs (http) took: ", time.Since(start))
	}
	if err != nil {
		log.Fatal("could not get pairs", err)
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal("could not get pairs", err)
	}
	pairs := make(map[string]interface{})
	err = json.Unmarshal(body, &pairs)
	if err != nil {
		fmt.Println("Error unmarshaling. Message was")
		fmt.Println(string(body))
		fmt.Println("requested url was")
		fmt.Println(url)
		log.Fatal("could not get orderbook", err)
	}
	ret := make([]string, len(pairs))
	i := 0
	for k := range pairs {
		ret[i] = k
		i++
	}
	return ret
}

//////////////////// API CALLS ////////////////////

// func (p *Poloniex) UpdateBalances() Balances {
// 	params := map[string]string{"command": "returnBalances"}
// 	tempbalances := make(map[string]string)
// 	p.mutex.Lock()
// 	p.signedNoncedPost(params, &tempbalances)
// 	p.balances = make(Balances)
// 	for k, v := range tempbalances {
// 		f := new(big.Rat)
// 		f.SetString(v)
// 		if f.Cmp(e.Zero()) > 0.0 {
// 			p.balances[k] = f
// 		}
// 	}
// 	p.mutex.Unlock()
// 	return p.balances
// }

// // Buying stuff. Returns order Number that is necessary to e.g. cancel the job. Returns empty string if no order was put for whatever reason.
// func (p *Poloniex) Buy(symbol string, rate decimal.Decimal, amount decimal.Decimal) (_ string, baseDiff decimal.Decimal, assetDiff decimal.Decimal) {
// 	fmt.Printf("P: Buying %v %v for %v each\n", amount.FloatString(8), symbol, rate.FloatString(8))
// 	if !p.trading {
// 		fmt.Println("testmode (not trading)")
// 		return "", nil, nil
// 	}

// 	params := map[string]string{
// 		"command":      "buy",
// 		"currencyPair": symbol,
// 		"rate":         rate.FloatString(8),
// 		"amount":       amount.FloatString(8)}

// 	p.mutex.Lock()
// 	defer p.mutex.Unlock()

// 	answer := struct {
// 		OrderNumber     string
// 		ResultingTrades []struct {
// 			Amount string
// 			Date   string
// 			Rate   string
// 			Total  string
// 		}
// 	}{}
// 	if err := p.signedNoncedPost(params, &answer); err != nil {
// 		fmt.Println("Order not placed!!")
// 		return "", nil, nil
// 	}

// 	p.orderPairs[answer.OrderNumber] = symbol

// 	baseDiff = e.Zero()
// 	assetDiff = e.Zero()
// 	for _, i := range answer.ResultingTrades {
// 		baseDiff.Sub(baseDiff, parseRat(i.Total))
// 		assetDiff.Add(assetDiff, parseRat(i.Amount))
// 	}
// 	oneminusfee := big.NewRat(1, 1)
// 	oneminusfee.Sub(oneminusfee, p.GetFee())
// 	assetDiff.Mul(assetDiff, oneminusfee)
// 	fmt.Printf("%v\n", answer)
// 	return answer.OrderNumber, baseDiff, assetDiff
// }

// // Selling stuff. Returns order Number that is necessary to e.g. cancel the job. Returns empty string if no order was put for whatever reason.
// func (p *Poloniex) Sell(symbol string, rate decimal.Decimal, amount decimal.Decimal) (_ string, baseDiff decimal.Decimal, assetDiff decimal.) {
// 	fmt.Printf("P: Selling %v %v for %v each\n", amount.FloatString(8), symbol, rate.FloatString(8))
// 	if !p.trading {
// 		fmt.Println("testmode (not trading)")
// 		return "", nil, nil
// 	}

// 	params := map[string]string{
// 		"command":      "sell",
// 		"currencyPair": symbol,
// 		"rate":         rate.FloatString(8),
// 		"amount":       amount.FloatString(8)}

// 	p.mutex.Lock()
// 	defer p.mutex.Unlock()

// 	answer := struct {
// 		OrderNumber     string
// 		ResultingTrades []struct {
// 			Amount string
// 			Date   string
// 			Rate   string
// 			Total  string
// 		}
// 	}{}
// 	if err := p.signedNoncedPost(params, &answer); err != nil {
// 		fmt.Println("Order not placed!!")
// 		return "", nil, nil
// 	}

// 	p.orderPairs[answer.OrderNumber] = symbol

// 	baseDiff = e.Zero()
// 	assetDiff = e.Zero()
// 	for _, i := range answer.ResultingTrades {
// 		baseDiff.Add(baseDiff, parseRat(i.Total))
// 		assetDiff.Sub(assetDiff, parseRat(i.Amount))
// 	}
// 	oneminusfee := big.NewRat(1, 1)
// 	oneminusfee.Sub(oneminusfee, p.GetFee())
// 	baseDiff.Mul(baseDiff, oneminusfee)
// 	fmt.Printf("%v\n", answer)
// 	return answer.OrderNumber, baseDiff, assetDiff
// }

// func (p *Poloniex) CancelOrder(orderNumber string) error {
// 	if !p.trading {
// 		fmt.Println("testmode (not trading)")
// 		return nil
// 	}
// 	if orderNumber == "" {
// 		fmt.Println("nothing to cancel")
// 		return nil
// 	}
// 	params := map[string]string{
// 		"command":      "cancelOrder",
// 		"orderNumber":  orderNumber,
// 		"currencyPair": p.orderPairs[orderNumber]}

// 	p.mutex.Lock()
// 	answer := struct{ Success int }{}
// 	p.signedNoncedPost(params, &answer)
// 	p.mutex.Unlock()

// 	if answer.Success == 1 {
// 		fmt.Printf("success cancelling (answer %v)\n", answer)
// 		return nil
// 	} else {
// 		fmt.Println("Could not cancel for some reason\n", answer)
// 		return errors.New("Could not cancel")
// 	}
// }

// func (p *Poloniex) Withdraw(symbol string, amount decimal.Decimal, address string) string {
// 	fmt.Printf("P: Withdrawing %v of %v to %v\n", amount.FloatString(8), symbol, address)
// 	if !p.trading {
// 		fmt.Println("testmode (not trading)")
// 		return ""
// 	}

// 	params := map[string]string{
// 		"command":  "withdraw",
// 		"currency": symbol,
// 		"address":  address,
// 		"amount":   amount.FloatString(8)}

// 	p.mutex.Lock()
// 	defer p.mutex.Unlock()
// 	answer := struct{ Response string }{}
// 	fmt.Println("1")
// 	if err := p.signedNoncedPost(params, &answer); err != nil {
// 		fmt.Println("Order not placed!!")
// 		return ""
// 	}
// 	return answer.Response
// }

//////////////////// HELPERS ////////////////////

func SetTrading(p *Poloniex) {
	p.trading = true
}

func SetFee(fee decimal.Decimal) func(p *Poloniex) {
	return func(p *Poloniex) {
		p.fee = fee
	}
}

func SetLogger(log *log.Logger) func(p *Poloniex) {
	return func(p *Poloniex) {
		p.httplog = log
	}
}

func SetSecrets(key string, secret string) func(p *Poloniex) {
	return func(p *Poloniex) {
		p.key = key
		p.secret = secret
	}
}

func (p *Poloniex) GetFee() decimal.Decimal {
	return p.fee
}

func (p *Poloniex) String() string {
	return "Poloniex"
}
