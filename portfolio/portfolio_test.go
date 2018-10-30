package portfolio

import (
	"reflect"
	"testing"
	"time"

	"github.com/thrasher-/gocryptotrader/currency/symbol"
)

func TestGetEthereumBalance(t *testing.T) {
	address := "0xb794f5ea0ba39494ce839613fffba74279579268"
	nonsenseAddress := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

	response, err := GetEthereumBalance(address)
	if err != nil {
		t.Errorf("Test Failed - Portfolio GetEthereumBalance() Error: %s", err)
	}

	if response.Address != "0xb794f5ea0ba39494ce839613fffba74279579268" {
		t.Error("Test Failed - Portfolio GetEthereumBalance() address invalid")
	}

	response, err = GetEthereumBalance(nonsenseAddress)
	if response.Error.Message != "" || err == nil {
		t.Errorf("Test Failed - Portfolio GetEthereumBalance() Error: %s",
			response.Error.Message)
	}
}

func TestGetCryptoIDBalance(t *testing.T) {
	ltcAddress := "LX2LMYXtuv5tiYEMztSSoEZcafFPYJFRK1"
	_, err := GetCryptoIDAddress(ltcAddress, symbol.LTC)
	if err != nil {
		t.Fatalf("Test failed. TestGetCryptoIDBalance error: %s", err)
	}
}

func TestGetAddressBalance(t *testing.T) {
	ltcAddress := "LdP8Qox1VAhCzLJNqrr74YovaWYyNBUWvL"
	ltc := symbol.LTC
	description := "Description of Wallet"
	balance := float64(1000)

	portfolio := Base{}
	portfolio.AddAddress(ltcAddress, ltc, description, balance)

	addBalance, _ := portfolio.GetAddressBalance("LdP8Qox1VAhCzLJNqrr74YovaWYyNBUWvL", ltc, description)
	if addBalance != balance {
		t.Error("Test Failed - Portfolio GetAddressBalance() Error: Incorrect value")
	}

	addBalance, found := portfolio.GetAddressBalance("WigWham", ltc, description)
	if addBalance != 0 {
		t.Error("Test Failed - Portfolio GetAddressBalance() Error: Incorrect value")
	}
	if found {
		t.Error("Test Failed - Portfolio GetAddressBalance() Error: Incorrect value")
	}
}

func TestExchangeExists(t *testing.T) {
	newBase := Base{}
	newBase.AddAddress("someaddress", symbol.LTC, "LTCWALLETTEST", 0.02)
	if !newBase.ExchangeExists("someaddress") {
		t.Error("Test Failed - portfolio_test.go - AddressExists error")
	}
	if newBase.ExchangeExists("bla") {
		t.Error("Test Failed - portfolio_test.go - AddressExists error")
	}
}

func TestAddressExists(t *testing.T) {
	newbase := Base{}
	newbase.AddAddress("someaddress", symbol.LTC, "LTCWALLETTEST", 0.02)
	if !newbase.AddressExists("someaddress") {
		t.Error("Test Failed - portfolio_test.go - AddressExists error")
	}
	if newbase.AddressExists("bla") {
		t.Error("Test Failed - portfolio_test.go - AddressExists error")
	}
}

func TestExchangeAddressExists(t *testing.T) {
	newbase := Base{}
	newbase.AddAddress("someaddress", symbol.LTC, "LTCWALLETTEST", 0.02)
	if !newbase.ExchangeAddressExists("someaddress", symbol.LTC) {
		t.Error("Test Failed - portfolio_test.go - ExchangeAddressExists error")
	}
	if newbase.ExchangeAddressExists("TEST", symbol.LTC) {
		t.Error("Test Failed - portfolio_test.go - ExchangeAddressExists error")
	}

}

func TestAddExchangeAddress(t *testing.T) {
	newbase := Base{}
	newbase.AddExchangeAddress("ANX", symbol.BTC, 100)
	newbase.AddExchangeAddress("ANX", symbol.BTC, 200)

	if !newbase.ExchangeAddressExists("ANX", symbol.BTC) {
		t.Error("Test Failed - TestExchangeAddressExists address doesn't exist")
	}
}

