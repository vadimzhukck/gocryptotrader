package gemini

import (
	"errors"
	"fmt"
	"net/url"
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
func (g *Gemini) GetDefaultConfig() (*config.ExchangeConfig, error) {
	g.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = g.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = common.JoinStrings(g.BaseCurrencies, ",")

	err := g.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if g.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = g.UpdateTradablePairs(true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets package defaults for gemini exchange
func (g *Gemini) SetDefaults() {
	g.Name = "Gemini"
	g.Enabled = true
	g.Verbose = true
	g.APIWithdrawPermissions = exchange.AutoWithdrawCryptoWithAPIPermission |
		exchange.AutoWithdrawCryptoWithSetup |
		exchange.WithdrawFiatViaWebsiteOnly
	g.API.CredentialsValidator.RequiresKey = true
	g.API.CredentialsValidator.RequiresSecret = true

	g.CurrencyPairs = exchange.CurrencyPairs{
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

	g.Features = exchange.Features{
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

	g.Requester = request.New(g.Name,
		request.NewRateLimit(time.Minute, geminiAuthRate),
		request.NewRateLimit(time.Minute, geminiUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))

	g.API.Endpoints.URLDefault = geminiAPIURL
	g.API.Endpoints.URL = g.API.Endpoints.URLDefault
	g.WebsocketInit()
	g.Websocket.Functionality = exchange.WebsocketOrderbookSupported |
		exchange.WebsocketTradeDataSupported
}

// Setup sets exchange configuration parameters
func (g *Gemini) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		g.SetEnabled(false)
		return nil
	}

	err := g.SetupDefaults(exch)
	if err != nil {
		return err
	}

	if exch.UseSandbox {
		g.API.Endpoints.URL = geminiSandboxAPIURL
	}

	return g.WebsocketSetup(g.WsConnect,
		exch.Name,
		exch.Features.Enabled.Websocket,
		geminiWebsocketEndpoint,
		exch.API.Endpoints.WebsocketURL)
}

// Start starts the Gemini go routine
func (g *Gemini) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		g.Run()
		wg.Done()
	}()
}

// Run implements the Gemini wrapper
func (g *Gemini) Run() {
	if g.Verbose {
		log.Debugf("%s %d currencies enabled: %s.\n", g.GetName(), len(g.CurrencyPairs.Spot.Enabled), g.CurrencyPairs.Spot.Enabled)
	}

	if !g.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := g.UpdateTradablePairs(false)
	if err != nil {
		log.Errorf("%s failed to update tradable pairs. Err: %s", g.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (g *Gemini) FetchTradablePairs(asset assets.AssetType) ([]string, error) {
	return g.GetSymbols()
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (g *Gemini) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := g.GetSymbols()
	if err != nil {
		return err
	}

	return g.UpdatePairs(pairs, assets.AssetTypeSpot, false, forceUpdate)
}

// GetAccountInfo Retrieves balances for all enabled currencies for the
// Gemini exchange
func (g *Gemini) GetAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.Exchange = g.GetName()
	accountBalance, err := g.GetBalances()
	if err != nil {
		return response, err
	}

	var currencies []exchange.AccountCurrencyInfo
	for i := 0; i < len(accountBalance); i++ {
		var exchangeCurrency exchange.AccountCurrencyInfo
		exchangeCurrency.CurrencyName = accountBalance[i].Currency
		exchangeCurrency.TotalValue = accountBalance[i].Amount
		exchangeCurrency.Hold = accountBalance[i].Available
		currencies = append(currencies, exchangeCurrency)
	}

	response.Accounts = append(response.Accounts, exchange.Account{
		Currencies: currencies,
	})

	return response, nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (g *Gemini) UpdateTicker(p pair.CurrencyPair, assetType assets.AssetType) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := g.GetTicker(p.Pair().String())
	if err != nil {
		return tickerPrice, err
	}
	tickerPrice.Pair = p
	tickerPrice.Ask = tick.Ask
	tickerPrice.Bid = tick.Bid
	tickerPrice.Last = tick.Last
	tickerPrice.Volume = tick.Volume.USD
	ticker.ProcessTicker(g.GetName(), p, tickerPrice, assetType)
	return ticker.GetTicker(g.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (g *Gemini) FetchTicker(p pair.CurrencyPair, assetType assets.AssetType) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(g.GetName(), p, assetType)
	if err != nil {
		return g.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (g *Gemini) FetchOrderbook(p pair.CurrencyPair, assetType assets.AssetType) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(g.GetName(), p, assetType)
	if err != nil {
		return g.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (g *Gemini) UpdateOrderbook(p pair.CurrencyPair, assetType assets.AssetType) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := g.GetOrderbook(p.Pair().String(), url.Values{})
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: orderbookNew.Bids[x].Amount, Price: orderbookNew.Bids[x].Price})
	}

	for x := range orderbookNew.Asks {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: orderbookNew.Asks[x].Amount, Price: orderbookNew.Asks[x].Price})
	}

	orderbook.ProcessOrderbook(g.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(g.Name, p, assetType)
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (g *Gemini) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (g *Gemini) GetExchangeHistory(p pair.CurrencyPair, assetType assets.AssetType) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (g *Gemini) SubmitOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, _ string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	response, err := g.NewOrder(p.Pair().String(), amount, price, side.ToString(), orderType.ToString())

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
func (g *Gemini) ModifyOrder(action exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (g *Gemini) CancelOrder(order exchange.OrderCancellation) error {
	orderIDInt, err := strconv.ParseInt(order.OrderID, 10, 64)
	if err != nil {
		return err
	}

	_, err = g.CancelExistingOrder(orderIDInt)
	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (g *Gemini) CancelAllOrders(_ exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	cancelAllOrdersResponse := exchange.CancelAllOrdersResponse{
		OrderStatus: make(map[string]string),
	}
	resp, err := g.CancelExistingOrders(false)
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	for _, order := range resp.Details.CancelRejects {
		cancelAllOrdersResponse.OrderStatus[order] = "Could not cancel order"
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (g *Gemini) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (g *Gemini) GetDepositAddress(cryptocurrency pair.CurrencyItem, _ string) (string, error) {
	addr, err := g.GetCryptoDepositAddress("", cryptocurrency.String())
	if err != nil {
		return "", err
	}
	return addr.Address, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (g *Gemini) WithdrawCryptocurrencyFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	resp, err := g.WithdrawCrypto(withdrawRequest.Address, withdrawRequest.Currency.String(), withdrawRequest.Amount)
	if err != nil {
		return "", err
	}
	if resp.Result == "error" {
		return "", errors.New(resp.Message)
	}

	return resp.TXHash, err
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (g *Gemini) WithdrawFiatFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (g *Gemini) WithdrawFiatFundsToInternationalBank(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (g *Gemini) GetWebsocket() (*exchange.Websocket, error) {
	return g.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (g *Gemini) GetFeeByType(feeBuilder exchange.FeeBuilder) (float64, error) {
	return g.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (g *Gemini) GetActiveOrders(getOrdersRequest exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	resp, err := g.GetOrders()
	if err != nil {
		return nil, err
	}

	var orders []exchange.OrderDetail
	for _, order := range resp {
		symbol := pair.NewCurrencyPairDelimiter(order.Symbol, g.CurrencyPairs.ConfigFormat.Delimiter)
		var orderType exchange.OrderType
		if order.Type == "exchange limit" {
			orderType = exchange.LimitOrderType
		} else if order.Type == "market buy" || order.Type == "market sell" {
			orderType = exchange.MarketOrderType
		}

		side := exchange.OrderSide(strings.ToUpper(order.Type))
		orderDate := time.Unix(order.Timestamp, 0)

		orders = append(orders, exchange.OrderDetail{
			Amount:          order.OriginalAmount,
			RemainingAmount: order.RemainingAmount,
			ID:              fmt.Sprintf("%v", order.OrderID),
			ExecutedAmount:  order.ExecutedAmount,
			Exchange:        g.Name,
			OrderType:       orderType,
			OrderSide:       side,
			Price:           order.Price,
			CurrencyPair:    symbol,
			OrderDate:       orderDate,
		})
	}

	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks, getOrdersRequest.EndTicks)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)
	exchange.FilterOrdersByType(&orders, getOrdersRequest.OrderType)
	exchange.FilterOrdersByCurrencies(&orders, getOrdersRequest.Currencies)

	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (g *Gemini) GetOrderHistory(getOrdersRequest exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	if len(getOrdersRequest.Currencies) == 0 {
		return nil, errors.New("currency must be supplied")
	}

	var trades []TradeHistory
	for _, currency := range getOrdersRequest.Currencies {
		resp, err := g.GetTradeHistory(g.FormatExchangeCurrency(currency, assets.AssetTypeSpot).String(), getOrdersRequest.StartTicks.Unix())
		if err != nil {
			return nil, err
		}

		for _, trade := range resp {
			trade.BaseCurrency = currency.FirstCurrency.String()
			trade.QuoteCurrency = currency.SecondCurrency.String()
			trades = append(trades, trade)
		}
	}

	var orders []exchange.OrderDetail
	for _, trade := range trades {
		side := exchange.OrderSide(strings.ToUpper(trade.Type))
		orderDate := time.Unix(trade.Timestamp, 0)

		orders = append(orders, exchange.OrderDetail{
			Amount:       trade.Amount,
			ID:           fmt.Sprintf("%v", trade.OrderID),
			Exchange:     g.Name,
			OrderDate:    orderDate,
			OrderSide:    side,
			Fee:          trade.FeeAmount,
			Price:        trade.Price,
			CurrencyPair: pair.NewCurrencyPairWithDelimiter(trade.BaseCurrency, trade.QuoteCurrency, g.CurrencyPairs.ConfigFormat.Delimiter),
		})
	}

	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks, getOrdersRequest.EndTicks)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)

	return orders, nil
}
