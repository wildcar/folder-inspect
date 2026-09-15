package i18n

import "testing"

func TestAllKeysTranslated(t *testing.T) {
	for _, k := range Keys() {
		if !Has(EN, k) {
			t.Errorf("key %q missing in EN", k)
		}
	}
	for k := range messages[EN] {
		if !Has(RU, k) {
			t.Errorf("key %q missing in RU", k)
		}
	}
}

func TestSetAndFallback(t *testing.T) {
	t.Cleanup(func() { Set(RU) })
	Set("EN ")
	if Lang() != EN || T("cat.junk") != "Junk" {
		t.Errorf("EN not selected: %q %q", Lang(), T("cat.junk"))
	}
	Set("de")
	if Lang() != RU {
		t.Error("unknown language must fall back to RU")
	}
	if T("no.such.key") != "no.such.key" {
		t.Error("missing key must return the key")
	}
}

func TestDetect(t *testing.T) {
	t.Setenv("LC_ALL", "")
	t.Setenv("LANGUAGE", "")
	t.Setenv("LANG", "")
	if Detect() != RU {
		t.Error("unset locale must mean RU")
	}
	t.Setenv("LANG", "en_US.UTF-8")
	if Detect() != EN {
		t.Error("en_US must mean EN")
	}
}
