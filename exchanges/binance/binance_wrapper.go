package binance

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
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/request"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-/gocryptotrader/logger"
)

// GetDefaultConfig returns a default exchange config
func (b *Binance) GetDefaultConfig() (*config.ExchangeConfig, error) {
	b.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = b.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = common.JoinStrings(b.BaseCurrencies, ",")

	err := b.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if b.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = b.UpdateTradablePairs(true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets the basic defaults for Binance
func (b *Binance) SetDefaults() {
	b.Name = "Binance"
	b.Enabled = true
	b.Verbose = true
	b.API.CredentialsValidator.RequiresKey = true
	b.API.CredentialsValidator.RequiresSecret = true
	b.APIWithdrawPermissions = exchange.AutoWithdrawCrypto |
		exchange.NoFiatWithdrawals
	b.SetValues()

	b.CurrencyPairs = exchange.CurrencyPairs{
		AssetTypes: assets.AssetTypes{
			assets.AssetTypeSpot,
		},

		UseGlobalPairFormat: true,
		RequestFormat: config.CurrencyPairFormatConfig{
			Uppercase: true,
		},
		ConfigFormat: config.CurrencyPairFormatConfig{
			Delimiter: "-",
			Uppercase: true,
		},
	}

	b.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,

			Trading: exchange.TradingSupported{
				Spot: true,
			},

			RESTCapabilities: exchange.ProtocolFeatures{
				AutoPairUpdates: true,
				TickerBatching:  true,
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	b.Requester = request.New(b.Name,
		request.NewRateLimit(time.Second, binanceAuthRate),
		request.NewRateLimit(time.Second, binanceUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))

	b.API.Endpoints.URLDefault = apiURL
	b.API.Endpoints.URL = b.API.Endpoints.URLDefault
	b.WebsocketInit()
	b.Websocket.Functionality = exchange.WebsocketTickerSupported |
		exchange.WebsocketTradeDataSupported |
		exchange.WebsocketOrderbookSupported
}

// Setup takes in the supplied exchange configuration details and sets params
func (b *Binance) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		b.SetEnabled(false)
		return nil
	}

	err := b.SetupDefaults(exch)
	if err != nil {
		return err
	}

	return b.WebsocketSetup(b.WSConnect,
		exch.Name,
		exch.Features.Enabled.Websocket,
		binanceDefaultWebsocketURL,
		exch.API.Endpoints.WebsocketURL)
}

// Start starts the Binance go routine
func (b *Binance) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		b.Run()
		wg.Done()
	}()
}

