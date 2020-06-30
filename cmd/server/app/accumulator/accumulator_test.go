package accumulator

import (
	"testing"
	"time"

	"github.com/golang/geo/s2"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name      string
		meters    float64
		wantLevel int
	}{
		{"Level_10Meters", 10, 20},
		{"Level_50Meters", 50, 18},
		{"Level_100Meters", 100, 17},
		{"Level_200Meters", 200, 16},
		{"Level_500Meters", 500, 14},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := s2.AvgEdgeMetric.ClosestLevel(test.meters / earthRadiusMeters)
			if got != test.wantLevel {
				t.Errorf("incorrect level, got = %v, want = %v", got, test.wantLevel)
			}
		})
	}
}

func TestCallback(t *testing.T) {
	const (
		edge     = 1000
		interval = 200 * time.Millisecond
		bufSize  = 3

		wantCallbacksCalled = 1
		wantEvents          = 3
		wantMax             = 3
	)

	var gotCallbacksCalled, gotEvents, gotMax int

	cb := func(r Result) {
		gotCallbacksCalled++
		gotEvents = r.Count
		gotMax = r.Max
	}

	acc, err := New(edge, interval, bufSize, cb)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	coords := [][]float64{
		{-33.85612567555415, 151.21445775032043},
		{-33.85767593977401, 151.21516585350037},
		{-33.856446422184305, 151.21599197387695},
	}
	for _, c := range coords {
		acc.Add(c[0], c[1])
	}

	time.Sleep(interval * 2)

	if gotCallbacksCalled != wantCallbacksCalled {
		t.Errorf("Callbacks called: got = %v, want = %v", gotCallbacksCalled, wantCallbacksCalled)
	}
	if gotEvents != wantEvents {
		t.Errorf("Events collected: got = %v, want = %v", gotEvents, wantEvents)
	}
	if gotMax != wantMax {
		t.Errorf("Max events per cell observed: got = %v, want = %v", gotMax, wantMax)
	}
}