func TestUpdateAddressBalance(t *testing.T) {
	newbase := Base{}
	newbase.AddAddress("someaddress", symbol.LTC, "LTCWALLETTEST", 0.02)
	newbase.UpdateAddressBalance("someaddress", 0.03)

	value := newbase.GetPortfolioSummary()
	if value.Totals[0].Coin != symbol.LTC && value.Totals[0].Balance != 0.03 {
		t.Error("Test Failed - portfolio_test.go - UpdateUpdateAddressBalance error")
	}
}

func TestRemoveAddress(t *testing.T) {
	newbase := Base{}
	newbase.AddAddress("someaddr", symbol.LTC, "LTCWALLETTEST", 420)

	if !newbase.AddressExists("someaddr") {
		t.Error("Test failed - portfolio_test.go - TestRemoveAddress")
	}

	newbase.RemoveAddress("someaddr", symbol.LTC, "LTCWALLETTEST")
	if newbase.AddressExists("someaddr") {
		t.Error("Test failed - portfolio_test.go - TestRemoveAddress")
	}
}

func TestRemoveExchangeAddress(t *testing.T) {
	newbase := Base{}
	exchangeName := "BallerExchange"
	coinType := symbol.LTC

	newbase.AddExchangeAddress(exchangeName, coinType, 420)

	if !newbase.ExchangeAddressExists(exchangeName, coinType) {
		t.Error("Test failed - portfolio_test.go - TestRemoveAddress")
	}

	newbase.RemoveExchangeAddress(exchangeName, coinType)
	if newbase.ExchangeAddressExists(exchangeName, coinType) {
		t.Error("Test failed - portfolio_test.go - TestRemoveAddress")
	}
}

func TestUpdateExchangeAddressBalance(t *testing.T) {
	newbase := Base{}
	newbase.AddExchangeAddress("someaddress", symbol.LTC, 0.02)
	portfolio := GetPortfolio()
	portfolio.Seed(newbase)
	portfolio.UpdateExchangeAddressBalance("someaddress", symbol.LTC, 0.04)

	value := portfolio.GetPortfolioSummary()
	if value.Totals[0].Coin != symbol.LTC && value.Totals[0].Balance != 0.04 {
		t.Error("Test Failed - portfolio_test.go - UpdateExchangeAddressBalance error")
	}
}

func TestAddAddress(t *testing.T) {
	newbase := Base{}
	newbase.AddAddress("Gibson", symbol.LTC, "LTCWALLETTEST", 0.02)
	portfolio := GetPortfolio()
	portfolio.Seed(newbase)
	if !portfolio.AddressExists("Gibson") {
		t.Error("Test Failed - portfolio_test.go - AddAddress error")
	}

	// Test updating balance to <= 0, expected result is to remove the address.
	// Fail if address still exists.
	newbase.AddAddress("Gibson", symbol.LTC, "LTCWALLETTEST", -1)
	if newbase.AddressExists("Gibson") {
		t.Error("Test Failed - portfolio_test.go - AddAddress error")
	}
}

