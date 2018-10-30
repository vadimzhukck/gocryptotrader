package okex

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/request"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-/gocryptotrader/logger"
)

const (
	// REST API information
	apiURL     = "https://www.okex.com/api/"
	apiVersion = "v1/"

	// Contract requests
	// Unauthenticated
	contractPrice            = "future_ticker"
	contractFutureDepth      = "future_depth"
	contractTradeHistory     = "future_trades"
	contractFutureIndex      = "future_index"
	contractExchangeRate     = "exchange_rate"
	contractFutureEstPrice   = "future_estimated_price"
	contractCandleStick      = "future_kline"
	contractFutureHoldAmount = "future_hold_amount"
	contractFutureLimits     = "future_price_limit"

	// Authenticated
	contractFutureUserInfo      = "future_userinfo"
	contractFuturePosition      = "future_position"
	contractFutureTrade         = "future_trade"
	contractFutureTradeHistory  = "future_trades_history"
	contractFutureBatchTrade    = "future_batch_trade"
	contractFutureCancel        = "future_cancel"
	contractFutureOrderInfo     = "future_order_info"
	contractFutureMultOrderInfo = "future_orders_info"
	contractFutureUserInfo4fix  = "future_userinfo_4fix"
	contractFuturePosition4fix  = "future_position_4fix"
	contractFutureExplosive     = "future_explosive"
	contractFutureDevolve       = "future_devolve"

	// Spot requests
	// Unauthenticated
	spotPrice   = "ticker"
	spotDepth   = "depth"
	spotTrades  = "trades"
	spotKline   = "kline"
	instruments = "instruments"

	// Authenticated
	spotUserInfo       = "userinfo"
	spotTrade          = "trade"
	spotBatchTrade     = "batch_trade"
	spotCancelTrade    = "cancel_order"
	spotOrderInfo      = "order_info.do"
	spotOrderHistory   = "order_history.do"
	spotMultiOrderInfo = "orders_info"
	spotWithdraw       = "withdraw.do"
	spotCancelWithdraw = "cancel_withdraw"
	spotWithdrawInfo   = "withdraw_info"
	spotAccountRecords = "account_records"

	myWalletInfo = "wallet_info.do"

	// just your average return type from okex
	returnTypeOne = "map[string]interface {}"

	okexAuthRate   = 0
	okexUnauthRate = 0
)

var errMissValue = errors.New("warning - resp value is missing from exchange")

// OKEX is the overaching type across the OKEX methods
type OKEX struct {
	exchange.Base
	WebsocketConn *websocket.Conn
	mu            sync.Mutex

	// Spot and contract market error codes as per https://www.okex.com/rest_request.html
	ErrorCodes map[string]error

	// Stores for corresponding variable checks
	ContractTypes    []string
	CurrencyPairs    []string
	ContractPosition []string
	Types            []string
}

// SetDefaults method assignes the default values for Bittrex
func (o *OKEX) SetDefaults() {
	o.SetErrorDefaults()
	o.SetCheckVarDefaults()
	o.Name = "OKEX"
	o.Enabled = false
	o.Verbose = false
	o.RESTPollingDelay = 10
	o.APIWithdrawPermissions = exchange.AutoWithdrawCrypto |
		exchange.NoFiatWithdrawals
	o.RequestCurrencyPairFormat.Delimiter = "_"
	o.RequestCurrencyPairFormat.Uppercase = false
	o.ConfigCurrencyPairFormat.Delimiter = "_"
	o.ConfigCurrencyPairFormat.Uppercase = true
	o.SupportsAutoPairUpdating = true
	o.SupportsRESTTickerBatching = false
	o.SupportsRESTAPI = true
	o.SupportsWebsocketAPI = true
	o.Requester = request.New(o.Name,
		request.NewRateLimit(time.Second, okexAuthRate),
		request.NewRateLimit(time.Second, okexUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	o.APIUrlDefault = apiURL
	o.APIUrl = o.APIUrlDefault
	o.AssetTypes = []string{ticker.Spot}
	o.WebsocketInit()
	o.Websocket.Functionality = exchange.WebsocketTickerSupported |
		exchange.WebsocketTradeDataSupported |
		exchange.WebsocketKlineSupported |
		exchange.WebsocketOrderbookSupported
}

// Setup method sets current configuration details if enabled
func (o *OKEX) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		o.SetEnabled(false)
	} else {
		o.Enabled = true
		o.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		o.SetAPIKeys(exch.APIKey, exch.APISecret, exch.ClientID, false)
		o.SetHTTPClientTimeout(exch.HTTPTimeout)
		o.SetHTTPClientUserAgent(exch.HTTPUserAgent)
		o.RESTPollingDelay = exch.RESTPollingDelay
		o.Verbose = exch.Verbose
		o.Websocket.SetEnabled(exch.Websocket)
		o.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		o.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		o.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")
		err := o.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = o.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
		err = o.SetAutoPairDefaults()
		if err != nil {
			log.Fatal(err)
		}
		err = o.SetAPIURL(exch)
		if err != nil {
			log.Fatal(err)
		}
		err = o.SetClientProxyAddress(exch.ProxyAddress)
		if err != nil {
			log.Fatal(err)
		}
		err = o.WebsocketSetup(o.WsConnect,
			exch.Name,
			exch.Websocket,
			okexDefaultWebsocketURL,
			exch.WebsocketURL)
		if err != nil {
			log.Fatal(err)
		}
	}
}

// GetSpotInstruments returns a list of tradable spot instruments and their properties
func (o *OKEX) GetSpotInstruments() ([]SpotInstrument, error) {
	var resp []SpotInstrument

	path := fmt.Sprintf("%sspot/v3/%s", o.APIUrl, instruments)
	err := o.SendHTTPRequest(path, &resp)

	if err != nil {
		return nil, err
	}

	return resp, nil
}

