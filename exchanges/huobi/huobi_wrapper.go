package huobi

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-/gocryptotrader/logger"
)

// Start starts the HUOBI go routine
func (h *HUOBI) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		h.Run()
		wg.Done()
	}()
}

// Run implements the HUOBI wrapper
func (h *HUOBI) Run() {
	if h.Verbose {
		log.Debugf("%s Websocket: %s (url: %s).\n", h.GetName(), common.IsEnabled(h.Websocket.IsEnabled()), huobiSocketIOAddress)
		log.Debugf("%s polling delay: %ds.\n", h.GetName(), h.RESTPollingDelay)
		log.Debugf("%s %d currencies enabled: %s.\n", h.GetName(), len(h.EnabledPairs), h.EnabledPairs)
	}

	exchangeProducts, err := h.GetSymbols()
	if err != nil {
		log.Errorf("%s Failed to get available symbols.\n", h.GetName())
	} else {
		forceUpgrade := false
		if common.StringDataContains(h.EnabledPairs, "CNY") || common.StringDataContains(h.AvailablePairs, "CNY") {
			forceUpgrade = true
		}

		if common.StringDataContains(h.BaseCurrencies, "CNY") {
			cfg := config.GetConfig()
			exchCfg, errCNY := cfg.GetExchangeConfig(h.Name)
			if err != nil {
				log.Errorf("%s failed to get exchange config. %s\n", h.Name, errCNY)
				return
			}
			exchCfg.BaseCurrencies = "USD"
			h.BaseCurrencies = []string{"USD"}

			errCNY = cfg.UpdateExchangeConfig(&exchCfg)
			if errCNY != nil {
				log.Errorf("%s failed to update config. %s\n", h.Name, errCNY)
				return
			}
		}

		var currencies []string
		for x := range exchangeProducts {
			newCurrency := exchangeProducts[x].BaseCurrency + "-" + exchangeProducts[x].QuoteCurrency
			currencies = append(currencies, newCurrency)
		}

		if forceUpgrade {
			enabledPairs := []string{"btc-usdt"}
			log.Warn("Available and enabled pairs for Huobi reset due to config upgrade, please enable the ones you would like again")

			err = h.UpdateCurrencies(enabledPairs, true, true)
			if err != nil {
				log.Errorf("%s Failed to update enabled currencies.\n", h.GetName())
			}
		}
		err = h.UpdateCurrencies(currencies, false, forceUpgrade)
		if err != nil {
			log.Errorf("%s Failed to update available currencies.\n", h.GetName())
		}
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (h *HUOBI) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := h.GetMarketDetailMerged(exchange.FormatExchangeCurrency(h.Name, p).String())
	if err != nil {
		return tickerPrice, err
	}

	tickerPrice.Pair = p
	tickerPrice.Low = tick.Low
	tickerPrice.Last = tick.Close
	tickerPrice.Volume = tick.Volume
	tickerPrice.High = tick.High

	if len(tick.Ask) > 0 {
		tickerPrice.Ask = tick.Ask[0]
	}

	if len(tick.Bid) > 0 {
		tickerPrice.Bid = tick.Bid[0]
	}

	ticker.ProcessTicker(h.GetName(), p, tickerPrice, assetType)
	return ticker.GetTicker(h.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (h *HUOBI) FetchTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(h.GetName(), p, assetType)
	if err != nil {
		return h.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (h *HUOBI) FetchOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(h.GetName(), p, assetType)
	if err != nil {
		return h.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (h *HUOBI) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := h.GetDepth(OrderBookDataRequestParams{
		Symbol: exchange.FormatExchangeCurrency(h.Name, p).String(),
		Type:   OrderBookDataRequestParamsTypeStep1,
	})
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		data := orderbookNew.Bids[x]
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: data[1], Price: data[0]})
	}

	for x := range orderbookNew.Asks {
		data := orderbookNew.Asks[x]
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: data[1], Price: data[0]})
	}

	orderbook.ProcessOrderbook(h.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(h.Name, p, assetType)
}

