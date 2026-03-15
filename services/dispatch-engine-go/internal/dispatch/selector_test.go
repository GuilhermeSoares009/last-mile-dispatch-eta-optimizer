package dispatch

import "testing"

func TestSelectCourierFallbackDeterministic(t *testing.T) {
	couriers := []Courier{
		{ID: "c-2", EtaMinutes: 12, Availability: 0.1, Load: 0.4},
		{ID: "c-1", EtaMinutes: 12, Availability: 0.1, Load: 0.4},
		{ID: "c-3", EtaMinutes: 15, Availability: 0.2, Load: 0.2},
	}

	first, err := SelectCourier(couriers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !first.Fallback {
		t.Fatalf("expected fallback decision")
	}

	for i := 0; i < 20; i++ {
		next, callErr := SelectCourier(couriers)
		if callErr != nil {
			t.Fatalf("unexpected error in iteration %d: %v", i, callErr)
		}
		if next.Courier.ID != first.Courier.ID {
			t.Fatalf("non-deterministic courier in iteration %d: got %s want %s", i, next.Courier.ID, first.Courier.ID)
		}
	}
}

func TestSelectCourierEmptyInput(t *testing.T) {
	_, err := SelectCourier(nil)
	if err != ErrNoCouriers {
		t.Fatalf("expected ErrNoCouriers, got %v", err)
	}
}
