package main

import (
	"fmt"
	"sync"
	"time"

	pogo "github.com/quietnan/pogoloniex"
	"github.com/shopspring/decimal"
)

// merge is stolen from https://blog.golang.org/pipelines
func merge(cs ...<-chan interface{}) <-chan interface{} {
	var wg sync.WaitGroup
	out := make(chan interface{})

	// Start an output goroutine for each input channel in cs.  output
	// copies values from c to out until c is closed, then calls wg.Done.
	output := func(c <-chan interface{}) {
		for n := range c {
			out <- n
		}
		wg.Done()
	}
	wg.Add(len(cs))
	for _, c := range cs {
		go output(c)
	}

	// Start a goroutine to close out once all the output goroutines are
	// done.  This must start after the wg.Add call.
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func main() {

	fmt.Println("WARNING! This example is to show how to use pogoloniex. It has tons of issues and while a version of this did work successfully in early 2014, if you try to do this now, you WILL lose out.")

	p := pogo.New()
	book1 := p.NewOrderBook("BTC_ETH")
	book2 := p.NewOrderBook("ETH_ETC")
	book3 := p.NewOrderBook("BTC_ETC")
	BC := merge(book1.BookChange.Listen().C, book2.BookChange.Listen().C, book3.BookChange.Listen().C)
	Trade := merge(book1.Trades.Listen().C, book2.Trades.Listen().C, book3.Trades.Listen().C)

	go book1.Maintain()
	go book2.Maintain()
	go book3.Maintain()

	minimaltrade, _ := decimal.NewFromString("0.01")

	time.Sleep(15 * time.Second) // To init all books (requires 7 updates per book before it is considered initialized). Better would be to ask the book if it is current and discard if not.

	for {
		select {
		case <-BC:

			cumulativeAmount := minimaltrade
			iter := book1.Trees[pogo.AskType].Iterator()
			var o1 pogo.Offerer
			for iter.Next() {
				if o1 = iter.Value().(pogo.Offerer); o1.In().Cmp(cumulativeAmount) == 1 {
					break
				}
			}

			cumulativeAmount = o1.Out(cumulativeAmount)
			iter = book2.Trees[pogo.AskType].Iterator()
			var o2 pogo.Offerer
			for iter.Next() {
				if o2 = iter.Value().(pogo.Offerer); o2.In().Cmp(cumulativeAmount) == 1 {
					break
				}
			}

			cumulativeAmount = o2.Out(cumulativeAmount)
			iter = book3.Trees[pogo.BidType].Iterator()
			var o3 pogo.Offerer
			for iter.Next() {
				if o3 = iter.Value().(pogo.Offerer); o3.In().Cmp(cumulativeAmount) == 1 {
					break
				}
			}

			outAmount, _ := o3.Out(cumulativeAmount).Float64()
			fmt.Printf("At %v, when piping 10mBTC through ETH and ETC, you will get %.2fmBTC out.", time.Now().Format("15:04:05"), outAmount*1000)
			if outAmount > 0.01 {
				fmt.Println(" GO FOR IT!")
			} else {
				fmt.Println(" meh...")
			}

		case <-Trade:
		}
	}
}
