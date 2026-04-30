package kalshi

import (
	"fmt"

	"github.com/AdityaTanejaRox/hip4-prediction-gateway/internal/domain"
)

func CentsToBps(cents int64) (domain.PriceBps, error) {
	if cents < 0 || cents > 100 {
		return 0, fmt.Errorf("kalshi cent price out of range: %d", cents)
	}

	return domain.PriceBps(cents * 100), nil
}

func NoBidCentsToYesAskBps(noBidCents int64) (domain.PriceBps, error) {
	noBidBps, err := CentsToBps(noBidCents)
	if err != nil {
		return 0, err
	}

	return domain.MaxPriceBps - noBidBps, nil
}