func TestUpdatePortfolio(t *testing.T) {
	newbase := Base{}
	newbase.AddAddress("someaddress", symbol.LTC, "LTCWALLETTEST", 0.02)
	portfolio := GetPortfolio()
	portfolio.Seed(newbase)

	value := portfolio.UpdatePortfolio(
		[]string{"LdP8Qox1VAhCzLJNqrr74YovaWYyNBUWvL"}, symbol.LTC,
	)
	if !value {
		t.Error("Test Failed - portfolio_test.go - UpdatePortfolio error")
	}
	value = portfolio.UpdatePortfolio([]string{"Testy"}, symbol.LTC)
	if value {
		t.Error("Test Failed - portfolio_test.go - UpdatePortfolio error")
	}
	value = portfolio.UpdatePortfolio(
		[]string{"LdP8Qox1VAhCzLJNqrr74YovaWYyNBUWvL", "LVa8wZ983PvWtdwXZ8viK6SocMENLCXkEy"},
		symbol.LTC,
	)
	if !value {
		t.Error("Test Failed - portfolio_test.go - UpdatePortfolio error")
	}
	value = portfolio.UpdatePortfolio(
		[]string{"LdP8Qox1VAhCzLJNqrr74YovaWYyNBUWvL", "Testy"}, symbol.LTC,
	)
	if value {
		t.Error("Test Failed - portfolio_test.go - UpdatePortfolio error")
	}

	time.Sleep(time.Second * 5)
	value = portfolio.UpdatePortfolio(
		[]string{"0xb794f5ea0ba39494ce839613fffba74279579268",
			"0xe853c56864a2ebe4576a807d26fdc4a0ada51919"}, symbol.ETH,
	)
	if !value {
		t.Error("Test Failed - portfolio_test.go - UpdatePortfolio error")
	}
	value = portfolio.UpdatePortfolio(
		[]string{"0xb794f5ea0ba39494ce839613fffba74279579268", "TESTY"}, symbol.ETH,
	)
	if value {
		t.Error("Test Failed - portfolio_test.go - UpdatePortfolio error")
	}

	value = portfolio.UpdatePortfolio(
		[]string{PortfolioAddressExchange, PortfolioAddressPersonal}, symbol.LTC)

	if !value {
		t.Error("Test Failed - portfolio_test.go - UpdatePortfolio error")
	}
}

func TestGetPortfolioByExchange(t *testing.T) {
	newbase := Base{}
	newbase.AddExchangeAddress("ANX", symbol.LTC, 0.07)
	newbase.AddExchangeAddress("Bitfinex", symbol.LTC, 0.05)
	newbase.AddAddress("someaddress", symbol.LTC, PortfolioAddressPersonal, 0.03)
	portfolio := GetPortfolio()
	portfolio.Seed(newbase)
	value := portfolio.GetPortfolioByExchange("ANX")
	result, ok := value[symbol.LTC]
	if !ok {
		t.Error("Test Failed - portfolio_test.go - GetPortfolioByExchange error")
	}

	if result != 0.07 {
		t.Error("Test Failed - portfolio_test.go - GetPortfolioByExchange result != 0.10")
	}

	value = portfolio.GetPortfolioByExchange("Bitfinex")
	result, ok = value[symbol.LTC]
	if !ok {
		t.Error("Test Failed - portfolio_test.go - GetPortfolioByExchange error")
	}

	if result != 0.05 {
		t.Error("Test Failed - portfolio_test.go - GetPortfolioByExchange result != 0.05")
	}
}

func TestGetExchangePortfolio(t *testing.T) {
	newbase := Base{}
	newbase.AddAddress("ANX", symbol.LTC, PortfolioAddressExchange, 0.03)
	newbase.AddAddress("Bitfinex", symbol.LTC, PortfolioAddressExchange, 0.05)
	newbase.AddAddress("someaddress", symbol.LTC, PortfolioAddressPersonal, 0.03)
	portfolio := GetPortfolio()
	portfolio.Seed(newbase)
	value := portfolio.GetExchangePortfolio()

	result, ok := value[symbol.LTC]
	if !ok {
		t.Error("Test Failed - portfolio_test.go - GetExchangePortfolio error")
	}

	if result != 0.08 {
		t.Error("Test Failed - portfolio_test.go - GetExchangePortfolio result != 0.08")
	}
}

func TestGetPersonalPortfolio(t *testing.T) {
	newbase := Base{}
	newbase.AddAddress("someaddress", symbol.LTC, "LTCWALLETTEST", 0.02)
	newbase.AddAddress("anotheraddress", symbol.LTC, "LTCWALLETTEST", 0.03)
	newbase.AddAddress("Exchange", symbol.LTC, PortfolioAddressExchange, 0.01)
	portfolio := GetPortfolio()
	portfolio.Seed(newbase)
	value := portfolio.GetPersonalPortfolio()
	result, ok := value[symbol.LTC]
	if !ok {
		t.Error("Test Failed - portfolio_test.go - GetPersonalPortfolio error")
	}

	if result != 0.05 {
		t.Error("Test Failed - portfolio_test.go - GetPersonalPortfolio result != 0.05")
	}
}