// GetAccountID returns the account ID for trades
func (h *HUOBI) GetAccountID() ([]Account, error) {
	acc, err := h.GetAccounts()
	if err != nil {
		return nil, err
	}

	if len(acc) < 1 {
		return nil, errors.New("no account returned")
	}

	return acc, nil
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// HUOBI exchange - to-do
func (h *HUOBI) GetAccountInfo() (exchange.AccountInfo, error) {
	var info exchange.AccountInfo
	info.Exchange = h.GetName()

	accounts, err := h.GetAccountID()
	if err != nil {
		return info, err
	}

	for _, account := range accounts {
		var acc exchange.Account

		acc.ID = strconv.FormatInt(account.ID, 10)

		balances, err := h.GetAccountBalance(acc.ID)
		if err != nil {
			return info, err
		}

		var currencyDetails []exchange.AccountCurrencyInfo
		for _, balance := range balances {
			var frozen bool
			if balance.Type == "frozen" {
				frozen = true
			}

			var updated bool
			for i := range currencyDetails {
				if currencyDetails[i].CurrencyName == balance.Currency {
					if frozen {
						currencyDetails[i].Hold = balance.Balance
					} else {
						currencyDetails[i].TotalValue = balance.Balance
					}
					updated = true
				}
			}

			if updated {
				continue
			}

			if frozen {
				currencyDetails = append(currencyDetails,
					exchange.AccountCurrencyInfo{
						CurrencyName: balance.Currency,
						Hold:         balance.Balance,
					})
			} else {
				currencyDetails = append(currencyDetails,
					exchange.AccountCurrencyInfo{
						CurrencyName: balance.Currency,
						TotalValue:   balance.Balance,
					})
			}
		}

		acc.Currencies = currencyDetails
		info.Accounts = append(info.Accounts, acc)
	}

	return info, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (h *HUOBI) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (h *HUOBI) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (h *HUOBI) SubmitOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	accountID, err := strconv.ParseInt(clientID, 10, 64)
	if err != nil {
		return submitOrderResponse, err
	}

	var formattedType SpotNewOrderRequestParamsType
	var params = SpotNewOrderRequestParams{
		Amount:    amount,
		Source:    "api",
		Symbol:    common.StringToLower(p.Pair().String()),
		AccountID: int(accountID),
	}

	switch {
	case side == exchange.BuyOrderSide && orderType == exchange.MarketOrderType:
		formattedType = SpotNewOrderRequestTypeBuyMarket
	case side == exchange.SellOrderSide && orderType == exchange.MarketOrderType:
		formattedType = SpotNewOrderRequestTypeSellMarket
	case side == exchange.BuyOrderSide && orderType == exchange.LimitOrderType:
		formattedType = SpotNewOrderRequestTypeBuyLimit
		params.Price = price
	case side == exchange.SellOrderSide && orderType == exchange.LimitOrderType:
		formattedType = SpotNewOrderRequestTypeSellLimit
		params.Price = price
	default:
		return submitOrderResponse, errors.New("unsupported order type")
	}

	params.Type = formattedType
	response, err := h.SpotNewOrder(params)
	if response > 0 {
		submitOrderResponse.OrderID = fmt.Sprintf("%v", response)
	}

	if err == nil {
		submitOrderResponse.IsOrderPlaced = true
	}

	return submitOrderResponse, err
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (h *HUOBI) ModifyOrder(action exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (h *HUOBI) CancelOrder(order exchange.OrderCancellation) error {
	orderIDInt, err := strconv.ParseInt(order.OrderID, 10, 64)

	if err != nil {
		return err
	}

	_, err = h.CancelExistingOrder(orderIDInt)

	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (h *HUOBI) CancelAllOrders(orderCancellation exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	cancelAllOrdersResponse := exchange.CancelAllOrdersResponse{
		OrderStatus: make(map[string]string),
	}
	for _, currency := range h.GetEnabledCurrencies() {
		resp, err := h.CancelOpenOrdersBatch(orderCancellation.AccountID, exchange.FormatExchangeCurrency(h.Name, currency).String())
		if err != nil {
			return cancelAllOrdersResponse, err
		}

		if resp.Data.FailedCount > 0 {
			return cancelAllOrdersResponse, fmt.Errorf("%v orders failed to cancel", resp.Data.FailedCount)
		}

		if resp.Status == "error" {
			return cancelAllOrdersResponse, errors.New(resp.ErrorMessage)
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (h *HUOBI) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (h *HUOBI) GetDepositAddress(cryptocurrency pair.CurrencyItem, accountID string) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (h *HUOBI) WithdrawCryptocurrencyFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	resp, err := h.Withdraw(withdrawRequest.Address, withdrawRequest.Currency.Lower().String(), withdrawRequest.AddressTag, withdrawRequest.Amount, withdrawRequest.FeeAmount)
	return fmt.Sprintf("%v", resp), err
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (h *HUOBI) WithdrawFiatFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (h *HUOBI) WithdrawFiatFundsToInternationalBank(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (h *HUOBI) GetWebsocket() (*exchange.Websocket, error) {
	return h.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (h *HUOBI) GetFeeByType(feeBuilder exchange.FeeBuilder) (float64, error) {
	return h.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (h *HUOBI) GetActiveOrders(getOrdersRequest exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	if len(getOrdersRequest.Currencies) == 0 {
		return nil, errors.New("currency must be supplied")
	}

	side := ""
	if getOrdersRequest.OrderSide == exchange.AnyOrderSide || getOrdersRequest.OrderSide == "" {
		side = ""
	} else if getOrdersRequest.OrderSide == exchange.SellOrderSide {
		side = strings.ToLower(string(getOrdersRequest.OrderSide))
	}

	var allOrders []OrderInfo
	for _, currency := range getOrdersRequest.Currencies {
		resp, err := h.GetOpenOrders(h.ClientID, currency.Pair().Lower().String(), side, 500)
		if err != nil {
			return nil, err
		}
		allOrders = append(allOrders, resp...)
	}

	var orders []exchange.OrderDetail
	for _, order := range allOrders {
		symbol := pair.NewCurrencyPairDelimiter(order.Symbol, h.ConfigCurrencyPairFormat.Delimiter)
		orderDate := time.Unix(order.CreatedAt, 0)

		orders = append(orders, exchange.OrderDetail{
			ID:              fmt.Sprintf("%v", order.ID),
			Exchange:        h.Name,
			Amount:          order.Amount,
			Price:           order.Price,
			OrderDate:       orderDate,
			ExecutedAmount:  order.FilledAmount,
			RemainingAmount: (order.Amount - order.FilledAmount),
			CurrencyPair:    symbol,
		})
	}

	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks, getOrdersRequest.EndTicks)

	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (h *HUOBI) GetOrderHistory(getOrdersRequest exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	if len(getOrdersRequest.Currencies) == 0 {
		return nil, errors.New("currency must be supplied")
	}

	states := "partial-canceled,filled,canceled"
	var allOrders []OrderInfo
	for _, currency := range getOrdersRequest.Currencies {
		resp, err := h.GetOrders(currency.Pair().Lower().String(), "", "", "", states, "", "", "")
		if err != nil {
			return nil, err
		}
		allOrders = append(allOrders, resp...)
	}

	var orders []exchange.OrderDetail
	for _, order := range allOrders {
		symbol := pair.NewCurrencyPairDelimiter(order.Symbol, h.ConfigCurrencyPairFormat.Delimiter)
		orderDate := time.Unix(order.CreatedAt, 0)

		orders = append(orders, exchange.OrderDetail{
			ID:           fmt.Sprintf("%v", order.ID),
			Exchange:     h.Name,
			Amount:       order.Amount,
			Price:        order.Price,
			OrderDate:    orderDate,
			CurrencyPair: symbol,
		})
	}

	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks, getOrdersRequest.EndTicks)

	return orders, nil
}