// Run implements the Binance wrapper
func (b *Binance) Run() {
	if b.Verbose {
		log.Printf("%s Websocket: %s. (url: %s).\n", b.GetName(), common.IsEnabled(b.Websocket.IsEnabled()), b.Websocket.GetWebsocketURL())
		log.Printf("%s %d currencies enabled: %s.\n", b.GetName(), len(b.CurrencyPairs.Spot.Enabled), b.CurrencyPairs.Spot.Enabled)
	}

	forceUpdate := false
	if !common.StringDataContains(b.CurrencyPairs.Spot.Enabled, "-") || !common.StringDataContains(b.CurrencyPairs.Spot.Available, "-") {
		enabledPairs := []string{"BTC-USDT"}
		log.Warn("WARNING: Available pairs for Binance reset due to config upgrade, please enable the ones you would like again")
		forceUpdate = true

		err := b.UpdatePairs(enabledPairs, assets.AssetTypeSpot, true, true)
		if err != nil {
			log.Errorf("%s failed to update currencies. Err: %s\n", b.Name, err)
		}
	}

	if !b.GetEnabledFeatures().AutoPairUpdates && !forceUpdate {
		return
	}

	err := b.UpdateTradablePairs(forceUpdate)
	if err != nil {
		log.Errorf("%s failed to update tradable pairs. Err: %s", b.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (b *Binance) FetchTradablePairs(asset assets.AssetType) ([]string, error) {
	var validCurrencyPairs []string

	info, err := b.GetExchangeInfo()
	if err != nil {
		return nil, err
	}

	for _, symbol := range info.Symbols {
		if symbol.Status == "TRADING" {
			validCurrencyPairs = append(validCurrencyPairs, symbol.BaseAsset+"-"+symbol.QuoteAsset)
		}
	}
	return validCurrencyPairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (b *Binance) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := b.FetchTradablePairs(assets.AssetTypeSpot)
	if err != nil {
		return err
	}

	return b.UpdatePairs(pairs, assets.AssetTypeSpot, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *Binance) UpdateTicker(p pair.CurrencyPair, assetType assets.AssetType) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := b.GetTickers()
	if err != nil {
		return tickerPrice, err
	}

	for _, x := range b.GetEnabledPairs(assetType) {
		curr := b.FormatExchangeCurrency(x, assetType)
		for y := range tick {
			if tick[y].Symbol != curr.String() {
				continue
			}
			tickerPrice.Pair = x
			tickerPrice.Ask = tick[y].AskPrice
			tickerPrice.Bid = tick[y].BidPrice
			tickerPrice.High = tick[y].HighPrice
			tickerPrice.Last = tick[y].LastPrice
			tickerPrice.Low = tick[y].LowPrice
			tickerPrice.Volume = tick[y].Volume
			ticker.ProcessTicker(b.Name, x, tickerPrice, assetType)
		}
	}
	return ticker.GetTicker(b.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (b *Binance) FetchTicker(p pair.CurrencyPair, assetType assets.AssetType) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (b *Binance) FetchOrderbook(currency pair.CurrencyPair, assetType assets.AssetType) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(b.GetName(), currency, assetType)
	if err != nil {
		return b.UpdateOrderbook(currency, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *Binance) UpdateOrderbook(p pair.CurrencyPair, assetType assets.AssetType) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := b.GetOrderBook(OrderBookDataRequestParams{Symbol: b.FormatExchangeCurrency(p,
		assetType).String(), Limit: 1000})
	if err != nil {
		return orderBook, err
	}

	for _, bids := range orderbookNew.Bids {
		orderBook.Bids = append(orderBook.Bids,
			orderbook.Item{Amount: bids.Quantity, Price: bids.Price})
	}

	for _, asks := range orderbookNew.Asks {
		orderBook.Asks = append(orderBook.Asks,
			orderbook.Item{Amount: asks.Quantity, Price: asks.Price})
	}

	orderbook.ProcessOrderbook(b.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(b.Name, p, assetType)
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// Bithumb exchange
func (b *Binance) GetAccountInfo() (exchange.AccountInfo, error) {
	var info exchange.AccountInfo
	raw, err := b.GetAccount()
	if err != nil {
		return info, err
	}

	var currencyBalance []exchange.AccountCurrencyInfo
	for _, balance := range raw.Balances {
		freeCurrency, err := strconv.ParseFloat(balance.Free, 64)
		if err != nil {
			return info, err
		}

		lockedCurrency, err := strconv.ParseFloat(balance.Locked, 64)
		if err != nil {
			return info, err
		}

		currencyBalance = append(currencyBalance, exchange.AccountCurrencyInfo{
			CurrencyName: balance.Asset,
			TotalValue:   freeCurrency + lockedCurrency,
			Hold:         freeCurrency,
		})
	}

	info.Exchange = b.GetName()
	info.Accounts = append(info.Accounts, exchange.Account{
		Currencies: currencyBalance,
	})

	return info, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (b *Binance) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (b *Binance) GetExchangeHistory(p pair.CurrencyPair, assetType assets.AssetType) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory
	return resp, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (b *Binance) SubmitOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, _ string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse

	var sideType RequestParamsSideType
	if side == exchange.BuyOrderSide {
		sideType = BinanceRequestParamsSideBuy
	} else {
		sideType = BinanceRequestParamsSideSell
	}

	var requestParamsOrderType RequestParamsOrderType
	switch orderType {
	case exchange.MarketOrderType:
		requestParamsOrderType = BinanceRequestParamsOrderMarket
	case exchange.LimitOrderType:
		requestParamsOrderType = BinanceRequestParamsOrderLimit
	default:
		submitOrderResponse.IsOrderPlaced = false
		return submitOrderResponse, errors.New("unsupported order type")
	}

	var orderRequest = NewOrderRequest{
		Symbol:    p.FirstCurrency.String() + p.SecondCurrency.String(),
		Side:      sideType,
		Price:     price,
		Quantity:  amount,
		TradeType: requestParamsOrderType,
	}

	response, err := b.NewOrder(orderRequest)

	if response.OrderID > 0 {
		submitOrderResponse.OrderID = fmt.Sprintf("%v", response.OrderID)
	}

	if err == nil {
		submitOrderResponse.IsOrderPlaced = true
	}

	return submitOrderResponse, err
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *Binance) ModifyOrder(action exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (b *Binance) CancelOrder(order exchange.OrderCancellation) error {
	orderIDInt, err := strconv.ParseInt(order.OrderID, 10, 64)
	if err != nil {
		return err
	}

	_, err = b.CancelExistingOrder(b.FormatExchangeCurrency(order.CurrencyPair,
		order.AssetType).String(), orderIDInt, order.AccountID)

	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (b *Binance) CancelAllOrders(_ exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	cancelAllOrdersResponse := exchange.CancelAllOrdersResponse{
		OrderStatus: make(map[string]string),
	}
	openOrders, err := b.OpenOrders("")
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	for _, order := range openOrders {
		_, err = b.CancelExistingOrder(order.Symbol, order.OrderID, "")
		if err != nil {
			cancelAllOrdersResponse.OrderStatus[strconv.FormatInt(order.OrderID, 10)] = err.Error()
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (b *Binance) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *Binance) GetDepositAddress(cryptocurrency pair.CurrencyItem, _ string) (string, error) {
	return b.GetDepositAddressForCurrency(cryptocurrency.String())
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *Binance) WithdrawCryptocurrencyFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	amountStr := strconv.FormatFloat(withdrawRequest.Amount, 'f', -1, 64)
	id, err := b.WithdrawCrypto(withdrawRequest.Currency.String(), withdrawRequest.Address, withdrawRequest.AddressTag, withdrawRequest.Description, amountStr)

	return strconv.FormatInt(id, 10), err
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (b *Binance) WithdrawFiatFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (b *Binance) WithdrawFiatFundsToInternationalBank(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (b *Binance) GetWebsocket() (*exchange.Websocket, error) {
	return b.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (b *Binance) GetFeeByType(feeBuilder exchange.FeeBuilder) (float64, error) {
	return b.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (b *Binance) GetActiveOrders(getOrdersRequest exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	if len(getOrdersRequest.Currencies) == 0 {
		return nil, errors.New("at least one currency is required to fetch order history")
	}

	var orders []exchange.OrderDetail
	for _, currency := range getOrdersRequest.Currencies {
		resp, err := b.OpenOrders(b.FormatExchangeCurrency(currency, assets.AssetTypeSpot).String())
		if err != nil {
			return nil, err
		}

		for _, order := range resp {
			orderSide := exchange.OrderSide(strings.ToUpper(order.Side))
			orderType := exchange.OrderType(strings.ToUpper(order.Type))
			orderDate := time.Unix(int64(order.Time), 0)

			orders = append(orders, exchange.OrderDetail{
				Amount:       order.OrigQty,
				OrderDate:    orderDate,
				Exchange:     b.Name,
				ID:           fmt.Sprintf("%v", order.OrderID),
				OrderSide:    orderSide,
				OrderType:    orderType,
				Price:        order.Price,
				Status:       order.Status,
				CurrencyPair: pair.NewCurrencyPairFromString(order.Symbol),
			})
		}
	}

	exchange.FilterOrdersByType(&orders, getOrdersRequest.OrderType)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)
	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks, getOrdersRequest.EndTicks)

	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *Binance) GetOrderHistory(getOrdersRequest exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	if len(getOrdersRequest.Currencies) == 0 {
		return nil, errors.New("at least one currency is required to fetch order history")
	}

	var orders []exchange.OrderDetail
	for _, currency := range getOrdersRequest.Currencies {
		resp, err := b.AllOrders(b.FormatExchangeCurrency(currency, assets.AssetTypeSpot).String(), "", "1000")
		if err != nil {
			return nil, err
		}

		for _, order := range resp {
			orderSide := exchange.OrderSide(strings.ToUpper(order.Side))
			orderType := exchange.OrderType(strings.ToUpper(order.Type))
			orderDate := time.Unix(int64(order.Time), 0)
			// New orders are covered in GetOpenOrders
			if order.Status == "NEW" {
				continue
			}

			orders = append(orders, exchange.OrderDetail{
				Amount:       order.OrigQty,
				OrderDate:    orderDate,
				Exchange:     b.Name,
				ID:           fmt.Sprintf("%v", order.OrderID),
				OrderSide:    orderSide,
				OrderType:    orderType,
				Price:        order.Price,
				CurrencyPair: pair.NewCurrencyPairFromString(order.Symbol),
				Status:       order.Status,
			})
		}
	}

	exchange.FilterOrdersByType(&orders, getOrdersRequest.OrderType)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)
	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks, getOrdersRequest.EndTicks)

	return orders, nil
}
