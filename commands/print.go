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

package commands

import (
	"fmt"
	"sort"
	"strings"
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

	statusCodeDist map[int]int
	lats           []float64
	latsById       []float64

	output string
}

func newReport(size int, results chan *result, output string) *report {
	return &report{
		statusCodeDist: make(map[int]int),
		latsById:       make([]float64, size),
		results:        results,
		output:         output,
	}
}

func (r *report) finalize(total time.Duration) {
	for {
		select {
		case res := <-r.results:
			r.lats = append(r.lats, res.duration.Seconds())
			r.avgTotal += res.duration.Seconds()
			r.statusCodeDist[res.statusCode]++
			r.latsById[res.id] = res.duration.Seconds()
		// default is executed when results channel is empty.
		default:
			r.total = total
			r.rps = float64(len(r.lats)) / r.total.Seconds()
			r.average = r.avgTotal / float64(len(r.lats))
			r.print()
			return
		}
	}
}

func (r *report) print() {
	if len(r.lats) > 0 {
		sort.Float64s(r.lats)

		if r.output == "csv" {
			r.printCSV()
			return
		}
		r.fastest = r.lats[0]
		r.slowest = r.lats[len(r.lats)-1]
		fmt.Printf("\nSummary:\n")
		fmt.Printf("  Total:\t%4.4f secs.\n", r.total.Seconds())
		fmt.Printf("  Slowest:\t%4.4f secs.\n", r.slowest)
		fmt.Printf("  Fastest:\t%4.4f secs.\n", r.fastest)
		fmt.Printf("  Average:\t%4.4f secs.\n", r.average)
		fmt.Printf("  Requests/sec:\t%4.4f\n", r.rps)
		r.printStatusCodes()
		r.printLatencyGraph()
		r.printHistogram()
		r.printLatencies()
	}
}

func (r *report) printCSV() {
	for i, val := range r.lats {
		fmt.Printf("%v,%4.4f\n", i+1, val)
	}
}

// Prints percentile latencies.
func (r *report) printLatencies() {
	pctls := []int{10, 25, 50, 75, 90, 95, 99}
	data := make([]float64, len(pctls))
	j := 0
	for i := 0; i < len(r.lats) && j < len(pctls); i++ {
		current := i * 100 / len(r.lats)
		if current >= pctls[j] {
			data[j] = r.lats[i]
			j++
		}
	}
	fmt.Printf("\nLatency distribution:\n")
	for i := 0; i < len(pctls); i++ {
		if data[i] > 0 {
			fmt.Printf("  %v%% in %4.4f secs.\n", pctls[i], data[i])
		}
	}
}

func (r *report) printHistogram() {
	bc := 10
	buckets := make([]float64, bc+1)
	counts := make([]int, bc+1)
	bs := (r.slowest - r.fastest) / float64(bc)
	for i := 0; i < bc; i++ {
		buckets[i] = r.fastest + bs*float64(i)
	}
	buckets[bc] = r.slowest
	var bi int
	var max int
	for i := 0; i < len(r.lats); {
		if r.lats[i] <= buckets[bi] {
			i++
			counts[bi]++
			if max < counts[bi] {
				max = counts[bi]
			}
		} else if bi < len(buckets)-1 {
			bi++
		}
	}
	fmt.Printf("\nResponse time histogram:\n")
	for i := 0; i < len(buckets); i++ {
		// Normalize bar lengths.
		var barLen int
		if max > 0 {
			barLen = counts[i] * 40 / max
		}
		fmt.Printf("  %4.3f [%v]\t|%v\n", buckets[i], counts[i], strings.Repeat(barChar, barLen))
	}
}

const (
	rows = 20
	cols = 50
)

func (r *report) printLatencyGraph() {
	sampleCnt := len(r.latsById)
	yNorm := float64(rows) / r.slowest
	xNorm := float64(cols) / float64(sampleCnt)
	var graph [rows + 1][cols + 1]int
	for i := 0; i < len(r.latsById); i++ {
		y := r.latsById[i] * yNorm
		x := float64(i) * xNorm
		fmt.Printf("%v. %5.3f %v %v\n", i, r.latsById[i], int(x), int(y))
		graph[rows-int(y)][int(x)]++
	}
	fmt.Printf("\nLatency of Requests:\n")
	maxSamples := float64(sampleCnt) / float64(cols)
	tiny := int(maxSamples/5.0) + 1
	medium := tiny + int(maxSamples/3.0)
	for i := 0; i < rows; i++ {
		fmt.Printf("  %5.3f |", float64(rows-i)/yNorm)
		for j := 0; j < cols; j++ {
			val := graph[i][j]
			if val == 0 {
				fmt.Printf(" ")
			} else if val <= tiny {
				fmt.Printf(".")
			} else if val <= medium {
				fmt.Printf("-")
			} else {
				fmt.Printf("x")
			}
		}
		fmt.Println("")
	}
	fmt.Printf("        %v\n", strings.Repeat("¯", cols))
}

// Prints status code distribution.
func (r *report) printStatusCodes() {
	fmt.Printf("\nStatus code distribution:\n")
	for code, num := range r.statusCodeDist {
		fmt.Printf("  [%d]\t%d responses\n", code, num)
	}
}