// GetContractPrice returns current contract prices
//
// symbol e.g. "btc_usd"
// contractType e.g. "this_week" "next_week" "quarter"
func (o *OKEX) GetContractPrice(symbol, contractType string) (ContractPrice, error) {
	resp := ContractPrice{}

	if err := o.CheckContractType(contractType); err != nil {
		return resp, err
	}
	if err := o.CheckSymbol(symbol); err != nil {
		return resp, err
	}

	values := url.Values{}
	values.Set("symbol", common.StringToLower(symbol))
	values.Set("contract_type", common.StringToLower(contractType))

	path := fmt.Sprintf("%s%s%s.do?%s", o.APIUrl, apiVersion, contractPrice, values.Encode())

	err := o.SendHTTPRequest(path, &resp)
	if err != nil {
		return resp, err
	}

	if !resp.Result {
		if resp.Error != nil {
			return resp, o.GetErrorCode(resp.Error)
		}
	}
	return resp, nil
}

// GetContractMarketDepth returns contract market depth
//
// symbol e.g. "btc_usd"
// contractType e.g. "this_week" "next_week" "quarter"
func (o *OKEX) GetContractMarketDepth(symbol, contractType string) (ActualContractDepth, error) {
	resp := ContractDepth{}
	fullDepth := ActualContractDepth{}

	if err := o.CheckContractType(contractType); err != nil {
		return fullDepth, err
	}
	if err := o.CheckSymbol(symbol); err != nil {
		return fullDepth, err
	}

	values := url.Values{}
	values.Set("symbol", common.StringToLower(symbol))
	values.Set("contract_type", common.StringToLower(contractType))

	path := fmt.Sprintf("%s%s%s.do?%s", o.APIUrl, apiVersion, contractFutureDepth, values.Encode())

	err := o.SendHTTPRequest(path, &resp)
	if err != nil {
		return fullDepth, err
	}

	if !resp.Result {
		if resp.Error != nil {
			return fullDepth, o.GetErrorCode(resp.Error)
		}
	}

	for _, ask := range resp.Asks {
		var askdepth struct {
			Price  float64
			Volume float64
		}
		for i, depth := range ask.([]interface{}) {
			if i == 0 {
				askdepth.Price = depth.(float64)
			}
			if i == 1 {
				askdepth.Volume = depth.(float64)
			}
		}
		fullDepth.Asks = append(fullDepth.Asks, askdepth)
	}

	for _, bid := range resp.Bids {
		var bidDepth struct {
			Price  float64
			Volume float64
		}
		for i, depth := range bid.([]interface{}) {
			if i == 0 {
				bidDepth.Price = depth.(float64)
			}
			if i == 1 {
				bidDepth.Volume = depth.(float64)
			}
		}
		fullDepth.Bids = append(fullDepth.Bids, bidDepth)
	}

	return fullDepth, nil
}

// GetContractTradeHistory returns trade history for the contract market
func (o *OKEX) GetContractTradeHistory(symbol, contractType string) ([]ActualContractTradeHistory, error) {
	actualTradeHistory := []ActualContractTradeHistory{}
	var resp interface{}

	if err := o.CheckContractType(contractType); err != nil {
		return actualTradeHistory, err
	}
	if err := o.CheckSymbol(symbol); err != nil {
		return actualTradeHistory, err
	}

	values := url.Values{}
	values.Set("symbol", common.StringToLower(symbol))
	values.Set("contract_type", common.StringToLower(contractType))

	path := fmt.Sprintf("%s%s%s.do?%s", o.APIUrl, apiVersion, contractTradeHistory, values.Encode())

	err := o.SendHTTPRequest(path, &resp)
	if err != nil {
		return actualTradeHistory, err
	}

	if reflect.TypeOf(resp).String() == returnTypeOne {
		errorMap := resp.(map[string]interface{})
		return actualTradeHistory, o.GetErrorCode(errorMap["error_code"].(float64))
	}

	for _, tradeHistory := range resp.([]interface{}) {
		quickHistory := ActualContractTradeHistory{}
		tradeHistoryM := tradeHistory.(map[string]interface{})
		quickHistory.Date = tradeHistoryM["date"].(float64)
		quickHistory.DateInMS = tradeHistoryM["date_ms"].(float64)
		quickHistory.Amount = tradeHistoryM["amount"].(float64)
		quickHistory.Price = tradeHistoryM["price"].(float64)
		quickHistory.Type = tradeHistoryM["type"].(string)
		quickHistory.TID = tradeHistoryM["tid"].(float64)
		actualTradeHistory = append(actualTradeHistory, quickHistory)
	}
	return actualTradeHistory, nil
}

// GetContractIndexPrice returns the current index price
//
// symbol e.g. btc_usd
func (o *OKEX) GetContractIndexPrice(symbol string) (float64, error) {
	if err := o.CheckSymbol(symbol); err != nil {
		return 0, err
	}

	values := url.Values{}
	values.Set("symbol", common.StringToLower(symbol))
	path := fmt.Sprintf("%s%s%s.do?%s", o.APIUrl, apiVersion, contractFutureIndex, values.Encode())
	var resp interface{}

	err := o.SendHTTPRequest(path, &resp)
	if err != nil {
		return 0, err
	}

	futureIndex := resp.(map[string]interface{})
	if i, ok := futureIndex["error_code"].(float64); ok {
		return 0, o.GetErrorCode(i)
	}

	if _, ok := futureIndex["future_index"].(float64); ok {
		return futureIndex["future_index"].(float64), nil
	}
	return 0, errMissValue
}

