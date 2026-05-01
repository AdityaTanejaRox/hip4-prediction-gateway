package hip4

import "testing"

func TestNormalizeHyperliquidPriceToOutcomeBpsTrueOutcomePrice(t *testing.T) {
	tests := []struct {
		name  string
		price float64
		want  int64
	}{
		{name: "zero", price: 0.0, want: 0},
		{name: "half", price: 0.5, want: 5000},
		{name: "one", price: 1.0, want: 10000},
		{name: "precise", price: 0.5321, want: 5321},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeHyperliquidPriceToOutcomeBps(tt.price, false)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if int64(got) != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestNormalizeHyperliquidPriceRejectsNonOutcomePriceWithoutSyntheticMode(t *testing.T) {
	_, err := NormalizeHyperliquidPriceToOutcomeBps(65000.0, false)
	if err == nil {
		t.Fatalf("expected error for non-outcome price without synthetic mode")
	}
}

func TestNormalizeHyperliquidPriceAllowsSyntheticMode(t *testing.T) {
	got, err := NormalizeHyperliquidPriceToOutcomeBps(65000.0, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got < 0 || got > 10000 {
		t.Fatalf("synthetic bps out of range: %d", got)
	}
}