func TestGetPortfolioSummary(t *testing.T) {
	newbase := Base{}
	// Personal holdings
	newbase.AddAddress("someaddress", symbol.LTC, PortfolioAddressPersonal, 1)
	newbase.AddAddress("someaddress2", symbol.LTC, PortfolioAddressPersonal, 2)
	newbase.AddAddress("someaddress3", symbol.BTC, PortfolioAddressPersonal, 100)
	newbase.AddAddress("0xde0b295669a9fd93d5f28d9ec85e40f4cb697bae", symbol.ETH,
		PortfolioAddressPersonal, 865346880000000000)
	newbase.AddAddress("0x9edc81c813b26165f607a8d1b8db87a02f34307f", symbol.ETH,
		PortfolioAddressPersonal, 165346880000000000)

	// Exchange holdings
	newbase.AddExchangeAddress("Bitfinex", symbol.LTC, 20)
	newbase.AddExchangeAddress("Bitfinex", symbol.BTC, 100)
	newbase.AddExchangeAddress("ANX", symbol.ETH, 42)

	portfolio := GetPortfolio()
	portfolio.Seed(newbase)
	value := portfolio.GetPortfolioSummary()

	getTotalsVal := func(s string) Coin {
		for x := range value.Totals {
			if value.Totals[x].Coin == s {
				return value.Totals[x]
			}
		}
		return Coin{}
	}

	if getTotalsVal(symbol.LTC).Coin != symbol.LTC {
		t.Error("Test Failed - portfolio_test.go - TestGetPortfolioSummary error")
	}

	if getTotalsVal(symbol.ETH).Coin != symbol.ETH {
		t.Error("Test Failed - portfolio_test.go - TestGetPortfolioSummary error")
	}

	if getTotalsVal(symbol.LTC).Balance != 23 {
		t.Error("Test Failed - portfolio_test.go - TestGetPortfolioSummary error")
	}

	if getTotalsVal(symbol.BTC).Balance != 200 {
		t.Error("Test Failed - portfolio_test.go - TestGetPortfolioSummary error")
	}
}

func TestGetPortfolioGroupedCoin(t *testing.T) {
	newbase := Base{}
	newbase.AddAddress("someaddress", symbol.LTC, "LTCWALLETTEST", 0.02)
	newbase.AddAddress("Exchange", symbol.LTC, PortfolioAddressExchange, 0.05)
	portfolio := GetPortfolio()
	portfolio.Seed(newbase)
	value := portfolio.GetPortfolioGroupedCoin()
	if value[symbol.LTC][0] != "someaddress" && len(value[symbol.LTC][0]) != 1 {
		t.Error("Test Failed - portfolio_test.go - GetPortfolioGroupedCoin error")
	}
}

func TestSeed(t *testing.T) {
	newbase := Base{}
	newbase.AddAddress("someaddress", symbol.LTC, "LTCWALLETTEST", 0.02)
	portfolio := GetPortfolio()
	portfolio.Seed(newbase)

	if !portfolio.AddressExists("someaddress") {
		t.Error("Test Failed - portfolio_test.go - Seed error")
	}
}

func TestStartPortfolioWatcher(t *testing.T) {
	newBase := Base{}
	newBase.AddAddress("LX2LMYXtuv5tiYEMztSSoEZcafFPYJFRK1", symbol.LTC, PortfolioAddressPersonal, 0.02)
	newBase.AddAddress("Testy", symbol.LTC, PortfolioAddressPersonal, 0.02)
	portfolio := GetPortfolio()
	portfolio.Seed(newBase)

	if !portfolio.AddressExists("LX2LMYXtuv5tiYEMztSSoEZcafFPYJFRK1") {
		t.Error("Test Failed - portfolio_test.go - TestStartPortfolioWatcher")
	}

	go StartPortfolioWatcher()
}

func TestGetPortfolio(t *testing.T) {
	ptrBASE := GetPortfolio()
	if reflect.TypeOf(ptrBASE).String() != "*portfolio.Base" {
		t.Error("Test Failed - portfolio_test.go - GetoPortfolio error")
	}
}