// GetContractExchangeRate returns the current exchange rate for the currency
// pair
// USD-CNY exchange rate used by OKEX, updated weekly
func (o *OKEX) GetContractExchangeRate() (float64, error) {
	path := fmt.Sprintf("%s%s%s.do?", o.APIUrl, apiVersion, contractExchangeRate)
	var resp interface{}

	if err := o.SendHTTPRequest(path, &resp); err != nil {
		return 0, err
	}

	exchangeRate := resp.(map[string]interface{})
	if i, ok := exchangeRate["error_code"].(float64); ok {
		return 0, o.GetErrorCode(i)
	}

	if _, ok := exchangeRate["rate"].(float64); ok {
		return exchangeRate["rate"].(float64), nil
	}
	return 0, errMissValue
}

// GetContractFutureEstimatedPrice returns futures estimated price
//
// symbol e.g btc_usd
func (o *OKEX) GetContractFutureEstimatedPrice(symbol string) (float64, error) {
	if err := o.CheckSymbol(symbol); err != nil {
		return 0, err
	}

	values := url.Values{}
	values.Set("symbol", symbol)
	path := fmt.Sprintf("%s%s%s.do?%s", o.APIUrl, apiVersion, contractFutureIndex, values.Encode())
	var resp interface{}

	if err := o.SendHTTPRequest(path, &resp); err != nil {
		return 0, err
	}

	futuresEstPrice := resp.(map[string]interface{})
	if i, ok := futuresEstPrice["error_code"].(float64); ok {
		return 0, o.GetErrorCode(i)
	}

	if _, ok := futuresEstPrice["future_index"].(float64); ok {
		return futuresEstPrice["future_index"].(float64), nil
	}
	return 0, errMissValue
}

// GetContractCandlestickData returns CandleStickData
//
// symbol e.g. btc_usd
// type e.g. 1min or 1 minute candlestick data
// contract_type e.g. this_week
// size: specify data size to be acquired
// since: timestamp(eg:1417536000000). data after the timestamp will be returned
func (o *OKEX) GetContractCandlestickData(symbol, typeInput, contractType string, size, since int) ([]CandleStickData, error) {
	var candleData []CandleStickData
	if err := o.CheckSymbol(symbol); err != nil {
		return candleData, err
	}
	if err := o.CheckContractType(contractType); err != nil {
		return candleData, err
	}
	if err := o.CheckType(typeInput); err != nil {
		return candleData, err
	}

	values := url.Values{}
	values.Set("symbol", symbol)
	values.Set("type", typeInput)
	values.Set("contract_type", contractType)
	values.Set("size", strconv.FormatInt(int64(size), 10))
	values.Set("since", strconv.FormatInt(int64(since), 10))

	path := fmt.Sprintf("%s%s%s.do?%s", o.APIUrl, apiVersion, contractCandleStick, values.Encode())
	var resp interface{}

	if err := o.SendHTTPRequest(path, &resp); err != nil {
		return candleData, err
	}

	if reflect.TypeOf(resp).String() == returnTypeOne {
		errorMap := resp.(map[string]interface{})
		return candleData, o.GetErrorCode(errorMap["error_code"].(float64))
	}

	for _, candleStickData := range resp.([]interface{}) {
		var quickCandle CandleStickData

		for i, datum := range candleStickData.([]interface{}) {
			switch i {
			case 0:
				quickCandle.Timestamp = datum.(float64)
			case 1:
				quickCandle.Open = datum.(float64)
			case 2:
				quickCandle.High = datum.(float64)
			case 3:
				quickCandle.Low = datum.(float64)
			case 4:
				quickCandle.Close = datum.(float64)
			case 5:
				quickCandle.Volume = datum.(float64)
			case 6:
				quickCandle.Amount = datum.(float64)
			default:
				return candleData, errors.New("incoming data out of range")
			}
		}
		candleData = append(candleData, quickCandle)
	}

	return candleData, nil
}

// GetContractHoldingsNumber returns current number of holdings
func (o *OKEX) GetContractHoldingsNumber(symbol, contractType string) (number float64, contract string, err error) {
	err = o.CheckSymbol(symbol)
	if err != nil {
		return number, contract, err
	}

	err = o.CheckContractType(contractType)
	if err != nil {
		return number, contract, err
	}

	values := url.Values{}
	values.Set("symbol", symbol)
	values.Set("contract_type", contractType)

	path := fmt.Sprintf("%s%s%s.do?%s", o.APIUrl, apiVersion, contractFutureHoldAmount, values.Encode())
	var resp interface{}

	err = o.SendHTTPRequest(path, &resp)
	if err != nil {
		return number, contract, err
	}

	if reflect.TypeOf(resp).String() == returnTypeOne {
		errorMap := resp.(map[string]interface{})
		return number, contract, o.GetErrorCode(errorMap["error_code"].(float64))
	}

	for _, holdings := range resp.([]interface{}) {
		if reflect.TypeOf(holdings).String() == returnTypeOne {
			holdingMap := holdings.(map[string]interface{})
			number = holdingMap["amount"].(float64)
			contract = holdingMap["contract_name"].(string)
		}
	}
	return number, contract, err
}

