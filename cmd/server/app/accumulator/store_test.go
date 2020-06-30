package accumulator

import (
	"sync"
	"testing"

	"github.com/golang/geo/s2"
)

func BenchmarkSlice(b *testing.B) {
	s := make([]s2.CellID, 0)

	b.ReportAllocs()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s = append(s, s2.CellID(i))
	}
}

func BenchmarkMap(b *testing.B) {
	s := make(map[s2.CellID]struct{}, 0)

	b.ReportAllocs()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s[s2.CellID(i)] = struct{}{}
	}
}

type storeMap interface {
	add(s2.CellID)
	close()
}

type lockedMap struct {
	mu    sync.Mutex
	cells map[s2.CellID]struct{}
}

func newLockedMap() *lockedMap {
	return &lockedMap{
		cells: make(map[s2.CellID]struct{}),
	}
}

func (m *lockedMap) add(key s2.CellID) {
	m.mu.Lock()
	m.cells[key] = struct{}{}
	m.mu.Unlock()
}

func (m *lockedMap) close() {}

type shardedMap struct {
	n      int
	shards []*lockedMap
}

func newShardedMap(n int) *shardedMap {
	m := shardedMap{
		n:      n,
		shards: make([]*lockedMap, n),
	}
	for i := 0; i < n; i++ {
		m.shards[i] = newLockedMap()
	}
	return &m
}

func (m *shardedMap) add(key s2.CellID) {
	m.shards[int(key)%m.n].add(key)
}

func (m *shardedMap) close() {}

type channelMap struct {
	in    chan s2.CellID
	stop  chan struct{}
	wg    sync.WaitGroup
	mu    sync.Mutex
	cells map[s2.CellID]struct{}
}

func newChannelMap(n, c int) *channelMap {
	m := channelMap{
		in:    make(chan s2.CellID, n),
		stop:  make(chan struct{}),
		cells: make(map[s2.CellID]struct{}),
	}

	m.wg.Add(c)
	for i := 0; i < c; i++ {
		go m.process()
	}
	return &m
}

func (m *channelMap) add(key s2.CellID) {
	m.in <- key
}

func (m *channelMap) close() {
	close(m.stop)
	m.wg.Wait()
}

func (m *channelMap) process() {
	defer m.wg.Done()

	for {
		select {
		case key := <-m.in:
			m.mu.Lock()
			m.cells[key] = struct{}{}
			m.mu.Unlock()
		case <-m.stop:
			return
		}
	}
}

type shardedChannelMap struct {
	in     chan s2.CellID
	stop   chan struct{}
	wg     sync.WaitGroup
	mu     sync.Mutex
	shards []*lockedMap
	n      int
}

func newShardedChannelMap(s, n, c int) *shardedChannelMap {
	m := shardedChannelMap{
		in:     make(chan s2.CellID, n),
		stop:   make(chan struct{}),
		shards: make([]*lockedMap, s),
		n:      s,
	}
	for i := 0; i < s; i++ {
		m.shards[i] = newLockedMap()
	}

	m.wg.Add(c)
	for i := 0; i < c; i++ {
		go m.process()
	}
	return &m
}

func (m *shardedChannelMap) add(key s2.CellID) {
	m.in <- key
}

func (m *shardedChannelMap) close() {
	close(m.stop)
	m.wg.Wait()
}

func (m *shardedChannelMap) process() {
	defer m.wg.Done()

	for {
		select {
		case key := <-m.in:
			m.shards[int(key)%m.n].add(key)
		case <-m.stop:
			return
		}
	}
}

func BenchmarkMaps(b *testing.B) {
	const (
		shards  = 256
		bufSize = 1000
	)

	type makeMap func() storeMap

	makeLockedMap := func() storeMap { return newLockedMap() }
	makeShardedMap := func() storeMap { return newShardedMap(shards) }

	makeChannelMap := func(consumers int) makeMap {
		return func() storeMap { return newChannelMap(bufSize, consumers) }
	}

	makeShardedChannelMap := func(consumers int) makeMap {
		return func() storeMap { return newShardedChannelMap(shards, bufSize, consumers) }
	}

	benchmarks := []struct {
		name          string
		m             makeMap
		numGoroutines int
	}{
		{"LockedMap_100", makeLockedMap, 100},
		{"ShardedMap_100", makeShardedMap, 100},
		{"ChannelMap_100_1", makeChannelMap(1), 100},
		{"ChannelMap_100_2", makeChannelMap(2), 100},
		{"ChannelMap_100_3", makeChannelMap(3), 100},
		{"ShardedChannelMap_100_1", makeShardedChannelMap(1), 100},
		{"ShardedChannelMap_100_2", makeShardedChannelMap(2), 100},
		{"ShardedChannelMap_100_3", makeShardedChannelMap(3), 100},
		{"LockedMap_500", makeLockedMap, 500},
		{"ShardedMap_500", makeShardedMap, 500},
		{"ChannelMap_500_1", makeChannelMap(1), 500},
		{"ChannelMap_500_2", makeChannelMap(2), 500},
		{"ChannelMap_500_3", makeChannelMap(3), 500},
		{"ShardedChannelMap_500_1", makeShardedChannelMap(1), 500},
		{"ShardedChannelMap_500_2", makeShardedChannelMap(2), 500},
		{"ShardedChannelMap_500_3", makeShardedChannelMap(3), 500},
		{"LockedMap_1000", makeLockedMap, 1000},
		{"ShardedMap_1000", makeShardedMap, 1000},
		{"ChannelMap_1000_1", makeChannelMap(1), 1000},
		{"ChannelMap_1000_2", makeChannelMap(2), 1000},
		{"ChannelMap_1000_3", makeChannelMap(3), 1000},
		{"ShardedChannelMap_1000_1", makeShardedChannelMap(1), 1000},
		{"ShardedChannelMap_1000_2", makeShardedChannelMap(2), 1000},
		{"ShardedChannelMap_1000_3", makeShardedChannelMap(3), 1000},
		{"LockedMap_2000", makeLockedMap, 2000},
		{"ShardedMap_2000", makeShardedMap, 2000},
		{"ChannelMap_2000_1", makeChannelMap(1), 2000},
		{"ChannelMap_2000_2", makeChannelMap(2), 2000},
		{"ChannelMap_2000_3", makeChannelMap(3), 2000},
		{"ShardedChannelMap_2000_1", makeShardedChannelMap(1), 2000},
		{"ShardedChannelMap_2000_2", makeShardedChannelMap(2), 2000},
		{"ShardedChannelMap_2000_3", makeShardedChannelMap(3), 2000},
	}

	var wg sync.WaitGroup

	for _, bm := range benchmarks {
		sem := make(chan struct{}, bm.numGoroutines)
		m := bm.m()

		b.Run(bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				sem <- struct{}{}

				wg.Add(1)
				go func(key s2.CellID) {
					defer func() {
						wg.Done()
						<-sem
					}()

					m.add(key)
				}(s2.CellID(i))
			}

			wg.Wait()
		})
	}
}
