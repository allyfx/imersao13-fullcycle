package entity

import (
	"container/heap"
	"sync"
)

type Book struct {
	Orders        []*Order
	Transactions  []*Transaction
	OrdersChan    chan *Order
	OrdersChanOut chan *Order
	Wg            *sync.WaitGroup
}

func NewBook(orderChan chan *Order, orderChanOut chan *Order, wg *sync.WaitGroup) *Book {
	return &Book{
		Orders:        []*Order{},
		Transactions:  []*Transaction{},
		OrdersChan:    orderChan,
		OrdersChanOut: orderChanOut,
		Wg:            wg,
	}
}

func (b *Book) Trade() {
	buyOrders := NewOrderQueue()
	sellOrders := NewOrderQueue()

	heap.Init(buyOrders)
	heap.Init(sellOrders)

	for order := range b.OrdersChan {
		if order.OrderType == "BUY" {
			buyOrders.Push(order)
			if sellOrders.Len() > 0 && sellOrders.Orders[0].Price <= order.Price {
				sellOrder := sellOrders.Pop().(*Order)
				if sellOrder.PendingShares > 0 {
					transaction := NewTransaction(sellOrder, order, order.Shares, sellOrder.Price)
					b.AddTransaction(transaction, b.Wg)

					sellOrder.Transactions = append(sellOrder.Transactions, transaction)
					order.Transactions = append(order.Transactions, transaction)

					b.OrdersChanOut <- sellOrder
					b.OrdersChanOut <- order

					if sellOrder.PendingShares > 0 {
						sellOrders.Push(sellOrder)
					}
				}
			}
		} else if order.OrderType == "SELL" {
			sellOrders.Push(order)
			if buyOrders.Len() > 0 && buyOrders.Orders[0].Price <= order.Price {
				buyOrders := buyOrders.Pop().(*Order)
				if buyOrders.PendingShares > 0 {
					transaction := NewTransaction(buyOrders, order, order.Shares, buyOrders.Price)
					b.AddTransaction(transaction, b.Wg)

					buyOrders.Transactions = append(buyOrders.Transactions, transaction)
					order.Transactions = append(order.Transactions, transaction)

					b.OrdersChanOut <- buyOrders
					b.OrdersChanOut <- order

					if buyOrders.PendingShares > 0 {
						sellOrders.Push(buyOrders)
					}
				}
			}
		}
	}
}

func (b *Book) AddTransaction(transaction *Transaction, wg *sync.WaitGroup) {
	defer wg.Done()

	sellingShares := transaction.SellingOrder.PendingShares
	buyingShares := transaction.BuyingOrder.PendingShares

	minShares := sellingShares
	if buyingShares < minShares {
		minShares = buyingShares
	}

	transaction.SellingOrder.Investor.UpdateAssetPosition(transaction.SellingOrder.Asset.ID, -minShares)
	transaction.AddSellOrderPendingShares(-minShares)

	transaction.BuyingOrder.Investor.UpdateAssetPosition(transaction.BuyingOrder.Asset.ID, minShares)
	transaction.AddBuyOrderPendingShares(-minShares)

	transaction.CalculateTotal(transaction.Shares, transaction.BuyingOrder.Price)
	transaction.CloseOrders()

	b.Transactions = append(b.Transactions, transaction)
}
