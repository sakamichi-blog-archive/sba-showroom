package cmd

import (
	"testing"
	"time"
)

func TestParseExpectedTime_HHmm(t *testing.T) {
	now := time.Now()
	got, err := parseExpectedTime("14:30")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Hour() != 14 || got.Minute() != 30 {
		t.Errorf("got %v, want 14:30", got)
	}
	if got.Year() != now.Year() || got.Month() != now.Month() || got.Day() != now.Day() {
		t.Errorf("date should be today, got %v", got)
	}
}

func TestParseExpectedTime_YYYYMMDDHHmm(t *testing.T) {
	got, err := parseExpectedTime("2024-03-15 17:30")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Year() != 2024 || got.Month() != 3 || got.Day() != 15 {
		t.Errorf("date: got %v", got)
	}
	if got.Hour() != 17 || got.Minute() != 30 {
		t.Errorf("time: got %v", got)
	}
}

func TestParseExpectedTime_SlashFormat(t *testing.T) {
	got, err := parseExpectedTime("2024/03/15 17:30")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Year() != 2024 || got.Month() != 3 || got.Day() != 15 {
		t.Errorf("date: got %v", got)
	}
}

func TestParseExpectedTime_Invalid(t *testing.T) {
	cases := []string{"not-a-time", "25:99", "2024-03-15", ""}
	for _, c := range cases {
		_, err := parseExpectedTime(c)
		if err == nil {
			t.Errorf("parseExpectedTime(%q): expected error, got nil", c)
		}
	}
}