// GetContractlimit returns upper and lower price limit
func (o *OKEX) GetContractlimit(symbol, contractType string) (map[string]float64, error) {
	contractLimits := make(map[string]float64)
	if err := o.CheckSymbol(symbol); err != nil {
		return contractLimits, err
	}
	if err := o.CheckContractType(contractType); err != nil {
		return contractLimits, err
	}

	values := url.Values{}
	values.Set("symbol", symbol)
	values.Set("contract_type", contractType)

	path := fmt.Sprintf("%s%s%s.do?%s", o.APIUrl, apiVersion, contractFutureLimits, values.Encode())
	var resp interface{}

	if err := o.SendHTTPRequest(path, &resp); err != nil {
		return contractLimits, err
	}

	contractLimitMap := resp.(map[string]interface{})
	if i, ok := contractLimitMap["error_code"].(float64); ok {
		return contractLimits, o.GetErrorCode(i)
	}

	contractLimits["high"] = contractLimitMap["high"].(float64)
	contractLimits["usdCnyRate"] = contractLimitMap["usdCnyRate"].(float64)
	contractLimits["low"] = contractLimitMap["low"].(float64)
	return contractLimits, nil
}

// GetContractUserInfo returns OKEX Contract Account Info（Cross-Margin Mode）
func (o *OKEX) GetContractUserInfo() error {
	var resp interface{}
	if err := o.SendAuthenticatedHTTPRequest(contractFutureUserInfo, url.Values{}, &resp); err != nil {
		return err
	}

	userInfoMap := resp.(map[string]interface{})
	if code, ok := userInfoMap["error_code"]; ok {
		return o.GetErrorCode(code)
	}
	return nil
}

// GetContractPosition returns User Contract Positions （Cross-Margin Mode）
func (o *OKEX) GetContractPosition(symbol, contractType string) error {
	var resp interface{}

	if err := o.CheckSymbol(symbol); err != nil {
		return err
	}
	if err := o.CheckContractType(contractType); err != nil {
		return err
	}

	values := url.Values{}
	values.Set("symbol", symbol)
	values.Set("contract_type", contractType)

	if err := o.SendAuthenticatedHTTPRequest(contractFuturePosition, values, &resp); err != nil {
		return err
	}

	userInfoMap := resp.(map[string]interface{})
	if code, ok := userInfoMap["error_code"]; ok {
		return o.GetErrorCode(code)
	}
	return nil
}

// PlaceContractOrders places orders
func (o *OKEX) PlaceContractOrders(symbol, contractType, position string, leverageRate int, price, amount float64, matchPrice bool) (float64, error) {
	var resp interface{}

	if err := o.CheckSymbol(symbol); err != nil {
		return 0, err
	}
	if err := o.CheckContractType(contractType); err != nil {
		return 0, err
	}
	if err := o.CheckContractPosition(position); err != nil {
		return 0, err
	}

	values := url.Values{}
	values.Set("symbol", symbol)
	values.Set("contract_type", contractType)
	values.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	values.Set("type", position)
	if matchPrice {
		values.Set("match_price", "1")
	} else {
		values.Set("match_price", "0")
	}

	if leverageRate != 10 && leverageRate != 20 {
		return 0, errors.New("leverage rate can only be 10 or 20")
	}
	values.Set("lever_rate", strconv.FormatInt(int64(leverageRate), 10))

	if err := o.SendAuthenticatedHTTPRequest(contractFutureTrade, values, &resp); err != nil {
		return 0, err
	}

	contractMap := resp.(map[string]interface{})
	if code, ok := contractMap["error_code"]; ok {
		return 0, o.GetErrorCode(code)
	}

	if orderID, ok := contractMap["order_id"]; ok {
		return orderID.(float64), nil
	}

	return 0, errors.New("orderID returned nil")
}

// GetContractFuturesTradeHistory returns OKEX Contract Trade History (Not for Personal)
func (o *OKEX) GetContractFuturesTradeHistory(symbol, date string, since int) error {
	var resp interface{}

	if err := o.CheckSymbol(symbol); err != nil {
		return err
	}

	values := url.Values{}
	values.Set("symbol", symbol)
	values.Set("date", date)
	values.Set("since", strconv.FormatInt(int64(since), 10))

	if err := o.SendAuthenticatedHTTPRequest(contractFutureTradeHistory, values, &resp); err != nil {
		return err
	}

	respMap := resp.(map[string]interface{})
	if code, ok := respMap["error_code"]; ok {
		return o.GetErrorCode(code)
	}
	return nil
}

// GetTokenOrders returns details for a single orderID or all open orders when orderID == -1
func (o *OKEX) GetTokenOrders(symbol string, orderID int64) (TokenOrdersResponse, error) {
	var resp TokenOrdersResponse
	values := url.Values{}
	values.Set("symbol", symbol)
	values.Set("order_id", strconv.FormatInt(orderID, 10))
	return resp, o.SendAuthenticatedHTTPRequest(contractFutureTradeHistory, values, &resp)
}

// GetUserInfo returns the user info
func (o *OKEX) GetUserInfo() (SpotUserInfo, error) {
	var resp SpotUserInfo
	return resp, o.SendAuthenticatedHTTPRequest(spotUserInfo, url.Values{}, &resp)
}

// SpotNewOrder creates a new spot order
func (o *OKEX) SpotNewOrder(arg SpotNewOrderRequestParams) (int64, error) {
	type response struct {
		Result  bool  `json:"result"`
		OrderID int64 `json:"order_id"`
	}

	var res response
	params := url.Values{}
	params.Set("symbol", arg.Symbol)
	params.Set("type", string(arg.Type))
	params.Set("price", strconv.FormatFloat(arg.Price, 'f', -1, 64))
	params.Set("amount", strconv.FormatFloat(arg.Amount, 'f', -1, 64))

	err := o.SendAuthenticatedHTTPRequest(spotTrade, params, &res)
	if err != nil {
		return res.OrderID, err
	}

	return res.OrderID, nil
}

