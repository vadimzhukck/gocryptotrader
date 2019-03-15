package bitfinex

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/symbol"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	log "github.com/thrasher-/gocryptotrader/logger"
)

const (
	bitfinexAPIURLBase         = "https://api.bitfinex.com"
	bitfinexAPIVersion         = "/v1/"
	bitfinexAPIVersion2        = "2"
	bitfinexTickerV2           = "ticker"
	bitfinexTickersV2          = "tickers"
	bitfinexTicker             = "pubticker/"
	bitfinexStats              = "stats/"
	bitfinexLendbook           = "lendbook/"
	bitfinexOrderbookV2        = "book"
	bitfinexOrderbook          = "book/"
	bitfinexTrades             = "trades/"
	bitfinexTradesV2           = "https://api.bitfinex.com/v2/trades/%s/hist?limit=1000&start=%s&end=%s"
	bitfinexKeyPermissions     = "key_info"
	bitfinexLends              = "lends/"
	bitfinexSymbols            = "symbols/"
	bitfinexSymbolsDetails     = "symbols_details/"
	bitfinexAccountInfo        = "account_infos"
	bitfinexAccountFees        = "account_fees"
	bitfinexAccountSummary     = "summary"
	bitfinexDeposit            = "deposit/new"
	bitfinexOrderNew           = "order/new"
	bitfinexOrderNewMulti      = "order/new/multi"
	bitfinexOrderCancel        = "order/cancel"
	bitfinexOrderCancelMulti   = "order/cancel/multi"
	bitfinexOrderCancelAll     = "order/cancel/all"
	bitfinexOrderCancelReplace = "order/cancel/replace"
	bitfinexOrderStatus        = "order/status"
	bitfinexOrders             = "orders"
	bitfinexInactiveOrders     = "orders/hist"
	bitfinexPositions          = "positions"
	bitfinexClaimPosition      = "position/claim"
	bitfinexHistory            = "history"
	bitfinexHistoryMovements   = "history/movements"
	bitfinexTradeHistory       = "mytrades"
	bitfinexOfferNew           = "offer/new"
	bitfinexOfferCancel        = "offer/cancel"
	bitfinexOfferStatus        = "offer/status"
	bitfinexOffers             = "offers"
	bitfinexMarginActiveFunds  = "taken_funds"
	bitfinexMarginTotalFunds   = "total_taken_funds"
	bitfinexMarginUnusedFunds  = "unused_taken_funds"
	bitfinexMarginClose        = "funding/close"
	bitfinexBalances           = "balances"
	bitfinexMarginInfo         = "margin_infos"
	bitfinexTransfer           = "transfer"
	bitfinexWithdrawal         = "withdraw"
	bitfinexActiveCredits      = "credits"
	bitfinexPlatformStatus     = "platform/status"

	// requests per minute
	bitfinexAuthRate   = 10
	bitfinexUnauthRate = 10

	// Bitfinex platform status values
	// When the platform is marked in maintenance mode bots should stop trading
	// activity. Cancelling orders will be still possible.
	bitfinexMaintenanceMode = 0
	bitfinexOperativeMode   = 1
)

// Bitfinex is the overarching type across the bitfinex package
// Notes: Bitfinex has added a rate limit to the number of REST requests.
// Rate limit policy can vary in a range of 10 to 90 requests per minute
// depending on some factors (e.g. servers load, endpoint, etc.).
type Bitfinex struct {
	exchange.Base
	WebsocketConn         *websocket.Conn
	WebsocketSubdChannels map[int]WebsocketChanInfo
}

// GetPlatformStatus returns the Bifinex platform status
func (b *Bitfinex) GetPlatformStatus() (int, error) {
	var response []interface{}
	path := fmt.Sprintf("%s/v%s/%s", b.API.Endpoints.URL, bitfinexAPIVersion2,
		bitfinexPlatformStatus)

	err := b.SendHTTPRequest(path, &response, b.Verbose)
	if err != nil {
		return 0, err
	}

	if (len(response)) != 1 {
		return 0, errors.New("unexpected platform status value")
	}

	return int(response[0].(float64)), nil
}

// GetLatestSpotPrice returns latest spot price of symbol
//
// symbol: string of currency pair
func (b *Bitfinex) GetLatestSpotPrice(symbol string) (float64, error) {
	res, err := b.GetTicker(symbol)
	if err != nil {
		return 0, err
	}
	return res.Mid, nil
}

