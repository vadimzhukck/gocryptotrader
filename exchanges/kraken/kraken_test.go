package kraken

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/currency/symbol"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
)

var k Kraken

// Please add your own APIkeys to do correct due diligence testing.
const (
	apiKey                  = ""
	apiSecret               = ""
	clientID                = ""
	canManipulateRealOrders = false
)

func TestSetDefaults(t *testing.T) {
	k.SetDefaults()
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	krakenConfig, err := cfg.GetExchangeConfig("Kraken")
	if err != nil {
		t.Error("Test Failed - kraken Setup() init error", err)
	}
	krakenConfig.API.AuthenticatedSupport = true
	krakenConfig.API.Credentials.Key = apiKey
	krakenConfig.API.Credentials.Secret = apiSecret
	krakenConfig.API.Credentials.ClientID = clientID

	k.Setup(krakenConfig)
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	_, err := k.GetServerTime()
	if err != nil {
		t.Error("Test Failed - GetServerTime() error", err)
	}
}

func TestGetAssets(t *testing.T) {
	t.Parallel()
	_, err := k.GetAssets()
	if err != nil {
		t.Error("Test Failed - GetAssets() error", err)
	}
}

func TestGetAssetPairs(t *testing.T) {
	t.Parallel()
	_, err := k.GetAssetPairs()
	if err != nil {
		t.Error("Test Failed - GetAssetPairs() error", err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := k.GetTicker("BCHEUR")
	if err != nil {
		t.Error("Test Failed - GetTicker() error", err)
	}
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	_, err := k.GetTickers("LTCUSD,ETCUSD")
	if err != nil {
		t.Error("Test failed - GetTickers() error", err)
	}
}

func TestGetOHLC(t *testing.T) {
	t.Parallel()
	_, err := k.GetOHLC("BCHEUR")
	if err != nil {
		t.Error("Test Failed - GetOHLC() error", err)
	}
}

func TestGetDepth(t *testing.T) {
	t.Parallel()
	_, err := k.GetDepth("BCHEUR")
	if err != nil {
		t.Error("Test Failed - GetDepth() error", err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := k.GetTrades("BCHEUR")
	if err != nil {
		t.Error("Test Failed - GetTrades() error", err)
	}
}

func TestGetSpread(t *testing.T) {
	t.Parallel()
	_, err := k.GetSpread("BCHEUR")
	if err != nil {
		t.Error("Test Failed - GetSpread() error", err)
	}
}

func TestGetBalance(t *testing.T) {
	t.Parallel()
	_, err := k.GetBalance()
	if err == nil {
		t.Error("Test Failed - GetBalance() error", err)
	}
}

func TestGetTradeBalance(t *testing.T) {
	t.Parallel()
	args := TradeBalanceOptions{Asset: "ZEUR"}
	_, err := k.GetTradeBalance(args)
	if err == nil {
		t.Error("Test Failed - GetTradeBalance() error", err)
	}
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	args := OrderInfoOptions{Trades: true}
	_, err := k.GetOpenOrders(args)
	if err == nil {
		t.Error("Test Failed - GetOpenOrders() error", err)
	}
}

func TestGetClosedOrders(t *testing.T) {
	t.Parallel()
	args := GetClosedOrdersOptions{Trades: true, Start: "OE4KV4-4FVQ5-V7XGPU"}
	_, err := k.GetClosedOrders(args)
	if err == nil {
		t.Error("Test Failed - GetClosedOrders() error", err)
	}
}

func TestQueryOrdersInfo(t *testing.T) {
	t.Parallel()
	args := OrderInfoOptions{Trades: true}
	_, err := k.QueryOrdersInfo(args, "OR6ZFV-AA6TT-CKFFIW", "OAMUAJ-HLVKG-D3QJ5F")
	if err == nil {
		t.Error("Test Failed - QueryOrdersInfo() error", err)
	}
}

func TestGetTradesHistory(t *testing.T) {
	t.Parallel()
	args := GetTradesHistoryOptions{Trades: true, Start: "TMZEDR-VBJN2-NGY6DX", End: "TVRXG2-R62VE-RWP3UW"}
	_, err := k.GetTradesHistory(args)
	if err == nil {
		t.Error("Test Failed - GetTradesHistory() error", err)
	}
}

func TestQueryTrades(t *testing.T) {
	t.Parallel()
	_, err := k.QueryTrades(true, "TMZEDR-VBJN2-NGY6DX", "TFLWIB-KTT7L-4TWR3L", "TDVRAH-2H6OS-SLSXRX")
	if err == nil {
		t.Error("Test Failed - QueryTrades() error", err)
	}
}

func TestOpenPositions(t *testing.T) {
	t.Parallel()
	_, err := k.OpenPositions(false)
	if err == nil {
		t.Error("Test Failed - OpenPositions() error", err)
	}
}

func TestGetLedgers(t *testing.T) {
	t.Parallel()
	args := GetLedgersOptions{Start: "LRUHXI-IWECY-K4JYGO", End: "L5NIY7-JZQJD-3J4M2V", Ofs: 15}
	_, err := k.GetLedgers(args)
	if err == nil {
		t.Error("Test Failed - GetLedgers() error", err)
	}
}

func TestQueryLedgers(t *testing.T) {
	t.Parallel()
	_, err := k.QueryLedgers("LVTSFS-NHZVM-EXNZ5M")
	if err == nil {
		t.Error("Test Failed - QueryLedgers() error", err)
	}
}

func TestGetTradeVolume(t *testing.T) {
	t.Parallel()
	_, err := k.GetTradeVolume(true, "OAVY7T-MV5VK-KHDF5X")
	if err == nil {
		t.Error("Test Failed - GetTradeVolume() error", err)
	}
}

func TestAddOrder(t *testing.T) {
	t.Parallel()
	args := AddOrderOptions{Oflags: "fcib"}
	_, err := k.AddOrder("XXBTZUSD", "sell", "market", 0.00000001, 0, 0, 0, args)
	if err == nil {
		t.Error("Test Failed - AddOrder() error", err)
	}
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()
	_, err := k.CancelExistingOrder("OAVY7T-MV5VK-KHDF5X")
	if err == nil {
		t.Error("Test Failed - CancelExistingOrder() error", err)
	}
}

func setFeeBuilder() exchange.FeeBuilder {
	return exchange.FeeBuilder{
		Amount:              1,
		Delimiter:           "",
		FeeType:             exchange.CryptocurrencyTradeFee,
		FirstCurrency:       symbol.XXBT,
		SecondCurrency:      symbol.ZUSD,
		IsMaker:             false,
		PurchasePrice:       1,
		CurrencyItem:        symbol.USD,
		BankTransactionType: exchange.WireTransfer,
	}
}

func TestGetFee(t *testing.T) {
	k.SetDefaults()
	TestSetup(t)
	var feeBuilder = setFeeBuilder()

	if areTestAPIKeysSet() {
		// CryptocurrencyTradeFee Basic
		if resp, err := k.GetFee(feeBuilder); resp != float64(0.0026) || err != nil {
			t.Error(err)
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.0026), resp)
		}

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		if resp, err := k.GetFee(feeBuilder); resp != float64(2600) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(2600), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee IsMaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsMaker = true
		if resp, err := k.GetFee(feeBuilder); resp != float64(0.0016) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.0016), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		if resp, err := k.GetFee(feeBuilder); resp != float64(0) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
			t.Error(err)
		}

		// InternationalBankDepositFee Basic
		feeBuilder = setFeeBuilder()
		feeBuilder.FeeType = exchange.InternationalBankDepositFee
		if resp, err := k.GetFee(feeBuilder); resp != float64(5) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(5), resp)
			t.Error(err)
		}
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	feeBuilder.FirstCurrency = symbol.XXBT
	if resp, err := k.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(5), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := k.GetFee(feeBuilder); resp != float64(0.0005) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.0005), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.FirstCurrency = "hello"
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := k.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.CurrencyItem = symbol.USD
	if resp, err := k.GetFee(feeBuilder); resp != float64(5) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(5), resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	k.SetDefaults()
	expectedResult := exchange.AutoWithdrawCryptoWithSetupText + " & " + exchange.WithdrawCryptoWith2FAText + " & " + exchange.AutoWithdrawFiatWithSetupText + " & " + exchange.WithdrawFiatWith2FAText

	withdrawPermissions := k.FormatWithdrawPermissions()

	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	k.SetDefaults()
	TestSetup(t)

	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType: exchange.AnyOrderType,
	}

	_, err := k.GetActiveOrders(getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestGetOrderHistory(t *testing.T) {
	k.SetDefaults()
	TestSetup(t)

	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType: exchange.AnyOrderType,
	}

	_, err := k.GetOrderHistory(getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get order history: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func areTestAPIKeysSet() bool {
	return k.ValidateAPICredentials()
}

func TestSubmitOrder(t *testing.T) {
	k.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var p = pair.CurrencyPair{
		Delimiter:      "",
		FirstCurrency:  symbol.XBT,
		SecondCurrency: symbol.CAD,
	}
	response, err := k.SubmitOrder(p, exchange.BuyOrderSide, exchange.MarketOrderType, 1, 10, "hi")
	if areTestAPIKeysSet() && (err != nil || !response.IsOrderPlaced) {
		t.Errorf("Order failed to be placed: %v", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	k.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := pair.NewCurrencyPair(symbol.LTC, symbol.BTC)

	var orderCancellation = exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	err := k.CancelOrder(orderCancellation)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	k.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := pair.NewCurrencyPair(symbol.LTC, symbol.BTC)

	var orderCancellation = exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	resp, err := k.CancelAllOrders(orderCancellation)

	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}

	if len(resp.OrderStatus) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.OrderStatus))
	}
}