// SpotCancelOrder cancels a spot order
// symbol such as ltc_btc
// orderID orderID
// returns orderID or an error
func (o *OKEX) SpotCancelOrder(symbol string, argOrderID int64) (int64, error) {
	var res = struct {
		Result    bool   `json:"result"`
		OrderID   string `json:"order_id"`
		ErrorCode int    `json:"error_code"`
	}{}

	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("order_id", strconv.FormatInt(argOrderID, 10))
	var returnOrderID int64

	err := o.SendAuthenticatedHTTPRequest(spotCancelTrade+".do", params, &res)
	if err != nil {
		return returnOrderID, err
	}

	if res.ErrorCode != 0 {
		return returnOrderID, fmt.Errorf("failed to cancel order. code: %d err: %s",
			res.ErrorCode,
			o.ErrorCodes[strconv.Itoa(res.ErrorCode)],
		)
	}

	returnOrderID, _ = common.Int64FromString(res.OrderID)
	return returnOrderID, nil
}

// GetLatestSpotPrice returns latest spot price of symbol
//
// symbol: string of currency pair
func (o *OKEX) GetLatestSpotPrice(symbol string) (float64, error) {
	spotPrice, err := o.GetSpotTicker(symbol)

	if err != nil {
		return 0, err
	}

	return spotPrice.Ticker.Last, nil
}

// GetSpotTicker returns Price Ticker
func (o *OKEX) GetSpotTicker(symbol string) (SpotPrice, error) {
	var resp SpotPrice

	values := url.Values{}
	values.Set("symbol", symbol)
	path := fmt.Sprintf("%s%s%s.do?%s", o.APIUrl, apiVersion, "ticker", values.Encode())

	err := o.SendHTTPRequest(path, &resp)
	if err != nil {
		return resp, err
	}

	if resp.Error != nil {
		return resp, o.GetErrorCode(resp.Error.(float64))
	}
	return resp, nil
}

// GetSpotMarketDepth returns Market Depth
func (o *OKEX) GetSpotMarketDepth(asd ActualSpotDepthRequestParams) (ActualSpotDepth, error) {
	resp := SpotDepth{}
	fullDepth := ActualSpotDepth{}

	values := url.Values{}
	values.Set("symbol", asd.Symbol)
	values.Set("size", fmt.Sprintf("%d", asd.Size))

	path := fmt.Sprintf("%s%s%s.do?%s", o.APIUrl, apiVersion, "depth", values.Encode())

	err := o.SendHTTPRequest(path, &resp)
	if err != nil {
		return fullDepth, err
	}

	if !resp.Result {
		if resp.Error != nil {
			return fullDepth, o.GetErrorCode(resp.Error)
		}
	}

	for _, ask := range resp.Asks {
		var askdepth struct {
			Price  float64
			Volume float64
		}
		for i, depth := range ask.([]interface{}) {
			if i == 0 {
				askdepth.Price = depth.(float64)
			}
			if i == 1 {
				askdepth.Volume = depth.(float64)
			}
		}
		fullDepth.Asks = append(fullDepth.Asks, askdepth)
	}

	for _, bid := range resp.Bids {
		var bidDepth struct {
			Price  float64
			Volume float64
		}
		for i, depth := range bid.([]interface{}) {
			if i == 0 {
				bidDepth.Price = depth.(float64)
			}
			if i == 1 {
				bidDepth.Volume = depth.(float64)
			}
		}
		fullDepth.Bids = append(fullDepth.Bids, bidDepth)
	}

	return fullDepth, nil
}

// GetSpotRecentTrades returns recent trades
func (o *OKEX) GetSpotRecentTrades(ast ActualSpotTradeHistoryRequestParams) ([]ActualSpotTradeHistory, error) {
	actualTradeHistory := []ActualSpotTradeHistory{}
	var resp interface{}

	values := url.Values{}
	values.Set("symbol", ast.Symbol)
	values.Set("since", fmt.Sprintf("%d", ast.Since))

	path := fmt.Sprintf("%s%s%s.do?%s", o.APIUrl, apiVersion, "trades", values.Encode())

	err := o.SendHTTPRequest(path, &resp)
	if err != nil {
		return actualTradeHistory, err
	}

	if reflect.TypeOf(resp).String() == returnTypeOne {
		errorMap := resp.(map[string]interface{})
		return actualTradeHistory, o.GetErrorCode(errorMap["error_code"].(float64))
	}

	for _, tradeHistory := range resp.([]interface{}) {
		quickHistory := ActualSpotTradeHistory{}
		tradeHistoryM := tradeHistory.(map[string]interface{})
		quickHistory.Date = tradeHistoryM["date"].(float64)
		quickHistory.DateInMS = tradeHistoryM["date_ms"].(float64)
		quickHistory.Amount = tradeHistoryM["amount"].(float64)
		quickHistory.Price = tradeHistoryM["price"].(float64)
		quickHistory.Type = tradeHistoryM["type"].(string)
		quickHistory.TID = tradeHistoryM["tid"].(float64)
		actualTradeHistory = append(actualTradeHistory, quickHistory)
	}
	return actualTradeHistory, nil
}