// GetTicker returns ticker information
func (b *Bitfinex) GetTicker(symbol string) (Ticker, error) {
	response := Ticker{}
	path := common.EncodeURLValues(b.API.Endpoints.URL+bitfinexAPIVersion+bitfinexTicker+symbol, url.Values{})

	if err := b.SendHTTPRequest(path, &response, b.Verbose); err != nil {
		return response, err
	}

	if response.Message != "" {
		return response, errors.New(response.Message)
	}

	return response, nil
}

// GetTickerV2 returns ticker information
func (b *Bitfinex) GetTickerV2(symb string) (Tickerv2, error) {
	var response []interface{}
	var tick Tickerv2

	path := fmt.Sprintf("%s/v%s/%s/%s", b.API.Endpoints.URL, bitfinexAPIVersion2, bitfinexTickerV2, symb)
	err := b.SendHTTPRequest(path, &response, b.Verbose)
	if err != nil {
		return tick, err
	}

	if len(response) > 10 {
		tick.FlashReturnRate = response[0].(float64)
		tick.Bid = response[1].(float64)
		tick.BidSize = response[2].(float64)
		tick.BidPeriod = int64(response[3].(float64))
		tick.Ask = response[4].(float64)
		tick.AskSize = response[5].(float64)
		tick.AskPeriod = int64(response[6].(float64))
		tick.DailyChange = response[7].(float64)
		tick.DailyChangePerc = response[8].(float64)
		tick.Last = response[9].(float64)
		tick.Volume = response[10].(float64)
		tick.High = response[11].(float64)
		tick.Low = response[12].(float64)
	} else {
		tick.Bid = response[0].(float64)
		tick.BidSize = response[1].(float64)
		tick.Ask = response[2].(float64)
		tick.AskSize = response[3].(float64)
		tick.DailyChange = response[4].(float64)
		tick.DailyChangePerc = response[5].(float64)
		tick.Last = response[6].(float64)
		tick.Volume = response[7].(float64)
		tick.High = response[8].(float64)
		tick.Low = response[9].(float64)
	}
	return tick, nil
}

// GetTickersV2 returns ticker information for multiple symbols
func (b *Bitfinex) GetTickersV2(symbols string) ([]Tickersv2, error) {
	var response [][]interface{}
	var tickers []Tickersv2

	v := url.Values{}
	v.Set("symbols", symbols)

	path := common.EncodeURLValues(fmt.Sprintf("%s/v%s/%s",
		b.API.Endpoints.URL,
		bitfinexAPIVersion2,
		bitfinexTickersV2), v)

	err := b.SendHTTPRequest(path, &response, b.Verbose)
	if err != nil {
		return nil, err
	}

	for x := range response {
		var tick Tickersv2
		data := response[x]
		if len(data) > 11 {
			tick.Symbol = data[0].(string)
			tick.FlashReturnRate = data[1].(float64)
			tick.Bid = data[2].(float64)
			tick.BidSize = data[3].(float64)
			tick.BidPeriod = int64(data[4].(float64))
			tick.Ask = data[5].(float64)
			tick.AskSize = data[6].(float64)
			tick.AskPeriod = int64(data[7].(float64))
			tick.DailyChange = data[8].(float64)
			tick.DailyChangePerc = data[9].(float64)
			tick.Last = data[10].(float64)
			tick.Volume = data[11].(float64)
			tick.High = data[12].(float64)
			tick.Low = data[13].(float64)
		} else {
			tick.Symbol = data[0].(string)
			tick.Bid = data[1].(float64)
			tick.BidSize = data[2].(float64)
			tick.Ask = data[3].(float64)
			tick.AskSize = data[4].(float64)
			tick.DailyChange = data[5].(float64)
			tick.DailyChangePerc = data[6].(float64)
			tick.Last = data[7].(float64)
			tick.Volume = data[8].(float64)
			tick.High = data[9].(float64)
			tick.Low = data[10].(float64)
		}
		tickers = append(tickers, tick)
	}
	return tickers, nil
}

// GetStats returns various statistics about the requested pair
func (b *Bitfinex) GetStats(symbol string) ([]Stat, error) {
	response := []Stat{}
	path := fmt.Sprint(b.API.Endpoints.URL + bitfinexAPIVersion + bitfinexStats + symbol)

	return response, b.SendHTTPRequest(path, &response, b.Verbose)
}