func TestGetAccountInfo(t *testing.T) {
	if apiKey != "" || apiSecret != "" || clientID != "" {
		_, err := k.GetAccountInfo()
		if err != nil {
			t.Error("Test Failed - GetAccountInfo() error", err)
		}
	} else {
		_, err := k.GetAccountInfo()
		if err == nil {
			t.Error("Test Failed - GetAccountInfo() error")
		}
	}
}

func TestModifyOrder(t *testing.T) {
	_, err := k.ModifyOrder(exchange.ModifyOrder{})
	if err == nil {
		t.Error("Test failed - ModifyOrder() error")
	}
}

func TestWithdraw(t *testing.T) {
	k.SetDefaults()
	TestSetup(t)
	var withdrawCryptoRequest = exchange.WithdrawRequest{
		Amount:        100,
		Currency:      symbol.XXBT,
		Address:       "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		Description:   "donation",
		TradePassword: "Key",
	}

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	_, err := k.WithdrawCryptocurrencyFunds(withdrawCryptoRequest)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestWithdrawFiat(t *testing.T) {
	k.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = exchange.WithdrawRequest{
		Amount:        100,
		Currency:      symbol.EUR,
		Address:       "",
		Description:   "donation",
		TradePassword: "someBank",
	}

	_, err := k.WithdrawFiatFunds(withdrawFiatRequest)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	k.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = exchange.WithdrawRequest{
		Amount:        100,
		Currency:      symbol.EUR,
		Address:       "",
		Description:   "donation",
		TradePassword: "someBank",
	}

	_, err := k.WithdrawFiatFundsToInternationalBank(withdrawFiatRequest)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	if areTestAPIKeysSet() {
		_, err := k.GetDepositAddress(symbol.BTC, "")
		if err != nil {
			t.Error("Test Failed - GetDepositAddress() error", err)
		}
	} else {
		_, err := k.GetDepositAddress(symbol.BTC, "")
		if err == nil {
			t.Error("Test Failed - GetDepositAddress() error can not be nil")
		}
	}
}

func TestWithdrawStatus(t *testing.T) {
	k.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() {
		_, err := k.WithdrawStatus(symbol.BTC, "")
		if err != nil {
			t.Error("Test Failed - WithdrawStatus() error", err)
		}
	} else {
		_, err := k.WithdrawStatus(symbol.BTC, "")
		if err == nil {
			t.Error("Test Failed - GetDepositAddress() error can not be nil", err)
		}
	}
}

func TestWithdrawCancel(t *testing.T) {
	k.SetDefaults()
	TestSetup(t)

	_, err := k.WithdrawCancel(symbol.BTC, "")
	if areTestAPIKeysSet() && err == nil {
		t.Error("Test Failed - WithdrawCancel() error cannot be nil")
	} else if !areTestAPIKeysSet() && err == nil {
		t.Errorf("Test Failed - WithdrawCancel() error - expecting an error when no keys are set but received nil")
	}
}
