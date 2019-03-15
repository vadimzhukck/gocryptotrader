package anx

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/currency/symbol"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
)

// Please supply your own keys here for due diligence testing
const (
	testAPIKey              = ""
	testAPISecret           = ""
	canManipulateRealOrders = false
)

var a ANX

func TestSetDefaults(t *testing.T) {
	a.SetDefaults()

	if a.Name != "ANX" {
		t.Error("Test Failed - ANX SetDefaults() incorrect values set")
	}
	if !a.Enabled {
		t.Error("Test Failed - ANX SetDefaults() incorrect values set")
	}
	if !a.Verbose {
		t.Error("Test Failed - ANX SetDefaults() incorrect values set")
	}
}

func TestSetup(t *testing.T) {
	anxSetupConfig := config.GetConfig()
	anxSetupConfig.LoadConfig("../../testdata/configtest.json")
	anxConfig, err := anxSetupConfig.GetExchangeConfig("ANX")
	anxConfig.API.AuthenticatedSupport = true

	if err != nil {
		t.Error("Test Failed - ANX Setup() init error")
	}

	a.Setup(anxConfig)
	a.API.Credentials.Key = testAPIKey
	a.API.Credentials.Secret = testAPISecret
	a.API.AuthenticatedSupport = true

	if !a.Enabled {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
	if a.Verbose {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
	if len(a.BaseCurrencies) == 0 {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
	if len(a.CurrencyPairs.Spot.Available) == 0 {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
	if len(a.CurrencyPairs.Spot.Enabled) == 0 {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
}

func TestGetCurrencies(t *testing.T) {
	_, err := a.GetCurrencies()
	if err != nil {
		t.Fatalf("Test failed. TestGetCurrencies failed. Err: %s", err)
	}
}

func TestFetchTradablePairs(t *testing.T) {
	_, err := a.FetchTradablePairs(assets.AssetTypeSpot)
	if err != nil {
		t.Fatalf("Test failed. TestGetTradablePairs failed. Err: %s", err)
	}
}

func TestGetTicker(t *testing.T) {
	ticker, err := a.GetTicker("BTCUSD")
	if err != nil {
		t.Errorf("Test Failed - ANX GetTicker() error: %s", err)
	}
	if ticker.Result != "success" {
		t.Error("Test Failed - ANX GetTicker() unsuccessful")
	}
}

func TestGetDepth(t *testing.T) {
	ticker, err := a.GetDepth("BTCUSD")
	if err != nil {
		t.Errorf("Test Failed - ANX GetDepth() error: %s", err)
	}
	if ticker.Result != "success" {
		t.Error("Test Failed - ANX GetDepth() unsuccessful")
	}
}

func TestGetAPIKey(t *testing.T) {
	apiKey, apiSecret, err := a.GetAPIKey("userName", "passWord", "", "1337")
	if err == nil {
		t.Error("Test Failed - ANX GetAPIKey() Incorrect")
	}
	if apiKey != "" {
		t.Error("Test Failed - ANX GetAPIKey() Incorrect")
	}
	if apiSecret != "" {
		t.Error("Test Failed - ANX GetAPIKey() Incorrect")
	}
}

func setFeeBuilder() exchange.FeeBuilder {
	return exchange.FeeBuilder{
		Amount:         1,
		Delimiter:      "",
		FeeType:        exchange.CryptocurrencyTradeFee,
		FirstCurrency:  symbol.BTC,
		SecondCurrency: symbol.LTC,
		IsMaker:        false,
		PurchasePrice:  1,
	}
}

func TestGetFee(t *testing.T) {
	a.SetDefaults()
	TestSetup(t)

	var feeBuilder = setFeeBuilder()

	// CryptocurrencyTradeFee Basic
	if resp, err := a.GetFee(feeBuilder); resp != float64(0.02) || err != nil {
		t.Error(err)
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	if resp, err := a.GetFee(feeBuilder); resp != float64(20000) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(20000), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	if resp, err := a.GetFee(feeBuilder); resp != float64(0.01) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.01), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	if resp, err := a.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := a.GetFee(feeBuilder); resp != float64(0.002) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := a.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.CurrencyItem = symbol.HKD
	if resp, err := a.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.CurrencyItem = symbol.HKD
	if resp, err := a.GetFee(feeBuilder); resp != float64(250.01) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(250.01), resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	a.SetDefaults()
	expectedResult := exchange.AutoWithdrawCryptoWithSetupText + " & " + exchange.WithdrawCryptoWith2FAText + " & " +
		exchange.WithdrawCryptoWithEmailText + " & " + exchange.WithdrawFiatViaWebsiteOnlyText

	withdrawPermissions := a.FormatWithdrawPermissions()

	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	a.SetDefaults()
	TestSetup(t)

	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType: exchange.AnyOrderType,
	}

	_, err := a.GetActiveOrders(getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestGetOrderHistory(t *testing.T) {
	a.SetDefaults()
	TestSetup(t)

	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType: exchange.AnyOrderType,
	}

	_, err := a.GetOrderHistory(getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get order history: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------

func areTestAPIKeysSet() bool {
	return a.ValidateAPICredentials()
}

func TestSubmitOrder(t *testing.T) {
	a.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var p = pair.CurrencyPair{
		Delimiter:      "_",
		FirstCurrency:  symbol.BTC,
		SecondCurrency: symbol.USD,
	}
	response, err := a.SubmitOrder(p, exchange.BuyOrderSide, exchange.MarketOrderType, 1, 1, "clientId")
	if areTestAPIKeysSet() && (err != nil || !response.IsOrderPlaced) {
		t.Errorf("Order failed to be placed: %v", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	a.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := pair.NewCurrencyPair(symbol.BTC, symbol.LTC)

	var orderCancellation = exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	err := a.CancelOrder(orderCancellation)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel order: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	a.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := pair.NewCurrencyPair(symbol.BTC, symbol.LTC)

	var orderCancellation = exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	resp, err := a.CancelAllOrders(orderCancellation)

	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel order: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}

	if len(resp.OrderStatus) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.OrderStatus))
	}
}

func TestGetAccountInfo(t *testing.T) {
	if testAPIKey != "" || testAPISecret != "" {
		_, err := a.GetAccountInfo()
		if err != nil {
			t.Error("test failed - GetAccountInfo() error:", err)
		}
	} else {
		_, err := a.GetAccountInfo()
		if err == nil {
			t.Error("test failed - GetAccountInfo() error")
		}
	}
}

func TestModifyOrder(t *testing.T) {
	_, err := a.ModifyOrder(exchange.ModifyOrder{})
	if err == nil {
		t.Error("Test failed - ModifyOrder() error")
	}
}

func TestWithdraw(t *testing.T) {
	a.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawCryptoRequest = exchange.WithdrawRequest{
		Amount:      100,
		Currency:    symbol.BTC,
		Address:     "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		Description: "WITHDRAW IT ALL",
		AddressTag:  "0123456789",
	}

	_, err := a.WithdrawCryptocurrencyFunds(withdrawCryptoRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestWithdrawFiat(t *testing.T) {
	a.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = exchange.WithdrawRequest{}

	_, err := a.WithdrawFiatFunds(withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	a.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = exchange.WithdrawRequest{}

	_, err := a.WithdrawFiatFundsToInternationalBank(withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	if areTestAPIKeysSet() {
		_, err := a.GetDepositAddress(symbol.BTC, "")
		if err != nil {
			t.Error("Test Failed - GetDepositAddress() error", err)
		}
	} else {
		_, err := a.GetDepositAddress(symbol.BTC, "")
		if err == nil {
			t.Error("Test Failed - GetDepositAddress() error cannot be nil")
		}
	}
}
