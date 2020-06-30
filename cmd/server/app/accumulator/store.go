package accumulator

import (
	"time"

	"github.com/golang/geo/s2"
)

// cell is an internal accumulator result, which is used to accumulate
// s2.Points and computer "center of gravity" for them.
type cell struct {
	pt s2.Point
	n  int
}

// add adds an s2.Point to this cell.
func (c *cell) add(pt s2.Point) int {
	c.pt.X += pt.X
	c.pt.Y += pt.Y
	c.pt.Z += pt.Z
	c.n++
	return c.n
}

// result returns accumulated result.
func (c *cell) result() Result {
	c.pt.X /= float64(c.n)
	c.pt.Y /= float64(c.n)
	c.pt.Z /= float64(c.n)

	ll := s2.LatLngFromPoint(c.pt)
	return Result{
		Lat:   ll.Lat.Degrees(),
		Lng:   ll.Lng.Degrees(),
		Count: c.n,
	}
}

// store is an accumulator data store.
type store struct {
	cells      map[s2.CellID]cell
	maxBuckets map[int64]int
	ttlBuckets map[int64][]s2.CellID
	ttl        time.Duration
}

// newStore creates and returns a new store instance.
func newStore(ttl time.Duration) *store {
	return &store{
		cells:      make(map[s2.CellID]cell),
		maxBuckets: make(map[int64]int),
		ttlBuckets: make(map[int64][]s2.CellID),
		ttl:        ttl,
	}
}

// add adds a point to the cell associated with the key.
func (s *store) add(key s2.CellID, pt s2.Point) {
	ts := s.bucket()

	cell, ok := s.cells[key]
	n := cell.add(pt)
	s.cells[key] = cell

	if !ok {
		s.ttlBuckets[ts] = append(s.ttlBuckets[ts], key)
	}
	if n > s.maxBuckets[ts] {
		s.maxBuckets[ts] = n
	}
}

// del deletes cell associated with the key and its Result.
func (s *store) del(key s2.CellID) Result {
	cell := s.cells[key]
	delete(s.cells, key)
	return cell.result()
}

// collect collects and returns accumulated results via cb function.
func (s *store) collect(cb OnResultFunc) {
	now := time.Now().UnixNano()
	for ts, keys := range s.ttlBuckets {
		if ts >= now {
			continue
		}
		for _, key := range keys {
			r := s.del(key)
			r.Max = s.maxBuckets[ts]
			if cb != nil {
				cb(r)
			}
		}
		delete(s.maxBuckets, ts)
		delete(s.ttlBuckets, ts)
	}
}

func (s *store) bucket() int64 {
	return time.Now().Truncate(s.ttl).Add(s.ttl).UnixNano()
}
