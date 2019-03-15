package btcmarkets

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
func (b *BTCMarkets) GetDefaultConfig() (*config.ExchangeConfig, error) {
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

// SetDefaults sets basic defaults
func (b *BTCMarkets) SetDefaults() {
	b.Name = "BTC Markets"
	b.Enabled = true
	b.Verbose = true
	b.API.CredentialsValidator.RequiresKey = true
	b.API.CredentialsValidator.RequiresSecret = true
	b.API.CredentialsValidator.RequiresBase64DecodeSecret = true
	b.API.Endpoints.URLDefault = btcMarketsAPIURL
	b.API.Endpoints.URL = b.API.Endpoints.URLDefault
	b.APIWithdrawPermissions = exchange.AutoWithdrawCrypto |
		exchange.AutoWithdrawFiat

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
			Websocket: false,

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

	b.Requester = request.New(b.Name,
		request.NewRateLimit(time.Second*10, btcmarketsAuthLimit),
		request.NewRateLimit(time.Second*10, btcmarketsUnauthLimit),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
}

// Setup takes in an exchange configuration and sets all parameters
func (b *BTCMarkets) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		b.SetEnabled(false)
		return nil
	}

	return b.SetupDefaults(exch)
}

// Start starts the BTC Markets go routine
func (b *BTCMarkets) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		b.Run()
		wg.Done()
	}()
}