// GetFundingBook the entire margin funding book for both bids and asks sides
// per currency string
// symbol - example "USD"
func (b *Bitfinex) GetFundingBook(symbol string) (FundingBook, error) {
	response := FundingBook{}
	path := fmt.Sprint(b.API.Endpoints.URL + bitfinexAPIVersion + bitfinexLendbook + symbol)

	if err := b.SendHTTPRequest(path, &response, b.Verbose); err != nil {
		return response, err
	}

	if response.Message != "" {
		return response, errors.New(response.Message)
	}

	return response, nil
}

// GetOrderbook retieves the orderbook bid and ask price points for a currency
// pair - By default the response will return 25 bid and 25 ask price points.
// CurrencyPair - Example "BTCUSD"
// Values can contain limit amounts for both the asks and bids - Example
// "limit_bids" = 1000
func (b *Bitfinex) GetOrderbook(currencyPair string, values url.Values) (Orderbook, error) {
	response := Orderbook{}
	path := common.EncodeURLValues(
		b.API.Endpoints.URL+bitfinexAPIVersion+bitfinexOrderbook+currencyPair,
		values,
	)
	return response, b.SendHTTPRequest(path, &response, b.Verbose)
}

// GetOrderbookV2 retieves the orderbook bid and ask price points for a currency
// pair - By default the response will return 25 bid and 25 ask price points.
// symbol - Example "tBTCUSD"
// precision - P0,P1,P2,P3,R0
// Values can contain limit amounts for both the asks and bids - Example
// "len" = 1000
func (b *Bitfinex) GetOrderbookV2(symbol, precision string, values url.Values) (OrderbookV2, error) {
	var response [][]interface{}
	var book OrderbookV2
	path := common.EncodeURLValues(fmt.Sprintf("%s/v%s/%s/%s/%s", b.API.Endpoints.URL,
		bitfinexAPIVersion2, bitfinexOrderbookV2, symbol, precision), values)
	err := b.SendHTTPRequest(path, &response, b.Verbose)
	if err != nil {
		return book, err
	}

	for x := range response {
		data := response[x]
		bookItem := BookV2{}

		if len(data) > 3 {
			bookItem.Rate = data[0].(float64)
			bookItem.Price = data[1].(float64)
			bookItem.Count = int64(data[2].(float64))
			bookItem.Amount = data[3].(float64)
		} else {
			bookItem.Price = data[0].(float64)
			bookItem.Count = int64(data[1].(float64))
			bookItem.Amount = data[2].(float64)
		}

		if symbol[0] == 't' {
			if bookItem.Amount > 0 {
				book.Bids = append(book.Bids, bookItem)
			} else {
				book.Asks = append(book.Asks, bookItem)
			}
		} else {
			if bookItem.Amount > 0 {
				book.Asks = append(book.Asks, bookItem)
			} else {
				book.Bids = append(book.Bids, bookItem)
			}
		}
	}
	return book, nil
}

// GetTrades returns a list of the most recent trades for the given curencyPair
// By default the response will return 100 trades
// CurrencyPair - Example "BTCUSD"
// Values can contain limit amounts for the number of trades returned - Example
// "limit_trades" = 1000
func (b *Bitfinex) GetTrades(currencyPair string, values url.Values) ([]TradeStructure, error) {
	response := []TradeStructure{}
	path := common.EncodeURLValues(
		b.API.Endpoints.URL+bitfinexAPIVersion+bitfinexTrades+currencyPair,
		values,
	)
	return response, b.SendHTTPRequest(path, &response, b.Verbose)
}

// GetTradesV2 uses the V2 API to get historic trades that occurred on the
// exchange
//
// currencyPair e.g. "tBTCUSD" v2 prefixes currency pairs with t. (?)
// timestampStart is an int64 unix epoch time
// timestampEnd is an int64 unix epoch time, make sure this is always there or
// you will get the most recent trades.
// reOrderResp reorders the returned data.
func (b *Bitfinex) GetTradesV2(currencyPair string, timestampStart, timestampEnd int64, reOrderResp bool) ([]TradeStructureV2, error) {
	var resp [][]interface{}
	var actualHistory []TradeStructureV2

	path := fmt.Sprintf(bitfinexTradesV2,
		currencyPair,
		strconv.FormatInt(timestampStart, 10),
		strconv.FormatInt(timestampEnd, 10))

	err := b.SendHTTPRequest(path, &resp, b.Verbose)
	if err != nil {
		return actualHistory, err
	}

	var tempHistory TradeStructureV2
	for _, data := range resp {
		tempHistory.TID = int64(data[0].(float64))
		tempHistory.Timestamp = int64(data[1].(float64))
		tempHistory.Amount = data[2].(float64)
		tempHistory.Price = data[3].(float64)
		tempHistory.Exchange = b.Name
		tempHistory.Type = "BUY"

		if tempHistory.Amount < 0 {
			tempHistory.Type = "SELL"
			tempHistory.Amount *= -1
		}

		actualHistory = append(actualHistory, tempHistory)
	}

	// re-order index
	if reOrderResp {
		orderedHistory := make([]TradeStructureV2, len(actualHistory))
		for i, quickRange := range actualHistory {
			orderedHistory[len(actualHistory)-i-1] = quickRange
		}
		return orderedHistory, nil
	}
	return actualHistory, nil
}

