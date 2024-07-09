package pd

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/pingcap/log"
	"go.uber.org/zap"
)

type Histogram struct {
	buckets   []int
	sum       float64
	sumSquare float64
	count     int
	interval  float64
	cutoff    float64
}

func NewHistogram(interval float64, bucketsCount int, cutoff float64) *Histogram {
	return &Histogram{
		buckets:  make([]int, bucketsCount),
		interval: interval,
		count:    0,
		cutoff:   cutoff,
	}
}

func (h *Histogram) Observe(value float64) {
	if value >= h.cutoff {
		return
	}

	index := int(value / h.interval)
	for index >= len(h.buckets) {
		h.buckets = append(h.buckets, 0)
	}

	h.buckets[index]++
	h.count++
	h.sum += value
	h.sumSquare += value * value
}

func (h *Histogram) GetPercentile(p float64) float64 {
	if h.count == 0 {
		return 0
	}
	limit := float64(h.count) * p
	result := 0.
	for i := 0; i < len(h.buckets); i += 1 {
		samplesInBucket := float64(h.buckets[i])
		if samplesInBucket >= limit {
			result += limit / samplesInBucket * h.interval
			break
		}
		result += h.interval
		limit -= samplesInBucket
	}
	return result
}

func (h *Histogram) GetAvg() float64 {
	return h.sum / float64(h.count)
}

func (h *Histogram) String() string {
	sb := &strings.Builder{}
	_, err := fmt.Fprintf(sb, "{ count: %v, sum: %v, sum_square: %v, interval: %v, buckets.len: %v, buckets: [", h.count, h.sum, h.sumSquare, h.interval, len(h.buckets))
	if err != nil {
		panic("unreachable")
	}

	if len(h.buckets) > 0 {
		put := func(value, count int) {
			if count == 1 {
				_, err = fmt.Fprintf(sb, "%v;", value)
			} else {
				_, err = fmt.Fprintf(sb, "%v,%v;", value, count)
			}
			if err != nil {
				panic("unreachable")
			}
		}

		lastValue := h.buckets[0]
		lastValueCount := 1

		for i := 1; i < len(h.buckets); i++ {
			if h.buckets[i] == lastValue {
				lastValueCount++
				continue
			}

			put(lastValue, lastValueCount)
			lastValue = h.buckets[i]
			lastValueCount = 1
		}

		put(lastValue, lastValueCount)
	}

	_, err = sb.WriteString("] }")
	if err != nil {
		panic("unreachable")
	}

	return sb.String()
}

func (h *Histogram) Clear() {
	h.sum = 0
	h.sumSquare = 0
	h.count = 0
	for i := 0; i < len(h.buckets); i++ {
		h.buckets[i] = 0
	}
}

type AutoDumpHistogram struct {
	name                  string
	mainHistogram         *Histogram
	backHistogram         *Histogram
	accumulated           *Histogram
	isDumping             atomic.Bool
	lastDumpHistogramTime time.Time
	dumpInterval          time.Duration
}

func NewAutoDumpingHistogram(name string, interval float64, bucketsCount int, cutoff float64, dumpInterval time.Duration) *AutoDumpHistogram {
	return &AutoDumpHistogram{
		name:                  name,
		mainHistogram:         NewHistogram(interval, bucketsCount, cutoff),
		backHistogram:         NewHistogram(interval, bucketsCount, cutoff),
		accumulated:           NewHistogram(interval, bucketsCount, cutoff),
		lastDumpHistogramTime: time.Now(),
		dumpInterval:          dumpInterval,
	}
}

func (h *AutoDumpHistogram) Observe(value float64, now time.Time) {
	// Not thread-safe.
	h.mainHistogram.Observe(value)
	if now.Sub(h.lastDumpHistogramTime) >= h.dumpInterval && !h.isDumping.Load() {
		h.isDumping.Store(true)
		h.mainHistogram, h.backHistogram = h.backHistogram, h.mainHistogram
		h.lastDumpHistogramTime = now
		go h.dump(now)
	}
}

func (h *AutoDumpHistogram) dump(now time.Time) {
	defer h.isDumping.Store(false)

	h.accumulated.sum += h.backHistogram.sum
	h.accumulated.sumSquare += h.backHistogram.sumSquare
	h.accumulated.count += h.backHistogram.count
	for i := 0; i < len(h.accumulated.buckets) && i < len(h.backHistogram.buckets); i++ {
		h.accumulated.buckets[i] += h.backHistogram.buckets[i]
	}
	if len(h.backHistogram.buckets) > len(h.accumulated.buckets) {
		h.accumulated.buckets = append(h.accumulated.buckets, h.backHistogram.buckets[len(h.accumulated.buckets):]...)
	}

	log.Info("dumping histogram", zap.String("name", h.name), zap.Time("time", now), zap.Stringer("histogram", h.accumulated))

	h.backHistogram.Clear()
}
