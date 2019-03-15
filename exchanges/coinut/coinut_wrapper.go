package coinut

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
	"github.com/thrasher-/gocryptotrader/currency/symbol"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/request"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-/gocryptotrader/logger"
)

// GetDefaultConfig returns a default exchange config
func (c *COINUT) GetDefaultConfig() (*config.ExchangeConfig, error) {
	c.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = c.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = common.JoinStrings(c.BaseCurrencies, ",")

	err := c.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if c.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = c.UpdateTradablePairs(true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets current default values
func (c *COINUT) SetDefaults() {
	c.Name = "COINUT"
	c.Enabled = true
	c.Verbose = true
	c.APIWithdrawPermissions = exchange.WithdrawCryptoViaWebsiteOnly |
		exchange.WithdrawFiatViaWebsiteOnly
	c.API.CredentialsValidator.RequiresKey = true
	c.API.CredentialsValidator.RequiresClientID = true
	c.API.CredentialsValidator.RequiresBase64DecodeSecret = true

	c.CurrencyPairs = exchange.CurrencyPairs{
		AssetTypes: assets.AssetTypes{
			assets.AssetTypeSpot,
		},

		UseGlobalPairFormat: true,
		RequestFormat: config.CurrencyPairFormatConfig{
			Uppercase: true,
		},
		ConfigFormat: config.CurrencyPairFormatConfig{
			Uppercase: true,
		},
	}

	c.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,

			Trading: exchange.TradingSupported{
				Spot: true,
			},

			RESTCapabilities: exchange.ProtocolFeatures{
				AutoPairUpdates: true,
				TickerBatching:  false,
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	c.Requester = request.New(c.Name,
		request.NewRateLimit(time.Second, coinutAuthRate),
		request.NewRateLimit(time.Second, coinutUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))

	c.API.Endpoints.URLDefault = coinutAPIURL
	c.API.Endpoints.URL = c.API.Endpoints.URLDefault
	c.WebsocketInit()
	c.Websocket.Functionality = exchange.WebsocketTickerSupported |
		exchange.WebsocketOrderbookSupported |
		exchange.WebsocketTradeDataSupported
}

// Setup sets the current exchange configuration
func (c *COINUT) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		c.SetEnabled(false)
		return nil
	}

	err := c.SetupDefaults(exch)
	if err != nil {
		return err
	}

	return c.WebsocketSetup(c.WsConnect,
		exch.Name,
		exch.Features.Enabled.Websocket,
		coinutWebsocketURL,
		exch.API.Endpoints.WebsocketURL)
}

// Start starts the COINUT go routine
func (c *COINUT) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		c.Run()
		wg.Done()
	}()
}