// GetLendbook returns a list of the most recent funding data for the given
// currency: total amount provided and Flash Return Rate (in % by 365 days) over
// time
// Symbol - example "USD"
func (b *Bitfinex) GetLendbook(symbol string, values url.Values) (Lendbook, error) {
	response := Lendbook{}
	if len(symbol) == 6 {
		symbol = symbol[:3]
	}
	path := common.EncodeURLValues(b.API.Endpoints.URL+bitfinexAPIVersion+bitfinexLendbook+symbol, values)

	return response, b.SendHTTPRequest(path, &response, b.Verbose)
}

// GetLends returns a list of the most recent funding data for the given
// currency: total amount provided and Flash Return Rate (in % by 365 days)
// over time
// Symbol - example "USD"
func (b *Bitfinex) GetLends(symbol string, values url.Values) ([]Lends, error) {
	response := []Lends{}
	path := common.EncodeURLValues(b.API.Endpoints.URL+bitfinexAPIVersion+bitfinexLends+symbol, values)

	return response, b.SendHTTPRequest(path, &response, b.Verbose)
}

// GetSymbols returns the available currency pairs on the exchange
func (b *Bitfinex) GetSymbols() ([]string, error) {
	products := []string{}
	path := fmt.Sprint(b.API.Endpoints.URL + bitfinexAPIVersion + bitfinexSymbols)

	return products, b.SendHTTPRequest(path, &products, b.Verbose)
}

// GetSymbolsDetails a list of valid symbol IDs and the pair details
func (b *Bitfinex) GetSymbolsDetails() ([]SymbolDetails, error) {
	response := []SymbolDetails{}
	path := fmt.Sprint(b.API.Endpoints.URL + bitfinexAPIVersion + bitfinexSymbolsDetails)

	return response, b.SendHTTPRequest(path, &response, b.Verbose)
}

// GetAccountInformation returns information about your account incl. trading fees
func (b *Bitfinex) GetAccountInformation() ([]AccountInfo, error) {
	var responses []AccountInfo
	return responses, b.SendAuthenticatedHTTPRequest(http.MethodPost, bitfinexAccountInfo, nil, &responses)
}

// GetAccountFees - Gets all fee rates for all currencies
func (b *Bitfinex) GetAccountFees() (AccountFees, error) {
	response := AccountFees{}
	return response, b.SendAuthenticatedHTTPRequest(http.MethodPost, bitfinexAccountFees, nil, &response)
}

// GetAccountSummary returns a 30-day summary of your trading volume and return
// on margin funding
func (b *Bitfinex) GetAccountSummary() (AccountSummary, error) {
	response := AccountSummary{}

	return response,
		b.SendAuthenticatedHTTPRequest(
			http.MethodPost, bitfinexAccountSummary, nil, &response,
		)
}

// NewDeposit returns a new deposit address
// Method - Example methods accepted: “bitcoin”, “litecoin”, “ethereum”,
// “tethers", "ethereumc", "zcash", "monero", "iota", "bcash"
// WalletName - accepted: “trading”, “exchange”, “deposit”
// renew - Default is 0. If set to 1, will return a new unused deposit address
func (b *Bitfinex) NewDeposit(method, walletName string, renew int) (DepositResponse, error) {
	response := DepositResponse{}
	req := make(map[string]interface{})
	req["method"] = method
	req["wallet_name"] = walletName
	req["renew"] = renew

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost, bitfinexDeposit, req, &response)
}

// GetKeyPermissions checks the permissions of the key being used to generate
// this request.
func (b *Bitfinex) GetKeyPermissions() (KeyPermissions, error) {
	response := KeyPermissions{}

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost, bitfinexKeyPermissions, nil, &response)
}

// GetMarginInfo shows your trading wallet information for margin trading
func (b *Bitfinex) GetMarginInfo() ([]MarginInfo, error) {
	response := []MarginInfo{}

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost, bitfinexMarginInfo, nil, &response)
}

