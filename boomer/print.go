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
	"os"
	"sort"
	"time"
)

const (
	barChar = "∎"
)

type report struct {
	avgTotal float64
	fastest  float64
	slowest  float64
	average  float64
	rps      float64

	results chan *result
	total   time.Duration

	errorDist      map[string]int
	statusCodeDist map[int]int
	lats           []float64
	sizeTotal      int64

	output string
}

func printReport(size int, results chan *result, output string, total time.Duration) {
	r := &report{
		output:         output,
		results:        results,
		total:          total,
		statusCodeDist: make(map[int]int),
		errorDist:      make(map[string]int),
	}
	r.finalize()
}

func (r *report) finalize() {
	for {
		select {
		case res := <-r.results:
			if res.err != nil {
				r.errorDist[res.err.Error()]++
			} else {
				r.lats = append(r.lats, res.duration.Seconds())
				r.avgTotal += res.duration.Seconds()
				r.statusCodeDist[res.statusCode]++
				if res.contentLength > 0 {
					r.sizeTotal += res.contentLength
				}
			}
		default:
			r.rps = float64(len(r.lats)) / r.total.Seconds()
			r.average = r.avgTotal / float64(len(r.lats))
			sort.Float64s(r.lats)
			if len(r.lats) > 0 {
				r.fastest = r.lats[0]
				r.slowest = r.lats[len(r.lats)-1]
			}
			r.print()
			return
		}
	}
}

func (r *report) print() {
	var printer Printer

	if r.output == "csv" {
		printer = CSVPrinter{
			Writer: os.Stdout,
		}
	} else {
		printer = DetailedPrinter{
			Writer: os.Stdout,
		}
	}

	printer.Print(*r)
}