// Run implements the COINUT wrapper
func (c *COINUT) Run() {
	if c.Verbose {
		log.Debugf("%s Websocket: %s. (url: %s).\n", c.GetName(), common.IsEnabled(c.Websocket.IsEnabled()), coinutWebsocketURL)
		log.Debugf("%s %d currencies enabled: %s.\n", c.GetName(), len(c.CurrencyPairs.Spot.Enabled), c.CurrencyPairs.Spot.Enabled)
	}

	if !c.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := c.UpdateTradablePairs(false)
	if err != nil {
		log.Errorf("%s failed to update tradable pairs. Err: %s", c.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (c *COINUT) FetchTradablePairs(asset assets.AssetType) ([]string, error) {
	i, err := c.GetInstruments()
	if err != nil {
		return nil, err
	}

	var pairs []string
	c.InstrumentMap = make(map[string]int)
	for x, y := range i.Instruments {
		c.InstrumentMap[x] = y[0].InstID
		pairs = append(pairs, x)
	}

	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (c *COINUT) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := c.FetchTradablePairs(assets.AssetTypeSpot)
	if err != nil {
		return err
	}

	return c.UpdatePairs(pairs, assets.AssetTypeSpot, false, forceUpdate)
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// COINUT exchange
func (c *COINUT) GetAccountInfo() (exchange.AccountInfo, error) {
	var info exchange.AccountInfo
	bal, err := c.GetUserBalance()
	if err != nil {
		return info, err
	}

	var balances = []exchange.AccountCurrencyInfo{
		{
			CurrencyName: symbol.BCH,
			TotalValue:   bal.BCH,
		},
		{
			CurrencyName: symbol.BTC,
			TotalValue:   bal.BTC,
		},
		{
			CurrencyName: symbol.BTG,
			TotalValue:   bal.BTG,
		},
		{
			CurrencyName: symbol.CAD,
			TotalValue:   bal.CAD,
		},
		{
			CurrencyName: symbol.ETC,
			TotalValue:   bal.ETC,
		},
		{
			CurrencyName: symbol.ETH,
			TotalValue:   bal.ETH,
		},
		{
			CurrencyName: symbol.LCH,
			TotalValue:   bal.LCH,
		},
		{
			CurrencyName: symbol.LTC,
			TotalValue:   bal.LTC,
		},
		{
			CurrencyName: symbol.MYR,
			TotalValue:   bal.MYR,
		},
		{
			CurrencyName: symbol.SGD,
			TotalValue:   bal.SGD,
		},
		{
			CurrencyName: symbol.USD,
			TotalValue:   bal.USD,
		},
		{
			CurrencyName: symbol.USDT,
			TotalValue:   bal.USDT,
		},
		{
			CurrencyName: symbol.XMR,
			TotalValue:   bal.XMR,
		},
		{
			CurrencyName: symbol.ZEC,
			TotalValue:   bal.ZEC,
		},
	}
	info.Exchange = c.GetName()
	info.Accounts = append(info.Accounts, exchange.Account{
		Currencies: balances,
	})

	return info, nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (c *COINUT) UpdateTicker(p pair.CurrencyPair, assetType assets.AssetType) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := c.GetInstrumentTicker(c.InstrumentMap[p.Pair().String()])
	if err != nil {
		return ticker.Price{}, err
	}

	tickerPrice.Pair = p
	tickerPrice.Volume = tick.Volume
	tickerPrice.Last = tick.Last
	tickerPrice.High = tick.HighestBuy
	tickerPrice.Low = tick.LowestSell
	ticker.ProcessTicker(c.GetName(), p, tickerPrice, assetType)
	return ticker.GetTicker(c.Name, p, assetType)

}

// FetchTicker returns the ticker for a currency pair
func (c *COINUT) FetchTicker(p pair.CurrencyPair, assetType assets.AssetType) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(c.GetName(), p, assetType)
	if err != nil {
		return c.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (c *COINUT) FetchOrderbook(p pair.CurrencyPair, assetType assets.AssetType) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(c.GetName(), p, assetType)
	if err != nil {
		return c.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (c *COINUT) UpdateOrderbook(p pair.CurrencyPair, assetType assets.AssetType) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := c.GetInstrumentOrderbook(c.InstrumentMap[p.Pair().String()], 200)
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Buy {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: orderbookNew.Buy[x].Quantity, Price: orderbookNew.Buy[x].Price})
	}

	for x := range orderbookNew.Sell {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: orderbookNew.Sell[x].Quantity, Price: orderbookNew.Sell[x].Price})
	}

	orderbook.ProcessOrderbook(c.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(c.Name, p, assetType)
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (c *COINUT) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory

	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (c *COINUT) GetExchangeHistory(p pair.CurrencyPair, assetType assets.AssetType) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (c *COINUT) SubmitOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	var err error
	var APIresponse interface{}
	isBuyOrder := side == exchange.BuyOrderSide
	clientIDInt, err := strconv.ParseUint(clientID, 0, 32)
	clientIDUint := uint32(clientIDInt)

	if err != nil {
		return submitOrderResponse, err
	}
	// Need to get the ID of the currency sent
	instruments, err := c.GetInstruments()
	if err != nil {
		return submitOrderResponse, err
	}

	currencyArray := instruments.Instruments[p.Pair().String()]
	currencyID := currencyArray[0].InstID

	switch orderType {
	case exchange.LimitOrderType:
		APIresponse, err = c.NewOrder(currencyID, amount, price, isBuyOrder, clientIDUint)
	case exchange.MarketOrderType:
		APIresponse, err = c.NewOrder(currencyID, amount, 0, isBuyOrder, clientIDUint)
	default:
		return submitOrderResponse, errors.New("unsupported order type")
	}

	switch apiResp := APIresponse.(type) {
	case OrdersBase:
		orderResult := apiResp
		submitOrderResponse.OrderID = fmt.Sprintf("%v", orderResult.OrderID)
	case OrderFilledResponse:
		orderResult := apiResp
		submitOrderResponse.OrderID = fmt.Sprintf("%v", orderResult.Order.OrderID)
	case OrderRejectResponse:
		orderResult := apiResp
		submitOrderResponse.OrderID = fmt.Sprintf("%v", orderResult.OrderID)
		err = fmt.Errorf("orderID: %v was rejected: %v", orderResult.OrderID, orderResult.Reasons)
	}

	if err == nil {
		submitOrderResponse.IsOrderPlaced = true
	}

	return submitOrderResponse, err
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (c *COINUT) ModifyOrder(action exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (c *COINUT) CancelOrder(order exchange.OrderCancellation) error {
	orderIDInt, err := strconv.ParseInt(order.OrderID, 10, 64)

	if err != nil {
		return err
	}

	// Need to get the ID of the currency sent
	instruments, err := c.GetInstruments()

	if err != nil {
		return err
	}

	currencyArray := instruments.Instruments[c.FormatExchangeCurrency(order.CurrencyPair,
		order.AssetType).String()]
	currencyID := currencyArray[0].InstID
	_, err = c.CancelExistingOrder(currencyID, int(orderIDInt))

	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (c *COINUT) CancelAllOrders(_ exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	// TODO, this is a terrible implementation. Requires DB to improve
	// Coinut provides no way of retrieving orders without a currency
	// So we need to retrieve all currencies, then retrieve orders for each currency
	// Then cancel. Advisable to never use this until DB due to performance
	cancelAllOrdersResponse := exchange.CancelAllOrdersResponse{
		OrderStatus: make(map[string]string),
	}
	instruments, err := c.GetInstruments()
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	var allTheOrders []OrderResponse
	for _, allInstrumentData := range instruments.Instruments {
		for _, instrumentData := range allInstrumentData {

			openOrders, err := c.GetOpenOrders(instrumentData.InstID)
			if err != nil {
				return cancelAllOrdersResponse, err
			}

			allTheOrders = append(allTheOrders, openOrders.Orders...)
		}
	}

	var allTheOrdersToCancel []CancelOrders
	for _, orderToCancel := range allTheOrders {
		cancelOrder := CancelOrders{
			InstrumentID: orderToCancel.InstrumentID,
			OrderID:      orderToCancel.OrderID,
		}
		allTheOrdersToCancel = append(allTheOrdersToCancel, cancelOrder)
	}

	if len(allTheOrdersToCancel) > 0 {
		resp, err := c.CancelOrders(allTheOrdersToCancel)
		if err != nil {
			return cancelAllOrdersResponse, err
		}

		for _, order := range resp.Results {
			if order.Status != "OK" {
				cancelAllOrdersResponse.OrderStatus[strconv.FormatInt(order.OrderID, 10)] = order.Status
			}
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (c *COINUT) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (c *COINUT) GetDepositAddress(cryptocurrency pair.CurrencyItem, accountID string) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (c *COINUT) WithdrawCryptocurrencyFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (c *COINUT) WithdrawFiatFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (c *COINUT) WithdrawFiatFundsToInternationalBank(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (c *COINUT) GetWebsocket() (*exchange.Websocket, error) {
	return c.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (c *COINUT) GetFeeByType(feeBuilder exchange.FeeBuilder) (float64, error) {
	return c.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (c *COINUT) GetActiveOrders(getOrdersRequest exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	instruments, err := c.GetInstruments()
	if err != nil {
		return nil, err
	}

	var allTheOrders []OrderResponse
	for instrument, allInstrumentData := range instruments.Instruments {
		for _, instrumentData := range allInstrumentData {
			for _, currency := range getOrdersRequest.Currencies {
				currStr := fmt.Sprintf("%v%v%v", currency.FirstCurrency.String(), c.CurrencyPairs.ConfigFormat.Delimiter, currency.SecondCurrency.String())
				if strings.EqualFold(currStr, instrument) {
					openOrders, err := c.GetOpenOrders(instrumentData.InstID)
					if err != nil {
						return nil, err
					}
					allTheOrders = append(allTheOrders, openOrders.Orders...)

					continue
				}
			}
		}
	}

	var orders []exchange.OrderDetail
	for _, order := range allTheOrders {
		for instrument, allInstrumentData := range instruments.Instruments {
			for _, instrumentData := range allInstrumentData {
				if instrumentData.InstID == int(order.InstrumentID) {
					currPair := pair.NewCurrencyPairDelimiter(instrument, "")
					orderSide := exchange.OrderSide(strings.ToUpper(order.Side))
					orderDate := time.Unix(order.Timestamp, 0)
					orders = append(orders, exchange.OrderDetail{
						ID:           strconv.FormatInt(order.OrderID, 10),
						Amount:       order.Quantity,
						Price:        order.Price,
						Exchange:     c.Name,
						OrderSide:    orderSide,
						OrderDate:    orderDate,
						CurrencyPair: currPair,
					})
				}
			}
		}
	}

	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks, getOrdersRequest.EndTicks)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)

	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (c *COINUT) GetOrderHistory(getOrdersRequest exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	instruments, err := c.GetInstruments()
	if err != nil {
		return nil, err
	}

	var allTheOrders []OrderFilledResponse
	for instrument, allInstrumentData := range instruments.Instruments {
		for _, instrumentData := range allInstrumentData {
			for _, currency := range getOrdersRequest.Currencies {
				currStr := fmt.Sprintf("%v%v%v", currency.FirstCurrency.String(), c.CurrencyPairs.ConfigFormat.Delimiter, currency.SecondCurrency.String())
				if strings.EqualFold(currStr, instrument) {
					orders, err := c.GetTradeHistory(instrumentData.InstID, -1, -1)
					if err != nil {
						return nil, err
					}
					allTheOrders = append(allTheOrders, orders.Trades...)

					continue
				}
			}
		}
	}

	var orders []exchange.OrderDetail
	for _, order := range allTheOrders {
		for instrument, allInstrumentData := range instruments.Instruments {
			for _, instrumentData := range allInstrumentData {
				if instrumentData.InstID == int(order.Order.InstrumentID) {
					currPair := pair.NewCurrencyPairDelimiter(instrument, "")
					orderSide := exchange.OrderSide(strings.ToUpper(order.Order.Side))
					orderDate := time.Unix(order.Order.Timestamp, 0)
					orders = append(orders, exchange.OrderDetail{
						ID:           strconv.FormatInt(order.Order.OrderID, 10),
						Amount:       order.Order.Quantity,
						Price:        order.Order.Price,
						Exchange:     c.Name,
						OrderSide:    orderSide,
						OrderDate:    orderDate,
						CurrencyPair: currPair,
					})
				}
			}
		}
	}

	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks, getOrdersRequest.EndTicks)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)

	return orders, nil
}
