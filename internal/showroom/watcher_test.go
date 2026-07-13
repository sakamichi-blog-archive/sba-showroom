package showroom

import (
	"testing"
	"time"
)

func TestWatchPollInterval(t *testing.T) {
	tests := []struct {
		name    string
		offset  time.Duration // positive = future, negative = past
		want    time.Duration
	}{
		{"no schedule", 0, 20 * time.Second},
		{"future", 10 * time.Minute, 20 * time.Second},
		{"2 min overdue", -2 * time.Minute, 4 * time.Second},
		{"6 min overdue", -6 * time.Minute, 8 * time.Second},
		{"25 min overdue", -25 * time.Minute, 20 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ts int64
			if tt.offset != 0 {
				ts = time.Now().Add(tt.offset).Unix()
			}
			got := watchPollInterval(ts)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
