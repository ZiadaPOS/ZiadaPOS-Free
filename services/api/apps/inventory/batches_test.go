package inventory

import (
	"testing"
	"time"
)

func TestParseMedicineExpiryDateUsesTanzaniaCalendarDay(t *testing.T) {
	location, err := time.LoadLocation("Africa/Dar_es_Salaam")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.September, 30, 1, 0, 0, 0, location)
	for _, date := range []string{"2026-09-30", "2026-10-01"} {
		if _, err := ParseMedicineExpiryDate(date, now); err != nil {
			t.Errorf("valid expiry %s was rejected: %v", date, err)
		}
	}
	if _, err := ParseMedicineExpiryDate("2026-09-29", now); err == nil {
		t.Fatal("already-expired batch was accepted")
	}
	if _, err := ParseMedicineExpiryDate("not-a-date", now); err == nil {
		t.Fatal("invalid expiry date was accepted")
	}
}
