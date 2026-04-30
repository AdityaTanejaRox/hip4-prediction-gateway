package normalize

import (
	"fmt"
	"math"
	"strconv"

	"github.com/AdityaTanejaRox/hip4-prediction-gateway/internal/domain"
)

func ProbabilityFloatToBps(price float64) (domain.PriceBps, error) {
	if price < 0.0 || price > 1.0 {
		return 0, fmt.Errorf("probability price out of range: %f", price)
	}

	return domain.PriceBps(math.Round(price * 10000.0)), nil
}

func DecimalStringToBps(price string) (domain.PriceBps, error) {
	value, err := strconv.ParseFloat(price, 64)
	if err != nil {
		return 0, fmt.Errorf("parse decimal price %q: %w", price, err)
	}

	return ProbabilityFloatToBps(value)
}

func KalshiCentPriceToBps(cents int64) (domain.PriceBps, error) {
	if cents < 0 || cents > 100 {
		return 0, fmt.Errorf("Kalshi cents out of range: %d", cents)
	}

	return domain.PriceBps(cents * 100), nil
}

func NoBidToYesAsk(noBid domain.PriceBps) domain.PriceBps {
	return domain.MaxPriceBps - noBid
}

func NoAskToYesBid(noAsk domain.PriceBps) domain.PriceBps {
	return domain.MaxPriceBps - noAsk
}
