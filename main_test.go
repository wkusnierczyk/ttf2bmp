package main

import (
	"math"
	"testing"
)

func TestStrokeSuffix(t *testing.T) {
	cases := map[float64]string{0: "", 1: "-stroke1", 0.77: "-stroke0p77", 1.25: "-stroke1p25", 2: "-stroke2"}
	for w, want := range cases {
		if got := strokeSuffix(w); got != want {
			t.Errorf("strokeSuffix(%v) = %q, want %q", w, got, want)
		}
	}
}

func TestValidateStroke(t *testing.T) {
	for _, w := range []float64{0, 0.5, 1, 3.25} {
		cfg, err := validateInputs("f.ttf", "12", "A", ".", "png", 2, "none", w)
		if err != nil || cfg.Stroke != w {
			t.Errorf("stroke %v: got %v, %v", w, cfg.Stroke, err)
		}
	}
	for _, w := range []float64{-1, math.NaN(), math.Inf(1)} {
		if _, err := validateInputs("f.ttf", "12", "A", ".", "png", 2, "none", w); err == nil {
			t.Errorf("stroke %v: accepted, want an error", w)
		}
	}
}