// GetAccountBalance returns full wallet balance information
func (b *Bitfinex) GetAccountBalance() ([]Balance, error) {
	response := []Balance{}

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost, bitfinexBalances, nil, &response)
}

// WalletTransfer move available balances between your wallets
// Amount - Amount to move
// Currency -  example "BTC"
// WalletFrom - example "exchange"
// WalletTo -  example "deposit"
func (b *Bitfinex) WalletTransfer(amount float64, currency, walletFrom, walletTo string) ([]WalletTransfer, error) {
	response := []WalletTransfer{}
	req := make(map[string]interface{})
	req["amount"] = amount
	req["currency"] = currency
	req["walletfrom"] = walletFrom
	req["walletTo"] = walletTo

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost, bitfinexTransfer, req, &response)
}

// WithdrawCryptocurrency requests a withdrawal from one of your wallets.
// For FIAT, use WithdrawFIAT
func (b *Bitfinex) WithdrawCryptocurrency(withdrawType, wallet, address, currency, paymentID string, amount float64) ([]Withdrawal, error) {
	response := []Withdrawal{}
	req := make(map[string]interface{})
	req["withdraw_type"] = withdrawType
	req["walletselected"] = wallet
	req["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	req["address"] = address
	if currency == symbol.XMR {
		req["paymend_id"] = paymentID
	}

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost, bitfinexWithdrawal, req, &response)
}

// WithdrawFIAT requests a withdrawal from one of your wallets.
// For Cryptocurrency, use WithdrawCryptocurrency
func (b *Bitfinex) WithdrawFIAT(withdrawType, wallet, wireCurrency,
	accountName, bankName, bankAddress, bankCity, bankCountry, swift, transactionMessage,
	intermediaryBankName, intermediaryBankAddress, intermediaryBankCity, intermediaryBankCountry, intermediaryBankSwift string,
	amount, accountNumber, intermediaryBankAccountNumber float64, isExpressWire, requiresIntermediaryBank bool) ([]Withdrawal, error) {
	response := []Withdrawal{}
	req := make(map[string]interface{})
	req["withdraw_type"] = withdrawType
	req["walletselected"] = wallet
	req["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	req["account_name"] = accountName
	req["account_number"] = strconv.FormatFloat(accountNumber, 'f', -1, 64)
	req["bank_name"] = bankName
	req["bank_address"] = bankAddress
	req["bank_city"] = bankCity
	req["bank_country"] = bankCountry
	req["expressWire"] = isExpressWire
	req["swift"] = swift
	req["detail_payment"] = transactionMessage
	req["currency"] = wireCurrency
	req["account_address"] = bankAddress

	if requiresIntermediaryBank {
		req["intermediary_bank_name"] = intermediaryBankName
		req["intermediary_bank_address"] = intermediaryBankAddress
		req["intermediary_bank_city"] = intermediaryBankCity
		req["intermediary_bank_country"] = intermediaryBankCountry
		req["intermediary_bank_account"] = strconv.FormatFloat(intermediaryBankAccountNumber, 'f', -1, 64)
		req["intermediary_bank_swift"] = intermediaryBankSwift
	}

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost, bitfinexWithdrawal, req, &response)
}

// NewOrder submits a new order and returns a order information
// Major Upgrade needed on this function to include all query params
func (b *Bitfinex) NewOrder(currencyPair string, amount, price float64, buy bool, orderType string, hidden bool) (Order, error) {
	response := Order{}
	req := make(map[string]interface{})
	req["symbol"] = currencyPair
	req["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	req["price"] = strconv.FormatFloat(price, 'f', -1, 64)
	req["exchange"] = "bitfinex"
	req["type"] = orderType
	req["is_hidden"] = hidden

	if buy {
		req["side"] = "buy"
	} else {
		req["side"] = "sell"
	}

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost, bitfinexOrderNew, req, &response)
}

// NewOrderMulti allows several new orders at once
func (b *Bitfinex) NewOrderMulti(orders []PlaceOrder) (OrderMultiResponse, error) {
	response := OrderMultiResponse{}
	req := make(map[string]interface{})
	req["orders"] = orders

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost, bitfinexOrderNewMulti, req, &response)
}

// CancelExistingOrder cancels a single order by OrderID
func (b *Bitfinex) CancelExistingOrder(orderID int64) (Order, error) {
	response := Order{}
	req := make(map[string]interface{})
	req["order_id"] = orderID

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost, bitfinexOrderCancel, req, &response)
}

