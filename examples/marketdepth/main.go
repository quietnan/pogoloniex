package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/quietnan/pogoloniex"
	"github.com/shopspring/decimal"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var p = pogoloniex.New()
var upgrader = websocket.Upgrader{}

type point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type depthData struct {
	Ask []point
	Bid []point
}

func depthHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fmt.Println(vars)
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}
	defer c.Close()

	book := p.NewOrderBook(fmt.Sprintf("%v_%v", vars["base"], vars["asset"]))
	BC := book.BookChange.Listen()
	Trade := book.Trades.Listen()

	go book.Maintain()

	for {
		select {
		case <-BC.C:

			highestBidRate := book.GetBestRate(pogoloniex.BidType)
			bidX := []decimal.Decimal{
				highestBidRate.Mul(decimal.NewFromFloat(1.000)),
				highestBidRate.Mul(decimal.NewFromFloat(0.999)),
				highestBidRate.Mul(decimal.NewFromFloat(0.998)),
				highestBidRate.Mul(decimal.NewFromFloat(0.997)),
				highestBidRate.Mul(decimal.NewFromFloat(0.996)),
				highestBidRate.Mul(decimal.NewFromFloat(0.995)),
				highestBidRate.Mul(decimal.NewFromFloat(0.994)),
				highestBidRate.Mul(decimal.NewFromFloat(0.993)),
				highestBidRate.Mul(decimal.NewFromFloat(0.992)),
				highestBidRate.Mul(decimal.NewFromFloat(0.991)),
				highestBidRate.Mul(decimal.NewFromFloat(0.990))}

			bidY := book.GetVolumeBetterThan(pogoloniex.BidType, bidX...)

			bidPoints := make([]point, len(bidX), len(bidX))
			for i, _ := range bidPoints {
				bidPoints[i].X, _ = bidX[i].Float64()
				bidPoints[i].Y, _ = bidY[i].Float64()
			}

			lowestAskRate := book.GetBestRate(pogoloniex.AskType)
			askX := []decimal.Decimal{
				lowestAskRate.Mul(decimal.NewFromFloat(1.000)),
				lowestAskRate.Mul(decimal.NewFromFloat(1.001)),
				lowestAskRate.Mul(decimal.NewFromFloat(1.002)),
				lowestAskRate.Mul(decimal.NewFromFloat(1.003)),
				lowestAskRate.Mul(decimal.NewFromFloat(1.004)),
				lowestAskRate.Mul(decimal.NewFromFloat(1.005)),
				lowestAskRate.Mul(decimal.NewFromFloat(1.006)),
				lowestAskRate.Mul(decimal.NewFromFloat(1.007)),
				lowestAskRate.Mul(decimal.NewFromFloat(1.008)),
				lowestAskRate.Mul(decimal.NewFromFloat(1.009)),
				lowestAskRate.Mul(decimal.NewFromFloat(1.010))}

			askY := book.GetVolumeBetterThan(pogoloniex.AskType, askX...)

			askPoints := make([]point, len(askX), len(askX))
			for i, _ := range askPoints {
				askPoints[i].X, _ = askX[i].Float64()
				askPoints[i].Y, _ = askY[i].Float64()
			}

			err := c.WriteJSON(depthData{askPoints, bidPoints})
			if err != nil {
				fmt.Println("Can not write to websocket")
				return
			}

		case <-Trade.C:
		}
	}
}

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/depth/{base}/{asset}", depthHandler)
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./www/")))

	fmt.Println("Now visit http://localhost:8000")
	err := http.ListenAndServe("localhost:8000", r)
	if err != nil {
		log.Fatal(err)
	}
}
