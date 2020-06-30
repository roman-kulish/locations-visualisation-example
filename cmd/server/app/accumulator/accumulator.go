package accumulator

import (
	"errors"
	"time"

	"github.com/golang/geo/s2"
)

const earthRadiusMeters = 6371010.0

// OnResultFunc function is called when accumulator has a result ready.
type OnResultFunc func(Result)

// Result represents accumulated result.
type Result struct {
	Lat, Lng float64

	// Count is a number of location events received for a single cell.
	Count int

	// Max is a maximum number of location events seen during the window.
	Max int
}

type item struct {
	key s2.CellID
	pt  s2.Point
}

// Accumulator is a locations accumulator.
type Accumulator struct {
	store *store
	cb    OnResultFunc
	buf   chan item
	stop  chan struct{}
	level int
}

// New creates and returns an instance of Accumulator configured
// to accumulate location events into an s2.CellID with given
// approximate cellEdgeLength and send results out at the end of
// window duration.
func New(cellEdgeLength float64, window time.Duration, bufSize int, cb OnResultFunc) (*Accumulator, error) {
	if cellEdgeLength <= 0 {
		return nil, errors.New("accumulator: negative cellEdgeLength")
	}
	if window <= 0 {
		return nil, errors.New("accumulator: negative window duration")
	}
	acc := Accumulator{
		store: newStore(window),
		cb:    cb,
		buf:   make(chan item, bufSize),
		stop:  make(chan struct{}),
		level: s2.AvgEdgeMetric.ClosestLevel(cellEdgeLength / earthRadiusMeters),
	}
	go acc.process(window)
	return &acc, nil
}

// Add adds a data point to the accumulator.
func (a *Accumulator) Add(lat, lng float64) {
	ll := s2.LatLngFromDegrees(lat, lng)
	item := item{
		key: s2.CellIDFromLatLng(ll).Parent(a.level),
		pt:  s2.PointFromLatLng(ll),
	}
	select {
	case a.buf <- item:
	default: // drop item, because the buffer is full
	}
}

// Stop accumulator.
func (a *Accumulator) Stop() {
	close(a.stop)
}

// process incoming and accumulated data.
func (a *Accumulator) process(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case item := <-a.buf:
			a.store.add(item.key, item.pt)
		case <-ticker.C:
			a.store.collect(a.cb)
		}
	}
}
