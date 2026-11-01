package base

import "testing"

func TestListed(t *testing.T) {
	const have = "C\nC.utf8\nen_US.utf8\nde_DE@euro\nPOSIX\n"
	for name, want := range map[string]bool{
		"en_US.UTF-8": true,
		"C.UTF-8":     true,
		"de_DE@euro":  true,
		"fr_FR.UTF-8": false,
		"en_US":       false,
	} {
		if got := listed(have, name); got != want {
			t.Errorf("listed(%q) = %v, want %v", name, got, want)
		}
	}
	if !listed("en_GB.iso88591\n", "en_GB.ISO-8859-1") || !listed("xx_YY.iso885915\n", "xx_YY.8859-15") {
		t.Error("a code set is not matched as glibc names it")
	}
}

func TestEntry(t *testing.T) {
	for name, want := range map[string]string{
		"en_US.UTF-8": "en_US.UTF-8 UTF-8", "de_DE.ISO-8859-15@euro": "de_DE.ISO-8859-15@euro ISO-8859-15",
	} {
		if got := entry(name); got != want {
			t.Errorf("entry(%q) = %q, want %q", name, got, want)
		}
	}
}