// GetSpotKline returns candlestick data
func (o *OKEX) GetSpotKline(arg KlinesRequestParams) ([]CandleStickData, error) {
	var candleData []CandleStickData

	values := url.Values{}
	values.Set("symbol", arg.Symbol)
	values.Set("type", string(arg.Type))
	if arg.Size != 0 {
		values.Set("size", strconv.FormatInt(int64(arg.Size), 10))
	}
	if arg.Since != 0 {
		values.Set("since", strconv.FormatInt(arg.Since, 10))
	}

	path := fmt.Sprintf("%s%s%s.do?%s", o.APIUrl, apiVersion, spotKline, values.Encode())
	var resp interface{}

	if err := o.SendHTTPRequest(path, &resp); err != nil {
		return candleData, err
	}

	if reflect.TypeOf(resp).String() == returnTypeOne {
		errorMap := resp.(map[string]interface{})
		return candleData, o.GetErrorCode(errorMap["error_code"].(float64))
	}

	for _, candleStickData := range resp.([]interface{}) {
		var quickCandle CandleStickData

		for i, datum := range candleStickData.([]interface{}) {
			switch i {
			case 0:
				quickCandle.Timestamp = datum.(float64)
			case 1:
				quickCandle.Open, _ = strconv.ParseFloat(datum.(string), 64)
			case 2:
				quickCandle.High, _ = strconv.ParseFloat(datum.(string), 64)
			case 3:
				quickCandle.Low, _ = strconv.ParseFloat(datum.(string), 64)
			case 4:
				quickCandle.Close, _ = strconv.ParseFloat(datum.(string), 64)
			case 5:
				quickCandle.Volume, _ = strconv.ParseFloat(datum.(string), 64)
			case 6:
				quickCandle.Amount, _ = strconv.ParseFloat(datum.(string), 64)
			default:
				return candleData, errors.New("incoming data out of range")
			}
		}
		candleData = append(candleData, quickCandle)
	}

	return candleData, nil
}

// GetErrorCode finds the associated error code and returns its corresponding
// string
func (o *OKEX) GetErrorCode(code interface{}) error {
	var assertedCode string

	switch reflect.TypeOf(code).String() {
	case "float64":
		assertedCode = strconv.FormatFloat(code.(float64), 'f', -1, 64)
	case "string":
		assertedCode = code.(string)
	default:
		return errors.New("unusual type returned")
	}

	if i, ok := o.ErrorCodes[assertedCode]; ok {
		return i
	}
	return errors.New("unable to find SPOT error code")
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (o *OKEX) SendHTTPRequest(path string, result interface{}) error {
	return o.SendPayload(http.MethodGet, path, nil, nil, result, false, o.Verbose)
}

// SendAuthenticatedHTTPRequest sends an authenticated http request to a desired
// path
func (o *OKEX) SendAuthenticatedHTTPRequest(method string, values url.Values, result interface{}) (err error) {
	if !o.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, o.Name)
	}

	values.Set("api_key", o.APIKey)
	hasher := common.GetMD5([]byte(values.Encode() + "&secret_key=" + o.APISecret))
	values.Set("sign", strings.ToUpper(common.HexEncodeToString(hasher)))

	encoded := values.Encode()
	path := o.APIUrl + apiVersion + method

	if o.Verbose {
		log.Debugf("Sending POST request to %s with params %s\n", path, encoded)
	}

	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	var intermediary json.RawMessage

	errCap := struct {
		Result bool  `json:"result"`
		Error  int64 `json:"error_code"`
	}{}

	err = o.SendPayload(http.MethodPost, path, headers, strings.NewReader(encoded), &intermediary, true, o.Verbose)
	if err != nil {
		return err
	}

	err = common.JSONDecode(intermediary, &errCap)
	if err == nil {
		if !errCap.Result {
			return fmt.Errorf("sendAuthenticatedHTTPRequest error - %s",
				o.ErrorCodes[strconv.FormatInt(errCap.Error, 10)])
		}
	}

	return common.JSONDecode(intermediary, result)
}

