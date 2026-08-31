package main

import "testing"

func TestHardBounceAddsAddressToSuppressionLog(t *testing.T) {
	log := NewSuppressionLog()
	if !IsHardBounce([]EmailEvent{{Event: "hard_bounce"}}) {
		t.Fatal("expected hard bounce")
	}
	log.Add("Driver@Example.com")
	if !log.Contains("driver@example.com") {
		t.Fatal("address was not suppressed")
	}
}
