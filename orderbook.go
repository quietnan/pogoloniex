package pogoloniex

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/SierraSoftworks/multicast"
	tree "github.com/emirpasic/gods/sets/treeset"
	"github.com/emirpasic/gods/utils"
	"github.com/shopspring/decimal"
)

type OrderBook struct {
	Pair            string
	Trees           [2]*tree.Set
	LastTimeUpdated time.Time
	Current         bool
	BookChange      *multicast.Channel
	Trades          *multicast.Channel
	Exchange        Exchange
}

func (p *Poloniex) NewOrderBook(pair string) *OrderBook {
	askTree := tree.NewWith(OfferCompare)
	bidTree := tree.NewWith(OfferCompare)
	return &OrderBook{
		Pair:       pair,
		Trees:      [2]*tree.Set{askTree, bidTree},
		BookChange: multicast.New(),
		Trades:     multicast.New(),
		Exchange:   p,
	}
}

func (o *OrderBook) Get() int {
	// TODO: Polo is currently the only exchange so this is fine. Should extract the relevant parts into an interface though.
	p := o.Exchange.(*Poloniex)

	pair := o.Pair
	url := p.publicurl + "?command=returnOrderBook&currencyPair=" + pair + "&depth=10000000"
	start := time.Now()
	res, err := http.Get(url)
	if p.httplog != nil {
		p.httplog.Println("Getting orderbook (http) took: ", time.Since(start))
	}
	if err != nil {
		log.Fatal("could not get orderbook", err)
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal("could not get orderbook", err)
	}
	depth := &struct {
		//Pair string
		Seq  int
		Asks [][2]interface{} `json:"asks"`
		Bids [][2]interface{} `json:"bids"`
	}{}
	err = json.Unmarshal(body, depth)
	if err != nil {
		fmt.Println("Error unmarshaling. Message was")
		fmt.Println(string(body))
		fmt.Println("requested url was")
		fmt.Println(url)
		log.Fatal("could not get orderbook", err)
	}
	o.Trees[AskType] = tree.NewWith(OfferCompare)
	for _, a := range depth.Asks {
		rate := parseDecimal(a[0])
		amount := parseDecimal(a[1])
		o.Trees[AskType].Add(NewOffer(o.Pair, rate, amount, AskType, p))
	}
	o.Trees[BidType] = tree.NewWith(OfferCompare)
	for _, b := range depth.Bids {
		rate := parseDecimal(b[0])
		amount := parseDecimal(b[1])
		o.Trees[BidType].Add(NewOffer(o.Pair, rate, amount, BidType, p))
	}
	return depth.Seq
}

func (o *OrderBook) Maintain() {
	p := o.Exchange.(*Poloniex)

	o.LastTimeUpdated = time.Now()
	in := o.Order()
	for {
		m := <-in
		arg := m.payload
		t := m.time
		v := arg.(map[string]interface{})
		d := v["data"].(map[string]interface{})
		rate, _ := decimal.NewFromString(d["rate"].(string))
		ordertypestring := d["type"].(string)
		ordertype, ok := ordertypemap[ordertypestring]
		if !ok {
			ordertype = InvalidType
		}
		switch v["type"] {
		case "orderBookModify":
			amount := parseDecimal(d["amount"])
			o.Trees[ordertype].Add(NewOffer(o.Pair, rate, amount, ordertype, p))
		case "orderBookRemove":
			toRemove := NewOffer(o.Pair, rate, decimal.Zero, ordertype, p)
			removedOffer := o.Trees[ordertype].Contains(toRemove)
			o.Trees[ordertype].Remove(toRemove)

			if removedOffer == false {
				log.Println("tried to remove nonexistent entry ", rate, " from ", ordertype, "s of ", o.Pair, ".\n Full OrderBook:\n", o)
				log.Fatalf("Problem with %v\n", o.Pair)
			}
		case "newTrade":
			amount := parseDecimal(d["amount"])
			o.Trades.C <- TradeMessage{o.Pair, ordertype, rate, amount}
			continue
		default:
			log.Fatalln("unknown update type", v["type"])
		}
		if m.latest {
			o.BookChange.C <- Message{time: t}
		}
	}
}