// Run implements the BTC Markets wrapper
func (b *BTCMarkets) Run() {
	if b.Verbose {
		log.Debugf("%s %d currencies enabled: %s.\n", b.GetName(), len(b.CurrencyPairs.Spot.Enabled), b.CurrencyPairs.Spot.Enabled)
	}

	forceUpdate := false
	if !common.StringDataContains(b.CurrencyPairs.Spot.Enabled, "-") || !common.StringDataContains(b.CurrencyPairs.Spot.Available, "-") {
		enabledPairs := []string{"BTC-AUD"}
		log.Println("WARNING: Available pairs for BTC Makrets reset due to config upgrade, please enable the pairs you would like again.")
		forceUpdate = true

		err := b.UpdatePairs(enabledPairs, assets.AssetTypeSpot, true, true)
		if err != nil {
			log.Errorf("%s failed to update currencies. Err: %s", b.Name, err)
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
func (b *BTCMarkets) FetchTradablePairs(asset assets.AssetType) ([]string, error) {
	markets, err := b.GetMarkets()
	if err != nil {
		return nil, err
	}

	var pairs []string
	for x := range markets {
		pairs = append(pairs, markets[x].Instrument+"-"+markets[x].Currency)
	}

	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (b *BTCMarkets) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := b.FetchTradablePairs(assets.AssetTypeSpot)
	if err != nil {
		return err
	}

	return b.UpdatePairs(pairs, assets.AssetTypeSpot, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *BTCMarkets) UpdateTicker(p pair.CurrencyPair, assetType assets.AssetType) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := b.GetTicker(p.FirstCurrency.String(),
		p.SecondCurrency.String())
	if err != nil {
		return tickerPrice, err
	}
	tickerPrice.Pair = p
	tickerPrice.Ask = tick.BestAsk
	tickerPrice.Bid = tick.BestBID
	tickerPrice.Last = tick.LastPrice
	ticker.ProcessTicker(b.GetName(), p, tickerPrice, assetType)
	return ticker.GetTicker(b.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (b *BTCMarkets) FetchTicker(p pair.CurrencyPair, assetType assets.AssetType) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (b *BTCMarkets) FetchOrderbook(p pair.CurrencyPair, assetType assets.AssetType) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *BTCMarkets) UpdateOrderbook(p pair.CurrencyPair, assetType assets.AssetType) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := b.GetOrderbook(p.FirstCurrency.String(),
		p.SecondCurrency.String())
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

	orderbook.ProcessOrderbook(b.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(b.Name, p, assetType)
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// BTCMarkets exchange
func (b *BTCMarkets) GetAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.Exchange = b.GetName()

	accountBalance, err := b.GetAccountBalance()
	if err != nil {
		return response, err
	}

	var currencies []exchange.AccountCurrencyInfo
	for i := 0; i < len(accountBalance); i++ {
		var exchangeCurrency exchange.AccountCurrencyInfo
		exchangeCurrency.CurrencyName = accountBalance[i].Currency
		exchangeCurrency.TotalValue = accountBalance[i].Balance
		exchangeCurrency.Hold = accountBalance[i].PendingFunds

		currencies = append(currencies, exchangeCurrency)
	}

	response.Accounts = append(response.Accounts, exchange.Account{
		Currencies: currencies,
	})

	return response, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (b *BTCMarkets) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (b *BTCMarkets) GetExchangeHistory(p pair.CurrencyPair, assetType assets.AssetType) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (b *BTCMarkets) SubmitOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	response, err := b.NewOrder(p.FirstCurrency.Upper().String(), p.SecondCurrency.Upper().String(), price, amount, side.ToString(), orderType.ToString(), clientID)

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
func (b *BTCMarkets) ModifyOrder(action exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (b *BTCMarkets) CancelOrder(order exchange.OrderCancellation) error {
	orderIDInt, err := strconv.ParseInt(order.OrderID, 10, 64)
	if err != nil {
		return err
	}

	_, err = b.CancelExistingOrder([]int64{orderIDInt})
	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (b *BTCMarkets) CancelAllOrders(_ exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	cancelAllOrdersResponse := exchange.CancelAllOrdersResponse{
		OrderStatus: make(map[string]string),
	}
	openOrders, err := b.GetOpenOrders()
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	var orderList []int64
	for _, order := range openOrders {
		orderList = append(orderList, order.ID)
	}

	if len(orderList) > 0 {
		orders, err := b.CancelExistingOrder(orderList)
		if err != nil {
			return cancelAllOrdersResponse, err
		}

		for _, order := range orders {
			if err != nil {
				cancelAllOrdersResponse.OrderStatus[strconv.FormatInt(order.ID, 10)] = err.Error()
			}
		}
	}
	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (b *BTCMarkets) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	var OrderDetail exchange.OrderDetail

	o, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil {
		return OrderDetail, err
	}

	orders, err := b.GetOrderDetail([]int64{o})
	if err != nil {
		return OrderDetail, err
	}

	if len(orders) > 1 {
		return OrderDetail, errors.New("too many orders returned")
	}

	if len(orders) == 0 {
		return OrderDetail, errors.New("no orders found")
	}

	for _, order := range orders {
		var side exchange.OrderSide
		if strings.EqualFold(order.OrderSide, exchange.AskOrderSide.ToString()) {
			side = exchange.SellOrderSide
		} else if strings.EqualFold(order.OrderSide, exchange.BidOrderSide.ToString()) {
			side = exchange.BuyOrderSide
		}
		orderDate := time.Unix(int64(order.CreationTime), 0)
		orderType := exchange.OrderType(strings.ToUpper(order.OrderType))

		OrderDetail.Amount = order.Volume
		OrderDetail.OrderDate = orderDate
		OrderDetail.Exchange = b.GetName()
		OrderDetail.ID = strconv.FormatInt(order.ID, 10)
		OrderDetail.RemainingAmount = order.OpenVolume
		OrderDetail.OrderSide = side
		OrderDetail.OrderType = orderType
		OrderDetail.Price = order.Price
		OrderDetail.Status = order.Status
		OrderDetail.CurrencyPair = pair.NewCurrencyPairWithDelimiter(order.Instrument, order.Currency, b.CurrencyPairs.ConfigFormat.Delimiter)
	}

	return OrderDetail, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *BTCMarkets) GetDepositAddress(cryptocurrency pair.CurrencyItem, accountID string) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is submitted
func (b *BTCMarkets) WithdrawCryptocurrencyFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return b.WithdrawCrypto(withdrawRequest.Amount, withdrawRequest.Currency.String(), withdrawRequest.Address)
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (b *BTCMarkets) WithdrawFiatFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	if withdrawRequest.Currency != symbol.AUD {
		return "", errors.New("only AUD is supported for withdrawals")
	}
	return b.WithdrawAUD(withdrawRequest.BankAccountName, fmt.Sprintf("%v", withdrawRequest.BankAccountNumber), withdrawRequest.BankName, fmt.Sprintf("%v", withdrawRequest.BankCode), withdrawRequest.Amount)
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (b *BTCMarkets) WithdrawFiatFundsToInternationalBank(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (b *BTCMarkets) GetWebsocket() (*exchange.Websocket, error) {
	return nil, common.ErrNotYetImplemented
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (b *BTCMarkets) GetFeeByType(feeBuilder exchange.FeeBuilder) (float64, error) {
	return b.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (b *BTCMarkets) GetActiveOrders(getOrdersRequest exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	resp, err := b.GetOpenOrders()
	if err != nil {
		return nil, err
	}

	var orders []exchange.OrderDetail
	for _, order := range resp {
		var side exchange.OrderSide
		if strings.EqualFold(order.OrderSide, exchange.AskOrderSide.ToString()) {
			side = exchange.SellOrderSide
		} else if strings.EqualFold(order.OrderSide, exchange.BidOrderSide.ToString()) {
			side = exchange.BuyOrderSide
		}
		orderDate := time.Unix(int64(order.CreationTime), 0)
		orderType := exchange.OrderType(strings.ToUpper(order.OrderType))

		openOrder := exchange.OrderDetail{
			ID:              strconv.FormatInt(order.ID, 10),
			Amount:          order.Volume,
			Exchange:        b.Name,
			RemainingAmount: order.OpenVolume,
			OrderDate:       orderDate,
			OrderSide:       side,
			OrderType:       orderType,
			Price:           order.Price,
			Status:          order.Status,
			CurrencyPair:    pair.NewCurrencyPairWithDelimiter(order.Instrument, order.Currency, b.CurrencyPairs.ConfigFormat.Delimiter),
		}

		for _, trade := range order.Trades {
			tradeDate := time.Unix(int64(trade.CreationTime), 0)
			openOrder.Trades = append(openOrder.Trades, exchange.TradeHistory{
				Amount:      trade.Volume,
				Exchange:    b.Name,
				Price:       trade.Price,
				TID:         trade.ID,
				Timestamp:   tradeDate,
				Fee:         trade.Fee,
				Description: trade.Description,
			})
		}

		orders = append(orders, openOrder)
	}

	exchange.FilterOrdersByType(&orders, getOrdersRequest.OrderType)
	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks, getOrdersRequest.EndTicks)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)

	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *BTCMarkets) GetOrderHistory(getOrdersRequest exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	if len(getOrdersRequest.Currencies) == 0 {
		return nil, errors.New("requires at least one currency pair to retrieve history")
	}

	var respOrders []Order
	for _, currency := range getOrdersRequest.Currencies {
		resp, err := b.GetOrders(currency.FirstCurrency.String(), currency.SecondCurrency.String(), 200, 0, true)
		if err != nil {
			return nil, err
		}
		respOrders = append(respOrders, resp...)
	}

	var orders []exchange.OrderDetail
	for _, order := range respOrders {
		var side exchange.OrderSide
		if strings.EqualFold(order.OrderSide, exchange.AskOrderSide.ToString()) {
			side = exchange.SellOrderSide
		} else if strings.EqualFold(order.OrderSide, exchange.BidOrderSide.ToString()) {
			side = exchange.BuyOrderSide
		}
		orderDate := time.Unix(int64(order.CreationTime), 0)
		orderType := exchange.OrderType(strings.ToUpper(order.OrderType))

		openOrder := exchange.OrderDetail{
			ID:              strconv.FormatInt(order.ID, 10),
			Amount:          order.Volume,
			Exchange:        b.Name,
			RemainingAmount: order.OpenVolume,
			OrderDate:       orderDate,
			OrderSide:       side,
			OrderType:       orderType,
			Price:           order.Price,
			Status:          order.Status,
			CurrencyPair:    pair.NewCurrencyPairWithDelimiter(order.Instrument, order.Currency, b.CurrencyPairs.ConfigFormat.Delimiter),
		}

		for _, trade := range order.Trades {
			tradeDate := time.Unix(int64(trade.CreationTime), 0)
			openOrder.Trades = append(openOrder.Trades, exchange.TradeHistory{
				Amount:      trade.Volume,
				Exchange:    b.Name,
				Price:       trade.Price,
				TID:         trade.ID,
				Timestamp:   tradeDate,
				Fee:         trade.Fee,
				Description: trade.Description,
			})
		}

		orders = append(orders, openOrder)
	}

	exchange.FilterOrdersByType(&orders, getOrdersRequest.OrderType)
	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks, getOrdersRequest.EndTicks)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)

	return orders, nil
}
