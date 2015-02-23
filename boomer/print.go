// Copyright 2014 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package boomer

import (
	"sort"
	"time"
)

const (
	barChar = "∎"
)

type Report struct {
	AvgTotal float64
	Fastest  float64
	Slowest  float64
	Average  float64
	Rps      float64

	results chan *result
	Total   time.Duration

	ErrorDist      map[string]int
	StatusCodeDist map[int]int
	Lats           []float64
	SizeTotal      int64

	printer Printer
}

func printReport(size int, results chan *result, printer Printer, total time.Duration) {
	r := &Report{
		printer:        printer,
		results:        results,
		Total:          total,
		StatusCodeDist: make(map[int]int),
		ErrorDist:      make(map[string]int),
	}
	r.finalize()
}

func (r *Report) finalize() {
	for {
		select {
		case res := <-r.results:
			if res.err != nil {
				r.ErrorDist[res.err.Error()]++
			} else {
				r.Lats = append(r.Lats, res.duration.Seconds())
				r.AvgTotal += res.duration.Seconds()
				r.StatusCodeDist[res.statusCode]++
				if res.contentLength > 0 {
					r.SizeTotal += res.contentLength
				}
			}
		default:
			r.Rps = float64(len(r.Lats)) / r.Total.Seconds()
			r.Average = r.AvgTotal / float64(len(r.Lats))
			sort.Float64s(r.Lats)
			if len(r.Lats) > 0 {
				r.Fastest = r.Lats[0]
				r.Slowest = r.Lats[len(r.Lats)-1]
			}
			r.print()
			return
		}
	}
}

func (r *Report) print() {
	r.printer.Print(*r)
}