// CancelMultipleOrders cancels multiple orders
func (b *Bitfinex) CancelMultipleOrders(orderIDs []int64) (string, error) {
	response := GenericResponse{}
	req := make(map[string]interface{})
	req["order_ids"] = orderIDs

	return response.Result,
		b.SendAuthenticatedHTTPRequest(http.MethodPost, bitfinexOrderCancelMulti, req, nil)
}

// CancelAllExistingOrders cancels all active and open orders
func (b *Bitfinex) CancelAllExistingOrders() (string, error) {
	response := GenericResponse{}

	return response.Result,
		b.SendAuthenticatedHTTPRequest(http.MethodPost, bitfinexOrderCancelAll, nil, nil)
}

// ReplaceOrder replaces an older order with a new order
func (b *Bitfinex) ReplaceOrder(orderID int64, symbol string, amount, price float64, buy bool, orderType string, hidden bool) (Order, error) {
	response := Order{}
	req := make(map[string]interface{})
	req["order_id"] = orderID
	req["symbol"] = symbol
	req["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	req["price"] = strconv.FormatFloat(price, 'f', -1, 64)
	req["exchange"] = "bitfinex"
	req["type"] = orderType
	req["is_hidden"] = hidden

	if buy {
		req["side"] = "buy"
	} else {
		req["side"] = "sell"
	}

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost, bitfinexOrderCancelReplace, req, &response)
}

// GetOrderStatus returns order status information
func (b *Bitfinex) GetOrderStatus(orderID int64) (Order, error) {
	orderStatus := Order{}
	req := make(map[string]interface{})
	req["order_id"] = orderID

	return orderStatus,
		b.SendAuthenticatedHTTPRequest(http.MethodPost, bitfinexOrderStatus, req, &orderStatus)
}

// GetInactiveOrders returns order status information
func (b *Bitfinex) GetInactiveOrders() ([]Order, error) {
	var response []Order
	req := make(map[string]interface{})
	req["limit"] = "100"

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost, bitfinexInactiveOrders, req, &response)
}

// GetOpenOrders returns all active orders and statuses
func (b *Bitfinex) GetOpenOrders() ([]Order, error) {
	var response []Order

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost, bitfinexOrders, nil, &response)
}

// GetActivePositions returns an array of active positions
func (b *Bitfinex) GetActivePositions() ([]Position, error) {
	response := []Position{}

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost, bitfinexPositions, nil, &response)
}

// ClaimPosition allows positions to be claimed
func (b *Bitfinex) ClaimPosition(positionID int) (Position, error) {
	response := Position{}
	req := make(map[string]interface{})
	req["position_id"] = positionID

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost, bitfinexClaimPosition, nil, nil)
}

// GetBalanceHistory returns balance history for the account
func (b *Bitfinex) GetBalanceHistory(symbol string, timeSince, timeUntil time.Time, limit int, wallet string) ([]BalanceHistory, error) {
	response := []BalanceHistory{}
	req := make(map[string]interface{})
	req["currency"] = symbol

	if !timeSince.IsZero() {
		req["since"] = timeSince
	}
	if !timeUntil.IsZero() {
		req["until"] = timeUntil
	}
	if limit > 0 {
		req["limit"] = limit
	}
	if len(wallet) > 0 {
		req["wallet"] = wallet
	}

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost, bitfinexHistory, req, &response)
}

// GetMovementHistory returns an array of past deposits and withdrawals
func (b *Bitfinex) GetMovementHistory(symbol, method string, timeSince, timeUntil time.Time, limit int) ([]MovementHistory, error) {
	response := []MovementHistory{}
	req := make(map[string]interface{})
	req["currency"] = symbol

	if len(method) > 0 {
		req["method"] = method
	}
	if !timeSince.IsZero() {
		req["since"] = timeSince
	}
	if !timeUntil.IsZero() {
		req["until"] = timeUntil
	}
	if limit > 0 {
		req["limit"] = limit
	}

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost, bitfinexHistoryMovements, req, &response)
}

// GetTradeHistory returns past executed trades
func (b *Bitfinex) GetTradeHistory(currencyPair string, timestamp, until time.Time, limit, reverse int) ([]TradeHistory, error) {
	response := []TradeHistory{}
	req := make(map[string]interface{})
	req["currency"] = currencyPair
	req["timestamp"] = timestamp

	if !until.IsZero() {
		req["until"] = until
	}
	if limit > 0 {
		req["limit"] = limit
	}
	if reverse > 0 {
		req["reverse"] = reverse
	}

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost, bitfinexTradeHistory, req, &response)
}