func (o *OrderBook) Order() (out chan *Message) {
	p := o.Exchange.(*Poloniex)

	out = make(chan *Message)
	orderedincoming := tree.NewWith(MessageCompare)
	currentSeq := int(0)
	mutex := sync.Mutex{}

	onMessage := func(args []interface{}, kwargs map[string]interface{}) {
		t := time.Now()

		if len(args) == 0 {
			// Ping message to indicate that nothing has happened.
			return
		}
		seq := int(kwargs["seq"].(float64))
		if seq < currentSeq {
			return
		}

		mutex.Lock()
		defer mutex.Unlock()
		o.LastTimeUpdated = t
		toRemove := make([]interface{}, 0, 15)

		orderedincoming.Add(&OrderableMessage{seq, args})
		iter := orderedincoming.Iterator()
		for iter.Next() {
			om := iter.Value().(*OrderableMessage)
			if om.Seq < currentSeq {
				toRemove = append(toRemove, om)
				continue
			}
			if om.Seq == currentSeq {
				for _, m := range om.Args {
					out <- &Message{t, orderedincoming.Size() == 1, m}
				}
				orderedincoming.Remove(om)
				currentSeq = currentSeq + 1
			} else {
				break
			}
		}
		orderedincoming.Remove(toRemove...)
		if orderedincoming.Size() > 15 {
			log.Println("(re)starting ", o.Pair)
			currentSeq = o.Get() + 1
		}
	}

	if err := p.wsClient.Subscribe(o.Pair, onMessage); err != nil {
		log.Fatalf("Error subscribing to channel %v: %v", o.Pair, err)
	}
	fmt.Println("Subscribed to ", o.Pair)
	return
}

func (o OrderBook) String() (ret string) {
	ret = fmt.Sprintf("ORDERBOOK FOR %v: %v at %v\nAsks:\n", o.Exchange, o.Pair, time.Now().Format("2006-01-02 15:04:05"))
	o.Trees[AskType].Each(func(_ int, value interface{}) { ret += fmt.Sprintf("%v\n", value) })
	ret += fmt.Sprintf("\nBids:\n")
	o.Trees[BidType].Each(func(_ int, value interface{}) { ret += fmt.Sprintf("%v\n", value) })
	return
}

//////////////////// CONVENIENCE OPERATIONS ////////////////////

// // TODO: check!
// func (o *OrderBook) GetMarketOrderPrice(orderType int, total ...decimal.Decimal) (ret []Offerer) {
// 	iter := o.Trees[orderType].Iterator()
// 	cumIn := decimal.Zero
// 	ret = make([]Offerer, len(total), len(total))
// 	i := 0
// 	for iter.Next() {
// 		offer := iter.Value().(Offerer)
// 		cumIn.Add(offer.In())
// 		if cumIn.Cmp(total[i]) > 0 {
// 			switch orderType {
// 			case AskType:
// 				ret[i] = NewOffer(o.Pair, offer.Rate(), cumIn.DivRound(offer.Rate(), 8), AskType, o.Exchange)
// 			case BidType:
// 				ret[i] = NewOffer(o.Pair, offer.Rate(), cumIn, BidType, o.Exchange)
// 			}
// 			i++
// 		}
// 	}

// 	return
// }

func (o *OrderBook) GetVolumeBetterThan(orderType int, rate ...decimal.Decimal) (volume []decimal.Decimal) {
	reference := NewOffer(o.Pair, rate[0], decimal.Zero, orderType, nil)
	iter := o.Trees[orderType].Iterator()
	volume = make([]decimal.Decimal, len(rate), len(rate))
	i := 0

	for iter.Next() {
		offer := iter.Value().(Offerer)
		if OfferCompare(offer, reference) >= 0 {
			i = i + 1
			if i == len(rate) {
				return
			}
			volume[i] = volume[i-1]
			reference = NewOffer(o.Pair, rate[i], decimal.Zero, orderType, nil)
		}
		volume[i] = volume[i].Add(offer.Amount())
	}

	fmt.Println("Could only satisfy the first ", i, " requests")
	return
}

func (o *OrderBook) GetBestRate(orderType int) decimal.Decimal {
	iter := o.Trees[orderType].Iterator()
	iter.First()
	return iter.Value().(Offerer).Rate()

}

//////////////////// HELPERS ////////////////////

type OrderableMessage struct {
	Seq  int
	Args []interface{}
}

func MessageCompare(ai, bi interface{}) int {
	return utils.IntComparator(ai.(*OrderableMessage).Seq, bi.(*OrderableMessage).Seq)
}
