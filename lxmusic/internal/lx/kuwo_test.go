package lx

import "testing"

// TestParseHumanSize covers the live kuwo form. The unit is two letters, "52.83Mb",
// and stripping a single trailing letter is not enough: the old version fed
// "52.83Mb" to ParseFloat and every size silently came back 0, leaving the search
// table with nothing to sort on. Tolerances absorb the float rounding rather
// than pinning an artifact of it.
func TestParseHumanSize(t *testing.T) {
	for _, c := range []struct {
		in   string
		want float64
	}{
		{"52.83Mb", 52.83 * 1048576},
		{"4.12Mb", 4.12 * 1048576},
		{"10.29Mb", 10.29 * 1048576},
		{"22.11Mb", 22.11 * 1048576},
		{"384K", 384 * 1024},
		{"1.2m", 1.2 * 1048576},
		{"0Mb", 0},
		{"", 0},
		{"zpMb", 0},
		{"Mb", 0},
	} {
		got := parseHumanSize(c.in)
		want := int64(c.want)
		if got < want-1 || got > want+1 {
			t.Errorf("parseHumanSize(%q) = %d, want about %d", c.in, got, want)
		}
	}
}

// TestBestQualitySize pins what the table shows and sorts on: the size of the
// best rendition, never a lower one's.
func TestBestQualitySize(t *testing.T) {
	if got := (&Song{Qualities: []Quality{
		{Type: Q128, Size: 1024}, {Type: Q320, Size: 2048}, {Type: QFlac, Size: 52428800},
	}}).BestQualitySize(); got != 52428800 {
		t.Errorf("BestQualitySize() = %d, want the flac size", got)
	}
	// A rendition that announces no size reports 0 rather than borrowing a lower
	// rendition's number, which QualityByType would otherwise hand back.
	if got := (&Song{Qualities: []Quality{{Type: Q320, Size: 2048}, {Type: QFlac}}}).BestQualitySize(); got != 0 {
		t.Errorf("best has no size: got %d, want 0", got)
	}
	if got := (&Song{Qualities: []Quality{{Type: Q128}}}).BestQualitySize(); got != 0 {
		t.Errorf("no size announced: got %d, want 0", got)
	}
	if got := (&Song{}).BestQualitySize(); got != 0 {
		t.Errorf("no qualities: got %d, want 0", got)
	}
}
