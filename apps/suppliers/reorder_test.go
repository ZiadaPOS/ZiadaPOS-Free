package suppliers

import "testing"

func TestCalculateReorderMatchesFoundationExample(t *testing.T) {
	dailyVelocity, leadTimeDemand, suggested := calculateReorder(300, 5, 20, 18)
	if dailyVelocity != 10 || leadTimeDemand != 50 || suggested != 52 {
		t.Fatalf("got velocity=%d demand=%d suggested=%d; want 10, 50, 52", dailyVelocity, leadTimeDemand, suggested)
	}
}

func TestCalculateReorderNeverSuggestsNegativeQuantity(t *testing.T) {
	_, _, suggested := calculateReorder(0, 5, 2, 20)
	if suggested != 0 {
		t.Fatalf("got suggested=%d; want 0", suggested)
	}
}
