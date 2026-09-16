package ansi

import (
	"math"
	"strings"
	"testing"
)

func TestBar(t *testing.T) {
	if got := Bar(10, 0.5, true); !strings.Contains(got, "#####") || !strings.Contains(got, ".....") || strings.Contains(got, "######") {
		t.Fatalf("half bar: %q", got)
	}
	for _, f := range []float64{math.NaN(), -1, 0} {
		if got := Bar(4, f, true); strings.Contains(got, "#") || !strings.Contains(got, "....") {
			t.Fatalf("empty bar for %v: %q", f, got)
		}
	}
	if got := Bar(4, 7, true); !strings.Contains(got, "####") {
		t.Fatalf("overflow must clamp: %q", got)
	}
	if Bar(0, 1, true) != "" || Orange("") != "" {
		t.Fatal("empty inputs must stay empty")
	}
}
