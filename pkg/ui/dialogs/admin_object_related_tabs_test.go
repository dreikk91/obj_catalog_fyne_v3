package dialogs

import "testing"

func TestParseAddressComponents_SettlementAndStreetInSinglePart(t *testing.T) {
	city, street, house, ok := parseAddressComponents("Львівська обл., смт. Брюховичі вул. Незалежності 1")
	if !ok {
		t.Fatalf("expected parse success")
	}
	if city != "Брюховичі" {
		t.Fatalf("unexpected city: %q", city)
	}
	if street != "Незалежності" {
		t.Fatalf("unexpected street: %q", street)
	}
	if house != "1" {
		t.Fatalf("unexpected house: %q", house)
	}
}

func TestBuildGeocodeQueries_ContainsStructuredSettlementQuery(t *testing.T) {
	queries := buildGeocodeQueries("Львівська обл., смт. Брюховичі вул. Незалежності 1")
	target := "вулиця Незалежності 1, Брюховичі, Україна"

	found := false
	for _, q := range queries {
		if q == target {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected query %q in %+v", target, queries)
	}
}

func TestParseAddressComponents_CityWithOblast(t *testing.T) {
	city, street, house, ok := parseAddressComponents("Львівська область, місто Львів, вул. Зелена 15")
	if !ok {
		t.Fatalf("expected parse success")
	}
	if city != "Львів" {
		t.Fatalf("unexpected city: %q", city)
	}
	if street != "Зелена" {
		t.Fatalf("unexpected street: %q", street)
	}
	if house != "15" {
		t.Fatalf("unexpected house: %q", house)
	}
}

func TestParseAddressComponents_VillageWithOblast(t *testing.T) {
	city, street, house, ok := parseAddressComponents("Львівська обл., с. Зимна Вода, вул. Шевченка 3")
	if !ok {
		t.Fatalf("expected parse success")
	}
	if city != "Зимна Вода" {
		t.Fatalf("unexpected city: %q", city)
	}
	if street != "Шевченка" {
		t.Fatalf("unexpected street: %q", street)
	}
	if house != "3" {
		t.Fatalf("unexpected house: %q", house)
	}
}

func TestParseAddressComponents_CombinedOblastAndCityInOnePart(t *testing.T) {
	city, street, house, ok := parseAddressComponents("Львівська область місто Львів вул. Зелена 15")
	if !ok {
		t.Fatalf("expected parse success")
	}
	if city != "Львів" {
		t.Fatalf("unexpected city: %q", city)
	}
	if street != "Зелена" {
		t.Fatalf("unexpected street: %q", street)
	}
	if house != "15" {
		t.Fatalf("unexpected house: %q", house)
	}
}

func TestParseAddressComponents_CityWithoutSpaceAfterDot(t *testing.T) {
	city, street, house, ok := parseAddressComponents("Львівська обл., м.Миколаїв, вул. Залізнична, 45а")
	if !ok {
		t.Fatalf("expected parse success")
	}
	if city != "Миколаїв" {
		t.Fatalf("unexpected city: %q", city)
	}
	if street != "Залізнична" {
		t.Fatalf("unexpected street: %q", street)
	}
	if house != "45а" {
		t.Fatalf("unexpected house: %q", house)
	}
}

func TestParseAddressComponents_SuffixStreetType(t *testing.T) {
	city, street, house, ok := parseAddressComponents("Львіська обл., Яворів м., Маковея вул., 62")
	if !ok {
		t.Fatalf("expected parse success")
	}
	if city != "Яворів" {
		t.Fatalf("unexpected city: %q", city)
	}
	if street != "Маковея" {
		t.Fatalf("unexpected street: %q", street)
	}
	if house != "62" {
		t.Fatalf("unexpected house: %q", house)
	}
}

func TestParseAddressComponents_AvenueAbbrevWithoutSpace(t *testing.T) {
	city, street, house, ok := parseAddressComponents("м. Львів, пр.Червоної Калини, 86а")
	if !ok {
		t.Fatalf("expected parse success")
	}
	if city != "Львів" {
		t.Fatalf("unexpected city: %q", city)
	}
	if street != "Червоної Калини" {
		t.Fatalf("unexpected street: %q", street)
	}
	if house != "86а" {
		t.Fatalf("unexpected house: %q", house)
	}
}

func TestNormalizeAddressForGeocode_StripsNoise(t *testing.T) {
	got := normalizeAddressForGeocode("м. Львів, вул. Коперніка, 16 У дворі 79013 297-69-43")
	want := "м. Львів, вул. Коперніка, 16"
	if got != want {
		t.Fatalf("unexpected normalized value: %q", got)
	}
}
