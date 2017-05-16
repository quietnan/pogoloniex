# pogoloniex
A client for the Poloniex API

The API that the cryptocurrency exchange http://poloniex.com provides has a few quirks. The push API for example will provide updates to the orderbook out of order and will send some messages twice or just drop some. Yet, having a reliable local representation is obviously crucial for algorithmic trading decisions.

pogoloniex provides a client that focusses on getting this right.

It is still very much under development and of course I do not give any guarantees or take liability.

Currently some public API calls and the push API are supported, the private API for putting up orders or interacting with your funds on poloniex are commented out as they have not been tested again after some major refactoring.

A minimal use looks like this
```go
	p := New()

	book := p.NewOrderBook("BTC_ETH")
	go func() {
		BC := book.BookChange.Listen()
		Trades := book.Trades.Listen()
		for {
			select {
			case <-BC.C:
				// the book changed
			case t := <-Trades.C:
				// a trade happened
				fmt.Println(t)
			}
		}
	}()
	book.Maintain()
```

You can get have multiple channels listening on an orderbook by calling `BookChange.Listen()` repeadedly.

Have a look at the [marketdepth example](examples/marketdepth).

Feedback and code review welcome.
