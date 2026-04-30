package kalshi

import (
	"testing"

	"github.com/AdityaTanejaRox/hip4-prediction-gateway/internal/domain"
)

func TestCentsToBps(t *testing.T) {
	tests := []struct {
		name    string
		cents   int64
		want    domain.PriceBps
		wantErr bool
	}{
		{name: "zero cents", cents: 0, want: 0},
		{name: "fifty three cents", cents: 53, want: 5300},
		{name: "one hundred cents", cents: 100, want: 10000},
		{name: "negative invalid", cents: -1, wantErr: true},
		{name: "above hundred invalid", cents: 101, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CentsToBps(tt.cents)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestNoBidCentsToYesAskBps(t *testing.T) {
	tests := []struct {
		name       string
		noBidCents int64
		want       domain.PriceBps
	}{
		{
			name:       "NO bid 47 implies YES ask 53",
			noBidCents: 47,
			want:       5300,
		},
		{
			name:       "NO bid 1 implies YES ask 99",
			noBidCents: 1,
			want:       9900,
		},
		{
			name:       "NO bid 99 implies YES ask 1",
			noBidCents: 99,
			want:       100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NoBidCentsToYesAskBps(tt.noBidCents)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}