// SetErrorDefaults sets the full error default list
func (o *OKEX) SetErrorDefaults() {
	o.ErrorCodes = map[string]error{
		// Spot Errors
		"10000": errors.New("required field, can not be null"),
		"10001": errors.New("request frequency too high to exceed the limit allowed"),
		"10002": errors.New("system error"),
		"10004": errors.New("request failed - Your API key might need to be recreated"),
		"10005": errors.New("'secretKey' does not exist"),
		"10006": errors.New("'api_key' does not exist"),
		"10007": errors.New("signature does not match"),
		"10008": errors.New("illegal parameter"),
		"10009": errors.New("order does not exist"),
		"10010": errors.New("insufficient funds"),
		"10011": errors.New("amount too low"),
		"10012": errors.New("only btc_usd ltc_usd supported"),
		"10013": errors.New("only support https request"),
		"10014": errors.New("order price must be between 0 and 1,000,000"),
		"10015": errors.New("order price differs from current market price too much"),
		"10016": errors.New("insufficient coins balance"),
		"10017": errors.New("api authorization error"),
		"10018": errors.New("borrow amount less than lower limit [usd:100,btc:0.1,ltc:1]"),
		"10019": errors.New("loan agreement not checked"),
		"10020": errors.New("rate cannot exceed 1%"),
		"10021": errors.New("rate cannot less than 0.01%"),
		"10023": errors.New("fail to get latest ticker"),
		"10024": errors.New("balance not sufficient"),
		"10025": errors.New("quota is full, cannot borrow temporarily"),
		"10026": errors.New("loan (including reserved loan) and margin cannot be withdrawn"),
		"10027": errors.New("cannot withdraw within 24 hrs of authentication information modification"),
		"10028": errors.New("withdrawal amount exceeds daily limit"),
		"10029": errors.New("account has unpaid loan, please cancel/pay off the loan before withdraw"),
		"10031": errors.New("deposits can only be withdrawn after 6 confirmations"),
		"10032": errors.New("please enabled phone/google authenticator"),
		"10033": errors.New("fee higher than maximum network transaction fee"),
		"10034": errors.New("fee lower than minimum network transaction fee"),
		"10035": errors.New("insufficient BTC/LTC"),
		"10036": errors.New("withdrawal amount too low"),
		"10037": errors.New("trade password not set"),
		"10040": errors.New("withdrawal cancellation fails"),
		"10041": errors.New("withdrawal address not exsit or approved"),
		"10042": errors.New("admin password error"),
		"10043": errors.New("account equity error, withdrawal failure"),
		"10044": errors.New("fail to cancel borrowing order"),
		"10047": errors.New("this function is disabled for sub-account"),
		"10048": errors.New("withdrawal information does not exist"),
		"10049": errors.New("user can not have more than 50 unfilled small orders (amount<0.15BTC)"),
		"10050": errors.New("can't cancel more than once"),
		"10051": errors.New("order completed transaction"),
		"10052": errors.New("not allowed to withdraw"),
		"10064": errors.New("after a USD deposit, that portion of assets will not be withdrawable for the next 48 hours"),
		"10100": errors.New("user account frozen"),
		"10101": errors.New("order type is wrong"),
		"10102": errors.New("incorrect ID"),
		"10103": errors.New("the private otc order's key incorrect"),
		"10216": errors.New("non-available API"),
		"1002":  errors.New("the transaction amount exceed the balance"),
		"1003":  errors.New("the transaction amount is less than the minimum requirement"),
		"1004":  errors.New("the transaction amount is less than 0"),
		"1007":  errors.New("no trading market information"),
		"1008":  errors.New("no latest market information"),
		"1009":  errors.New("no order"),
		"1010":  errors.New("different user of the cancelled order and the original order"),
		"1011":  errors.New("no documented user"),
		"1013":  errors.New("no order type"),
		"1014":  errors.New("no login"),
		"1015":  errors.New("no market depth information"),
		"1017":  errors.New("date error"),
		"1018":  errors.New("order failed"),
		"1019":  errors.New("undo order failed"),
		"1024":  errors.New("currency does not exist"),
		"1025":  errors.New("no chart type"),
		"1026":  errors.New("no base currency quantity"),
		"1027":  errors.New("incorrect parameter may exceeded limits"),
		"1028":  errors.New("reserved decimal failed"),
		"1029":  errors.New("preparing"),
		"1030":  errors.New("account has margin and futures, transactions can not be processed"),
		"1031":  errors.New("insufficient Transferring Balance"),
		"1032":  errors.New("transferring Not Allowed"),
		"1035":  errors.New("password incorrect"),
		"1036":  errors.New("google Verification code Invalid"),
		"1037":  errors.New("google Verification code incorrect"),
		"1038":  errors.New("google Verification replicated"),
		"1039":  errors.New("message Verification Input exceed the limit"),
		"1040":  errors.New("message Verification invalid"),
		"1041":  errors.New("message Verification incorrect"),
		"1042":  errors.New("wrong Google Verification Input exceed the limit"),
		"1043":  errors.New("login password cannot be same as the trading password"),
		"1044":  errors.New("old password incorrect"),
		"1045":  errors.New("2nd Verification Needed"),
		"1046":  errors.New("please input old password"),
		"1048":  errors.New("account Blocked"),
		"1201":  errors.New("account Deleted at 00: 00"),
		"1202":  errors.New("account Not Exist"),
		"1203":  errors.New("insufficient Balance"),
		"1204":  errors.New("invalid currency"),
		"1205":  errors.New("invalid Account"),
		"1206":  errors.New("cash Withdrawal Blocked"),
		"1207":  errors.New("transfer Not Support"),
		"1208":  errors.New("no designated account"),
		"1209":  errors.New("invalid api"),
		"1216":  errors.New("market order temporarily suspended. Please send limit order"),
		"1217":  errors.New("order was sent at ±5% of the current market price. Please resend"),
		"1218":  errors.New("place order failed. Please try again later"),
		// Errors for both
		"HTTP ERROR CODE 403": errors.New("too many requests, IP is shielded"),
		"Request Timed Out":   errors.New("too many requests, IP is shielded"),
		// contract errors
		"405":   errors.New("method not allowed"),
		"20001": errors.New("user does not exist"),
		"20002": errors.New("account frozen"),
		"20003": errors.New("account frozen due to liquidation"),
		"20004": errors.New("contract account frozen"),
		"20005": errors.New("user contract account does not exist"),
		"20006": errors.New("required field missing"),
		"20007": errors.New("illegal parameter"),
		"20008": errors.New("contract account balance is too low"),
		"20009": errors.New("contract status error"),
		"20010": errors.New("risk rate ratio does not exist"),
		"20011": errors.New("risk rate lower than 90%/80% before opening BTC position with 10x/20x leverage. or risk rate lower than 80%/60% before opening LTC position with 10x/20x leverage"),
		"20012": errors.New("risk rate lower than 90%/80% after opening BTC position with 10x/20x leverage. or risk rate lower than 80%/60% after opening LTC position with 10x/20x leverage"),
		"20013": errors.New("temporally no counter party price"),
		"20014": errors.New("system error"),
		"20015": errors.New("order does not exist"),
		"20016": errors.New("close amount bigger than your open positions"),
		"20017": errors.New("not authorized/illegal operation"),
		"20018": errors.New("order price cannot be more than 103% or less than 97% of the previous minute price"),
		"20019": errors.New("ip restricted from accessing the resource"),
		"20020": errors.New("secretKey does not exist"),
		"20021": errors.New("index information does not exist"),
		"20022": errors.New("wrong API interface (Cross margin mode shall call cross margin API, fixed margin mode shall call fixed margin API)"),
		"20023": errors.New("account in fixed-margin mode"),
		"20024": errors.New("signature does not match"),
		"20025": errors.New("leverage rate error"),
		"20026": errors.New("api permission error"),
		"20027": errors.New("no transaction record"),
		"20028": errors.New("no such contract"),
		"20029": errors.New("amount is large than available funds"),
		"20030": errors.New("account still has debts"),
		"20038": errors.New("due to regulation, this function is not available in the country/region your currently reside in"),
		"20049": errors.New("request frequency too high"),
	}
}