// NewOffer submits a new offer
func (b *Bitfinex) NewOffer(symbol string, amount, rate float64, period int64, direction string) (Offer, error) {
	response := Offer{}
	req := make(map[string]interface{})
	req["currency"] = symbol
	req["amount"] = amount
	req["rate"] = rate
	req["period"] = period
	req["direction"] = direction

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost, bitfinexOfferNew, req, &response)
}

// CancelOffer cancels offer by offerID
func (b *Bitfinex) CancelOffer(offerID int64) (Offer, error) {
	response := Offer{}
	req := make(map[string]interface{})
	req["offer_id"] = offerID

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost, bitfinexOfferCancel, req, &response)
}

// GetOfferStatus checks offer status whether it has been cancelled, execute or
// is still active
func (b *Bitfinex) GetOfferStatus(offerID int64) (Offer, error) {
	response := Offer{}
	req := make(map[string]interface{})
	req["offer_id"] = offerID

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost, bitfinexOrderStatus, req, &response)
}

// GetActiveCredits returns all available credits
func (b *Bitfinex) GetActiveCredits() ([]Offer, error) {
	response := []Offer{}

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost, bitfinexActiveCredits, nil, &response)
}

// GetActiveOffers returns all current active offers
func (b *Bitfinex) GetActiveOffers() ([]Offer, error) {
	response := []Offer{}

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost, bitfinexOffers, nil, &response)
}

// GetActiveMarginFunding returns an array of active margin funds
func (b *Bitfinex) GetActiveMarginFunding() ([]MarginFunds, error) {
	response := []MarginFunds{}

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost, bitfinexMarginActiveFunds, nil, &response)
}

// GetUnusedMarginFunds returns an array of funding borrowed but not currently
// used
func (b *Bitfinex) GetUnusedMarginFunds() ([]MarginFunds, error) {
	response := []MarginFunds{}

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost, bitfinexMarginUnusedFunds, nil, &response)
}

// GetMarginTotalTakenFunds returns an array of active funding used in a
// position
func (b *Bitfinex) GetMarginTotalTakenFunds() ([]MarginTotalTakenFunds, error) {
	response := []MarginTotalTakenFunds{}

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost, bitfinexMarginTotalFunds, nil, &response)
}

// CloseMarginFunding closes an unused or used taken fund
func (b *Bitfinex) CloseMarginFunding(swapID int64) (Offer, error) {
	response := Offer{}
	req := make(map[string]interface{})
	req["swap_id"] = swapID

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost, bitfinexMarginClose, req, &response)
}

// SendHTTPRequest sends an unauthenticated request
func (b *Bitfinex) SendHTTPRequest(path string, result interface{}, verbose bool) error {
	return b.SendPayload(http.MethodGet, path, nil, nil, result, false, verbose)
}

// SendAuthenticatedHTTPRequest sends an autheticated http request and json
// unmarshals result to a supplied variable
func (b *Bitfinex) SendAuthenticatedHTTPRequest(method, path string, params map[string]interface{}, result interface{}) error {
	if !b.AllowAuthenticatedRequest() {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, b.Name)
	}

	if b.Nonce.Get() == 0 {
		b.Nonce.Set(time.Now().UnixNano())
	} else {
		b.Nonce.Inc()
	}

	req := make(map[string]interface{})
	req["request"] = fmt.Sprintf("%s%s", bitfinexAPIVersion, path)
	req["nonce"] = b.Nonce.String()

	for key, value := range params {
		req[key] = value
	}

	PayloadJSON, err := common.JSONEncode(req)
	if err != nil {
		return errors.New("sendAuthenticatedAPIRequest: unable to JSON request")
	}

	if b.Verbose {
		log.Debugf("Request JSON: %s\n", PayloadJSON)
	}

	PayloadBase64 := common.Base64Encode(PayloadJSON)
	hmac := common.GetHMAC(common.HashSHA512_384, []byte(PayloadBase64), []byte(b.API.Credentials.Secret))
	headers := make(map[string]string)
	headers["X-BFX-APIKEY"] = b.API.Credentials.Key
	headers["X-BFX-PAYLOAD"] = PayloadBase64
	headers["X-BFX-SIGNATURE"] = common.HexEncodeToString(hmac)

	return b.SendPayload(method, b.API.Endpoints.URL+bitfinexAPIVersion+path, headers, nil, result, true, b.Verbose)
}

