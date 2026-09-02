package platform

import (
	"testing"
	"time"
)

func TestWeekStart_ReturnsMondayMidnightJakarta(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	// A Wednesday.
	wed := time.Date(2026, 8, 19, 14, 30, 0, 0, loc)
	got := WeekStart(wed)
	want := time.Date(2026, 8, 17, 0, 0, 0, 0, loc) // Monday
	if !got.Equal(want) {
		t.Fatalf("WeekStart(%v) = %v, mau %v", wed, got, want)
	}
	if got.Weekday() != time.Monday {
		t.Fatalf("hasil bukan Senin: %v", got.Weekday())
	}
}

func TestWeekStart_MondayIsIdempotent(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	mon := time.Date(2026, 8, 17, 0, 0, 0, 0, loc)
	if got := WeekStart(mon); !got.Equal(mon) {
		t.Fatalf("WeekStart(Senin) = %v, mau %v", got, mon)
	}
	// Late Monday still maps to Monday midnight.
	lateMon := time.Date(2026, 8, 17, 23, 59, 0, 0, loc)
	if got := WeekStart(lateMon); !got.Equal(mon) {
		t.Fatalf("WeekStart(Senin malam) = %v, mau %v", got, mon)
	}
}

func TestWeekStart_SundayBelongsToPriorMonday(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	sun := time.Date(2026, 8, 23, 10, 0, 0, 0, loc)
	want := time.Date(2026, 8, 17, 0, 0, 0, 0, loc)
	if got := WeekStart(sun); !got.Equal(want) {
		t.Fatalf("WeekStart(Minggu) = %v, mau %v", got, want)
	}
}

func TestTestClock_SetAndAdvance(t *testing.T) {
	start := time.Date(2026, 8, 17, 8, 0, 0, 0, time.UTC)
	c := NewTestClock(start)
	c.Advance(2 * time.Hour)
	if got := c.Now().Hour(); got != 17 { // 08:00 UTC + 2h = 10:00 UTC = 17:00 WIB
		t.Fatalf("jam setelah Advance = %d, mau 17 (WIB)", got)
	}
}

func TestSystemClock_LocalizedToJakarta(t *testing.T) {
	name, _ := SystemClock{}.Now().Zone()
	if name != "WIB" {
		t.Fatalf("zona SystemClock = %q, mau WIB", name)
	}
}
