package orderbook

import (
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
)

func TestCalculateTotalBids(t *testing.T) {
	t.Parallel()
	currency := pair.NewCurrencyPair("BTC", "USD")
	base := Base{
		Pair:         currency,
		CurrencyPair: currency.Pair().String(),
		Bids:         []Item{{Price: 100, Amount: 10}},
		LastUpdated:  time.Now(),
	}

	a, b := base.CalculateTotalBids()
	if a != 10 && b != 1000 {
		t.Fatal("Test failed. TestCalculateTotalBids expected a = 10 and b = 1000")
	}
}

func TestCalculateTotaAsks(t *testing.T) {
	t.Parallel()
	currency := pair.NewCurrencyPair("BTC", "USD")
	base := Base{
		Pair:         currency,
		CurrencyPair: currency.Pair().String(),
		Asks:         []Item{{Price: 100, Amount: 10}},
		LastUpdated:  time.Now(),
	}

	a, b := base.CalculateTotalAsks()
	if a != 10 && b != 1000 {
		t.Fatal("Test failed. TestCalculateTotalAsks expected a = 10 and b = 1000")
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()
	currency := pair.NewCurrencyPair("BTC", "USD")
	timeNow := time.Now()
	base := Base{
		Pair:         currency,
		CurrencyPair: currency.Pair().String(),
		Asks:         []Item{{Price: 100, Amount: 10}},
		Bids:         []Item{{Price: 200, Amount: 10}},
		LastUpdated:  timeNow,
	}

	asks := []Item{{Price: 200, Amount: 101}}
	bids := []Item{{Price: 201, Amount: 100}}
	time.Sleep(time.Millisecond * 50)
	base.Update(bids, asks)

	if !base.LastUpdated.After(timeNow) {
		t.Fatal("test failed. TestUpdate expected LastUpdated to be greater then original time")
	}

	a, b := base.CalculateTotalAsks()
	if a != 100 && b != 20200 {
		t.Fatal("Test failed. TestUpdate expected a = 100 and b = 20100")
	}

	a, b = base.CalculateTotalBids()
	if a != 100 && b != 20100 {
		t.Fatal("Test failed. TestUpdate expected a = 100 and b = 20100")
	}
}

func TestGetOrderbook(t *testing.T) {
	currency := pair.NewCurrencyPair("BTC", "USD")
	base := Base{
		Pair:         currency,
		CurrencyPair: currency.Pair().String(),
		Asks:         []Item{{Price: 100, Amount: 10}},
		Bids:         []Item{{Price: 200, Amount: 10}},
	}

	CreateNewOrderbook("Exchange", currency, base, assets.AssetTypeSpot)

	result, err := GetOrderbook("Exchange", currency, assets.AssetTypeSpot)
	if err != nil {
		t.Fatalf("Test failed. TestGetOrderbook failed to get orderbook. Error %s",
			err)
	}

	if result.Pair.Pair() != currency.Pair() {
		t.Fatal("Test failed. TestGetOrderbook failed. Mismatched pairs")
	}

	_, err = GetOrderbook("nonexistent", currency, assets.AssetTypeSpot)
	if err == nil {
		t.Fatal("Test failed. TestGetOrderbook retrieved non-existent orderbook")
	}

	currency.FirstCurrency = "blah"
	_, err = GetOrderbook("Exchange", currency, assets.AssetTypeSpot)
	if err == nil {
		t.Fatal("Test failed. TestGetOrderbook retrieved non-existent orderbook using invalid first currency")
	}

	newCurrency := pair.NewCurrencyPair("BTC", "AUD")
	_, err = GetOrderbook("Exchange", newCurrency, assets.AssetTypeSpot)
	if err == nil {
		t.Fatal("Test failed. TestGetOrderbook retrieved non-existent orderbook using invalid second currency")
	}
}

func TestGetOrderbookByExchange(t *testing.T) {
	currency := pair.NewCurrencyPair("BTC", "USD")
	base := Base{
		Pair:         currency,
		CurrencyPair: currency.Pair().String(),
		Asks:         []Item{{Price: 100, Amount: 10}},
		Bids:         []Item{{Price: 200, Amount: 10}},
	}

	CreateNewOrderbook("Exchange", currency, base, assets.AssetTypeSpot)

	_, err := GetOrderbookByExchange("Exchange")
	if err != nil {
		t.Fatalf("Test failed. TestGetOrderbookByExchange failed to get orderbook. Error %s",
			err)
	}

	_, err = GetOrderbookByExchange("nonexistent")
	if err == nil {
		t.Fatal("Test failed. TestGetOrderbookByExchange retrieved non-existent orderbook")
	}
}

func TestFirstCurrencyExists(t *testing.T) {
	currency := pair.NewCurrencyPair("BTC", "AUD")
	base := Base{
		Pair:         currency,
		CurrencyPair: currency.Pair().String(),
		Asks:         []Item{{Price: 100, Amount: 10}},
		Bids:         []Item{{Price: 200, Amount: 10}},
	}

	CreateNewOrderbook("Exchange", currency, base, assets.AssetTypeSpot)

	if !FirstCurrencyExists("Exchange", currency.FirstCurrency) {
		t.Fatal("Test failed. TestFirstCurrencyExists expected first currency doesn't exist")
	}

	var item pair.CurrencyItem = "blah"
	if FirstCurrencyExists("Exchange", item) {
		t.Fatal("Test failed. TestFirstCurrencyExists unexpected first currency exists")
	}
}

func TestSecondCurrencyExists(t *testing.T) {
	currency := pair.NewCurrencyPair("BTC", "USD")
	base := Base{
		Pair:         currency,
		CurrencyPair: currency.Pair().String(),
		Asks:         []Item{{Price: 100, Amount: 10}},
		Bids:         []Item{{Price: 200, Amount: 10}},
	}

	CreateNewOrderbook("Exchange", currency, base, assets.AssetTypeSpot)

	if !SecondCurrencyExists("Exchange", currency) {
		t.Fatal("Test failed. TestSecondCurrencyExists expected first currency doesn't exist")
	}

	currency.SecondCurrency = "blah"
	if SecondCurrencyExists("Exchange", currency) {
		t.Fatal("Test failed. TestSecondCurrencyExists unexpected first currency exists")
	}
}

func TestCreateNewOrderbook(t *testing.T) {
	currency := pair.NewCurrencyPair("BTC", "USD")
	base := Base{
		Pair:         currency,
		CurrencyPair: currency.Pair().String(),
		Asks:         []Item{{Price: 100, Amount: 10}},
		Bids:         []Item{{Price: 200, Amount: 10}},
	}

	CreateNewOrderbook("Exchange", currency, base, assets.AssetTypeSpot)

	result, err := GetOrderbook("Exchange", currency, assets.AssetTypeSpot)
	if err != nil {
		t.Fatal("Test failed. TestCreateNewOrderbook failed to create new orderbook")
	}

	if result.Pair.Pair() != currency.Pair() {
		t.Fatal("Test failed. TestCreateNewOrderbook result pair is incorrect")
	}

	a, b := result.CalculateTotalAsks()
	if a != 10 && b != 1000 {
		t.Fatal("Test failed. TestCreateNewOrderbook CalculateTotalAsks value is incorrect")
	}

	a, b = result.CalculateTotalBids()
	if a != 10 && b != 2000 {
		t.Fatal("Test failed. TestCreateNewOrderbook CalculateTotalBids value is incorrect")
	}
}

func TestProcessOrderbook(t *testing.T) {
	Orderbooks = []Orderbook{}
	currency := pair.NewCurrencyPair("BTC", "USD")
	base := Base{
		Pair:         currency,
		CurrencyPair: currency.Pair().String(),
		Asks:         []Item{{Price: 100, Amount: 10}},
		Bids:         []Item{{Price: 200, Amount: 10}},
	}

	ProcessOrderbook("Exchange", currency, base, assets.AssetTypeSpot)

	result, err := GetOrderbook("Exchange", currency, assets.AssetTypeSpot)
	if err != nil {
		t.Fatal("Test failed. TestProcessOrderbook failed to create new orderbook")
	}

	if result.Pair.Pair() != currency.Pair() {
		t.Fatal("Test failed. TestProcessOrderbook result pair is incorrect")
	}

	currency = pair.NewCurrencyPair("BTC", "GBP")
	base.Pair = currency
	ProcessOrderbook("Exchange", currency, base, assets.AssetTypeSpot)

	result, err = GetOrderbook("Exchange", currency, assets.AssetTypeSpot)
	if err != nil {
		t.Fatal("Test failed. TestProcessOrderbook failed to retrieve new orderbook")
	}

	if result.Pair.Pair() != currency.Pair() {
		t.Fatal("Test failed. TestProcessOrderbook result pair is incorrect")
	}

	base.Asks = []Item{{Price: 200, Amount: 200}}
	ProcessOrderbook("Exchange", currency, base, "monthly")

	result, err = GetOrderbook("Exchange", currency, "monthly")
	if err != nil {
		t.Fatal("Test failed. TestProcessOrderbook failed to retrieve new orderbook")
	}

	a, b := result.CalculateTotalAsks()
	if a != 200 && b != 40000 {
		t.Fatal("Test failed. TestProcessOrderbook CalculateTotalsAsks incorrect values")
	}

	base.Bids = []Item{{Price: 420, Amount: 200}}
	ProcessOrderbook("Blah", currency, base, "quarterly")
	result, err = GetOrderbook("Blah", currency, "quarterly")
	if err != nil {
		t.Fatal("Test failed. TestProcessOrderbook failed to create new orderbook")
	}

	if a != 200 && b != 84000 {
		t.Fatal("Test failed. TestProcessOrderbook CalculateTotalsBids incorrect values")
	}

	type quick struct {
		Name string
		P    pair.CurrencyPair
		Bids []Item
		Asks []Item
	}

	var testArray []quick

	_ = rand.NewSource(time.Now().Unix())

	var wg sync.WaitGroup
	var m sync.Mutex

	for i := 0; i < 500; i++ {
		wg.Add(1)
		go func() {
			newName := "Exchange" + strconv.FormatInt(rand.Int63(), 10)
			newPairs := pair.NewCurrencyPair("BTC"+strconv.FormatInt(rand.Int63(), 10),
				"USD"+strconv.FormatInt(rand.Int63(), 10))

			asks := []Item{{Price: rand.Float64(), Amount: rand.Float64()}}
			bids := []Item{{Price: rand.Float64(), Amount: rand.Float64()}}
			base := Base{
				Pair:         newPairs,
				CurrencyPair: newPairs.Pair().String(),
				Asks:         asks,
				Bids:         bids,
			}

			ProcessOrderbook(newName, newPairs, base, assets.AssetTypeSpot)
			m.Lock()
			testArray = append(testArray, quick{Name: newName, P: newPairs, Bids: bids, Asks: asks})
			m.Unlock()
			wg.Done()
		}()
	}
	wg.Wait()

	for _, test := range testArray {
		wg.Add(1)
		fatalErr := false
		go func(test quick) {
			result, err := GetOrderbook(test.Name, test.P, assets.AssetTypeSpot)
			if err != nil {
				fatalErr = true
				return
			}

			if result.Asks[0] != test.Asks[0] {
				t.Error("Test failed. TestProcessOrderbook failed bad values")
			}

			if result.Bids[0] != test.Bids[0] {
				t.Error("Test failed. TestProcessOrderbook failed bad values")
			}

			wg.Done()
		}(test)

		if fatalErr {
			t.Fatal("Test failed. TestProcessOrderbook failed to retrieve new orderbook")
		}
	}

	wg.Wait()
}