// SetCheckVarDefaults sets main variables that will be used in requests because
// api does not return an error if there are misspellings in strings. So better
// to check on this, this end.
func (o *OKEX) SetCheckVarDefaults() {
	o.ContractTypes = []string{"this_week", "next_week", "quarter"}
	o.CurrencyPairs = []string{"btc_usd", "ltc_usd", "eth_usd", "etc_usd", "bch_usd"}
	o.Types = []string{"1min", "3min", "5min", "15min", "30min", "1day", "3day",
		"1week", "1hour", "2hour", "4hour", "6hour", "12hour"}
	o.ContractPosition = []string{"1", "2", "3", "4"}
}

// CheckContractPosition checks to see if the string is a valid position for okex
func (o *OKEX) CheckContractPosition(position string) error {
	if !common.StringDataCompare(o.ContractPosition, position) {
		return errors.New("invalid position string - e.g. 1 = open long position, 2 = open short position, 3 = liquidate long position, 4 = liquidate short position")
	}
	return nil
}

// CheckSymbol checks to see if the string is a valid symbol for okex
func (o *OKEX) CheckSymbol(symbol string) error {
	if !common.StringDataCompare(o.CurrencyPairs, symbol) {
		return errors.New("invalid symbol string")
	}
	return nil
}

// CheckContractType checks to see if the string is a correct asset
func (o *OKEX) CheckContractType(contractType string) error {
	if !common.StringDataCompare(o.ContractTypes, contractType) {
		return errors.New("invalid contract type string")
	}
	return nil
}

// CheckType checks to see if the string is a correct type
func (o *OKEX) CheckType(typeInput string) error {
	if !common.StringDataCompare(o.Types, typeInput) {
		return errors.New("invalid type string")
	}
	return nil
}

// GetFee returns an estimate of fee based on type of transaction
func (o *OKEX) GetFee(feeBuilder exchange.FeeBuilder) (float64, error) {
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		fee = calculateTradingFee(feeBuilder.PurchasePrice, feeBuilder.Amount, feeBuilder.IsMaker)
	case exchange.CryptocurrencyWithdrawalFee:
		fee = getWithdrawalFee(feeBuilder.FirstCurrency)
	}
	if fee < 0 {
		fee = 0
	}

	return fee, nil
}

func calculateTradingFee(purchasePrice, amount float64, isMaker bool) (fee float64) {
	// TODO volume based fees
	if isMaker {
		fee = 0.001
	} else {
		fee = 0.0015
	}
	return fee * amount * purchasePrice
}

func getWithdrawalFee(currency string) float64 {
	return WithdrawalFees[currency]
}

// GetBalance returns the full balance across all wallets
func (o *OKEX) GetBalance() ([]FullBalance, error) {
	var resp Balance
	var balances []FullBalance

	err := o.SendAuthenticatedHTTPRequest(myWalletInfo, url.Values{}, &resp)
	if err != nil {
		return balances, err
	}

	for key, available := range resp.Info.Funds.Free {
		free, err := strconv.ParseFloat(available, 64)
		if err != nil {
			return balances, err
		}

		inUse, ok := resp.Info.Funds.Holds[key]
		if !ok {
			return balances, fmt.Errorf("hold currency %s not found in map", key)
		}

		hold, err := strconv.ParseFloat(inUse, 64)
		if err != nil {
			return balances, err
		}

		balances = append(balances, FullBalance{
			Currency:  key,
			Available: free,
			Hold:      hold,
		})
	}

	return balances, nil
}

// Withdrawal withdraws a cryptocurrency to a supplied address
func (o *OKEX) Withdrawal(symbol string, fee float64, tradePWD, address string, amount float64) (int, error) {
	v := url.Values{}
	v.Set("symbol", symbol)

	if fee != 0 {
		v.Set("chargefee", strconv.FormatFloat(fee, 'f', -1, 64))
	}
	v.Set("trade_pwd", tradePWD)
	v.Set("withdraw_address", address)
	v.Set("withdraw_amount", strconv.FormatFloat(amount, 'f', -1, 64))
	v.Set("target", "address")
	resp := WithdrawalResponse{}

	err := o.SendAuthenticatedHTTPRequest(spotWithdraw, v, &resp)
	if err != nil {
		return 0, err
	}

	if !resp.Result {
		return 0, errors.New("unable to process withdrawal request")
	}

	return resp.WithdrawID, nil
}

// GetOrderInformation withdraws a cryptocurrency to a supplied address
func (o *OKEX) GetOrderInformation(orderID int64, symbol string) ([]OrderInfo, error) {
	type Response struct {
		Result bool        `json:"result"`
		Orders []OrderInfo `json:"orders"`
	}

	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("order_id", strconv.FormatInt(orderID, 10))
	result := Response{}

	err := o.SendAuthenticatedHTTPRequest(spotOrderInfo, v, &result)
	if err != nil {
		return nil, err
	}

	if !result.Result {
		return nil, errors.New("unable to retrieve order info")
	}

	return result.Orders, nil
}

// GetOrderHistoryForCurrency returns a history of orders
func (o *OKEX) GetOrderHistoryForCurrency(pageLength, currentPage, status int64, symbol string) (OrderHistory, error) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("status", strconv.FormatInt(status, 10))
	v.Set("current_page", strconv.FormatInt(currentPage, 10))
	v.Set("page_length", strconv.FormatInt(pageLength, 10))
	result := OrderHistory{}
	return result, o.SendAuthenticatedHTTPRequest(spotOrderHistory, v, &result)
}
