package producttruth

import "testing"

func TestValidateFact(t *testing.T) {
	valid := map[string]string{
		"regular_price": "VND 1990000", "discount_percentage": "20%", "offer_valid_until": "2026-12-31",
		"dimensions": "55 x 36 x 23 cm", "weight": "2.9 kg", "capacity": "38 L", "sku": "CASE-20-BLK",
		"phone_number": "+84 901 234 567", "website": "https://example.com/product", "discount_code": "TRAVEL20",
	}
	for key, value := range valid {
		if err := ValidateFact(key, value); err != nil {
			t.Errorf("%s=%q should be valid: %v", key, value, err)
		}
	}
	if err := ValidateFact("dimensions", "about cabin size"); err == nil {
		t.Fatal("expected imprecise dimensions to fail")
	}
}

func TestGuardrails(t *testing.T) {
	if err := ValidateNoUniversalAirlineClaim("Accepted by all airlines", false); err == nil {
		t.Fatal("expected unsupported universal claim to fail")
	}
	if err := ValidateLockedFacts("Giá VND 1990000", []LockedFact{{Key: "regular_price", ExactValue: "VND 1990000"}}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateLockedFacts("Giá 1.990.000đ", []LockedFact{{Key: "regular_price", ExactValue: "VND 1990000"}}); err == nil {
		t.Fatal("expected reformatted locked fact to fail")
	}
}
