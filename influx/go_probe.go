// Copyright (c) 2022 Exograd SAS.
//
// Permission to use, copy, modify, and distribute this software for any
// purpose with or without fee is hereby granted, provided that the above
// copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
// WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR ANY
// SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
// WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
// ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF OR
// IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

package influx

import (
	"runtime"
	"time"
)

func (c *Client) goProbeMain() {
	defer c.wg.Done()

	timer := time.NewTicker(time.Second)
	defer timer.Stop()

	for {
		select {
		case <-c.stopChan:
			return

		case <-timer.C:
			now := time.Now()

			points := Points{
				goProbeGoroutinePoint(now),
				goProbeMemPoint(now),
			}

			c.EnqueuePoints(points)
		}
	}
}

func goProbeGoroutinePoint(now time.Time) *Point {
	fields := Fields{
		"count": runtime.NumGoroutine(),
	}

	return NewPointWithTimestamp("go_goroutines", Tags{}, fields, now)
}

func goProbeMemPoint(now time.Time) *Point {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	fields := Fields{
		"heap_alloc":    stats.HeapAlloc,
		"heap_sys":      stats.HeapSys,
		"heap_idle":     stats.HeapIdle,
		"heap_in_use":   stats.HeapInuse,
		"heap_released": stats.HeapReleased,

		"stack_in_use": stats.StackInuse,
		"stack_sys":    stats.StackSys,

		"nb_gcs":               stats.NumGC,
		"gc_cpu_time_fraction": stats.GCCPUFraction,
	}

	return NewPointWithTimestamp("go_memory", Tags{}, fields, now)
}
