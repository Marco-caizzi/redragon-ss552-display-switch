package display

import "testing"

func TestStepWrapsBothWays(t *testing.T) {
	cases := []struct {
		from  Mode
		ticks int
		want  Mode
	}{
		{Internal, 1, Clone},
		{Extend, 1, External},
		{External, 1, Internal},
		{Internal, -1, External},
		{Extend, -2, Internal},
		{Clone, 9, Extend},
		{Unknown, 1, External},
		{Unknown, -1, Clone},
	}
	for _, c := range cases {
		if got := c.from.Step(c.ticks); got != c.want {
			t.Errorf("%s.Step(%d) = %s, want %s", c.from, c.ticks, got, c.want)
		}
	}
}
