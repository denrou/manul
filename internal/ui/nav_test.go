package ui

import (
	"fmt"
	"math"
	"testing"
)

func TestPushCapped(t *testing.T) {
	var stack []histEntry
	for i := 0; i < historyCap+10; i++ {
		stack = pushCapped(stack, histEntry{url: fmt.Sprintf("u%d", i)})
	}
	if len(stack) != historyCap {
		t.Fatalf("stack length = %d, want %d", len(stack), historyCap)
	}
	if stack[0].url != "u10" {
		t.Errorf("oldest entry = %s, want u10 (oldest dropped first)", stack[0].url)
	}
	if stack[len(stack)-1].url != fmt.Sprintf("u%d", historyCap+9) {
		t.Errorf("newest entry = %s", stack[len(stack)-1].url)
	}
}

func TestClampFraction(t *testing.T) {
	cases := []struct {
		p    float64
		fits bool
		want float64
	}{
		{0.5, false, 0.5},
		{0.5, true, 0}, // content fits: always 0
		{math.NaN(), false, 0},
		{-0.2, false, 0},
		{1.7, false, 1},
	}
	for _, c := range cases {
		if got := clampFraction(c.p, c.fits); got != c.want {
			t.Errorf("clampFraction(%v, %v) = %v, want %v", c.p, c.fits, got, c.want)
		}
	}
}

func TestRestoreOffset(t *testing.T) {
	cases := []struct {
		frac          float64
		total, height int
		want          int
	}{
		{0.5, 100, 20, 40}, // round(0.5 * 80)
		{1, 100, 20, 80},
		{0, 100, 20, 0},
		{0.5, 10, 20, 0}, // content fits: scrollable clamps to 0
		{0.333, 50, 10, 13},
	}
	for _, c := range cases {
		if got := restoreOffset(c.frac, c.total, c.height); got != c.want {
			t.Errorf("restoreOffset(%v, %d, %d) = %d, want %d", c.frac, c.total, c.height, got, c.want)
		}
	}
}

func TestDisplayHost(t *testing.T) {
	cases := []struct{ in, want string }{
		{"docs.anthropic.com", "docs.anthropic.com"},
		{"https://example.com/path.md", "example.com"},
		{"  bun.sh  ", "bun.sh"},
	}
	for _, c := range cases {
		if got := displayHost(c.in); got != c.want {
			t.Errorf("displayHost(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestHostOf(t *testing.T) {
	if got := hostOf("https://example.com/a/b.md"); got != "example.com" {
		t.Errorf("hostOf = %q", got)
	}
	if got := hostOf("manul:start"); got != "" {
		t.Errorf("hostOf(internal) = %q, want empty", got)
	}
}