// GetFee returns an estimate of fee based on type of transaction
func (b *Bitfinex) GetFee(feeBuilder exchange.FeeBuilder) (float64, error) {
	var fee float64

	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		accountInfos, err := b.GetAccountInformation()
		if err != nil {
			return 0, err
		}
		fee, err = b.CalculateTradingFee(accountInfos, feeBuilder.PurchasePrice, feeBuilder.Amount, feeBuilder.FirstCurrency, feeBuilder.IsMaker)
		if err != nil {
			return 0, err
		}
	case exchange.CyptocurrencyDepositFee:
		//TODO: fee is charged when < $1000USD is transferred, need to infer value in some way
		fee = 0
	case exchange.CryptocurrencyWithdrawalFee:
		accountFees, err := b.GetAccountFees()
		if err != nil {
			return 0, err
		}
		fee, err = b.GetCryptocurrencyWithdrawalFee(feeBuilder.FirstCurrency, accountFees)
		if err != nil {
			return 0, err
		}
	case exchange.InternationalBankDepositFee:
		fee = getInternationalBankDepositFee(feeBuilder.Amount)
	case exchange.InternationalBankWithdrawalFee:
		fee = getInternationalBankWithdrawalFee(feeBuilder.Amount)
	}
	if fee < 0 {
		fee = 0
	}
	return fee, nil
}

// GetCryptocurrencyWithdrawalFee returns an estimate of fee based on type of transaction
func (b *Bitfinex) GetCryptocurrencyWithdrawalFee(currency string, accountFees AccountFees) (fee float64, err error) {
	switch result := accountFees.Withdraw[currency].(type) {
	case string:
		fee, err = strconv.ParseFloat(result, 64)
		if err != nil {
			return 0, err
		}
	case float64:
		fee = result
	}

	return fee, nil
}

func getInternationalBankDepositFee(amount float64) float64 {
	return 0.001 * amount
}

func getInternationalBankWithdrawalFee(amount float64) float64 {
	return 0.001 * amount
}

// CalculateTradingFee returns an estimate of fee based on type of whether is maker or taker fee
func (b *Bitfinex) CalculateTradingFee(accountInfos []AccountInfo, purchasePrice, amount float64, currency string, isMaker bool) (fee float64, err error) {
	for _, i := range accountInfos {
		for _, j := range i.Fees {
			if currency == j.Pairs {
				if isMaker {
					fee = j.MakerFees
				} else {
					fee = j.TakerFees
				}
				break
			}
		}
		if fee > 0 {
			break
		}
	}
	return (fee / 100) * purchasePrice * amount, err
}

// ConvertSymbolToWithdrawalType You need to have specific withdrawal types to withdraw from Bitfinex
func (b *Bitfinex) ConvertSymbolToWithdrawalType(currency string) string {
	switch currency {
	case symbol.BTC:
		return "bitcoin"
	case symbol.LTC:
		return "litecoin"
	case symbol.ETH:
		return "ethereum"
	case symbol.ETC:
		return "ethereumc"
	case symbol.USDT:
		return "tetheruso"
	case "Wire":
		return "wire"
	case symbol.ZEC:
		return "zcash"
	case symbol.XMR:
		return "monero"
	case symbol.DSH:
		return "dash"
	case symbol.XRP:
		return "ripple"
	case symbol.SAN:
		return "santiment"
	case symbol.OMG:
		return "omisego"
	case symbol.BCH:
		return "bcash"
	case symbol.ETP:
		return "metaverse"
	case symbol.AVT:
		return "aventus"
	case symbol.EDO:
		return "eidoo"
	case symbol.BTG:
		return "bgold"
	case symbol.DATA:
		return "datacoin"
	case symbol.GNT:
		return "golem"
	case symbol.SNT:
		return "status"
	default:
		return common.StringToLower(currency)
	}
}

// ConvertSymbolToDepositMethod returns a converted currency deposit method
func (b *Bitfinex) ConvertSymbolToDepositMethod(currency string) (method string, err error) {
	switch currency {
	case symbol.BTC:
		method = "bitcoin"
	case symbol.LTC:
		method = "litecoin"
	case symbol.ETH:
		method = "ethereum"
	case symbol.ETC:
		method = "ethereumc"
	case symbol.USDT:
		method = "tetheruso"
	case symbol.ZEC:
		method = "zcash"
	case symbol.XMR:
		method = "monero"
	case symbol.BCH:
		method = "bcash"
	case symbol.MIOTA:
		method = "iota"
	default:
		err = fmt.Errorf("currency %s not supported in method list",
			currency)
	}
	return
}
