package boomer

import (
	"fmt"
	"io"
	"strings"
)

// Printer defines an interface to print a report
type Printer interface {
	// Print receives a report and prints it
	Print(Report) error
	// Defines whether or not it should display the progress bar when running
	// the benchmark
	ShouldDisplayProgressBar() bool
}

// CSVPrinter is an implementation of Printer to print the report in csv format
type CSVPrinter struct {
	Writer io.Writer
}

// Print receives a report and prints it
func (c CSVPrinter) Print(r Report) error {
	for i, val := range r.Lats {
		c.Writer.Write([]byte(fmt.Sprintf("%v,%4.4f\n", i+1, val)))
	}
	return nil
}

// Returns if it displays the progress bar or not when running the benchmark
func (c CSVPrinter) ShouldDisplayProgressBar() bool {
	return false
}

// DetailedPrinter is an implementation of Printer to print the details
// of a report. It prints summary, status codes, histogram, latencies and errors
type DetailedPrinter struct {
	Writer io.Writer
	r      Report
}

// Print receives a report and prints it
func (d DetailedPrinter) Print(r Report) error {
	d.r = r

	d.printSummary()
	d.printStatusCodes()
	d.printHistogram()
	d.printLatencies()

	if len(d.r.ErrorDist) > 0 {
		d.printErrors()
	}
	return nil
}

// Returns if it displays the progress bar or not when running the benchmark
func (d DetailedPrinter) ShouldDisplayProgressBar() bool {
	return true
}

func (d DetailedPrinter) printSummary() {
	fmt.Fprintf(d.Writer, "\nSummary:\n")
	fmt.Fprintf(d.Writer, "  Total:\t%4.4f secs.\n", d.r.Total.Seconds())
	fmt.Fprintf(d.Writer, "  Slowest:\t%4.4f secs.\n", d.r.Slowest)
	fmt.Fprintf(d.Writer, "  Fastest:\t%4.4f secs.\n", d.r.Fastest)
	fmt.Fprintf(d.Writer, "  Average:\t%4.4f secs.\n", d.r.Average)
	fmt.Fprintf(d.Writer, "  Requests/sec:\t%4.4f\n", d.r.Rps)
	if d.r.SizeTotal > 0 {
		fmt.Fprintf(d.Writer, "  Total Data Received:\t%d bytes.\n", d.r.SizeTotal)
		fmt.Fprintf(d.Writer, "  Response Size per Request:\t%d bytes.\n", d.r.SizeTotal/int64(len(d.r.Lats)))
	}
}

func (d DetailedPrinter) printStatusCodes() {
	fmt.Fprintf(d.Writer, "\nStatus code distribution:\n")
	for code, num := range d.r.StatusCodeDist {
		fmt.Fprintf(d.Writer, "  [%d]\t%d responses\n", code, num)
	}
}

func (d DetailedPrinter) printHistogram() {
	bc := 10
	buckets := make([]float64, bc+1)
	counts := make([]int, bc+1)
	bs := (d.r.Slowest - d.r.Fastest) / float64(bc)
	for i := 0; i < bc; i++ {
		buckets[i] = d.r.Fastest + bs*float64(i)
	}
	buckets[bc] = d.r.Slowest
	var bi int
	var max int
	for i := 0; i < len(d.r.Lats); {
		if d.r.Lats[i] <= buckets[bi] {
			i++
			counts[bi]++
			if max < counts[bi] {
				max = counts[bi]
			}
		} else if bi < len(buckets)-1 {
			bi++
		}
	}
	fmt.Fprintf(d.Writer, "\nResponse time histogram:\n")
	for i := 0; i < len(buckets); i++ {
		// Normalize bar lengths.
		var barLen int
		if max > 0 {
			barLen = counts[i] * 40 / max
		}
		fmt.Fprintf(d.Writer, "  %4.3f [%v]\t|%v\n", buckets[i], counts[i], strings.Repeat(barChar, barLen))
	}
}

func (d DetailedPrinter) printErrors() {
	fmt.Fprintf(d.Writer, "\nError distribution:\n")
	for err, num := range d.r.ErrorDist {
		fmt.Fprintf(d.Writer, "  [%d]\t%s\n", num, err)
	}
}

// Prints percentile latencies.
func (d DetailedPrinter) printLatencies() {
	pctls := []int{10, 25, 50, 75, 90, 95, 99}
	data := make([]float64, len(pctls))
	j := 0
	for i := 0; i < len(d.r.Lats) && j < len(pctls); i++ {
		current := i * 100 / len(d.r.Lats)
		if current >= pctls[j] {
			data[j] = d.r.Lats[i]
			j++
		}
	}
	fmt.Fprintf(d.Writer, "\nLatency distribution:\n")
	for i := 0; i < len(pctls); i++ {
		if data[i] > 0 {
			fmt.Fprintf(d.Writer, "  %v%% in %4.4f secs.\n", pctls[i], data[i])
		}
	}
}

// ExportPrinter is an implementation of Printer to get Report information after
// the benchmark is completed. 
type ExportPrinter struct {
	Report Report
}

// Print receives a report and assigns it to exported variable
func (c *ExportPrinter) Print(r Report) error {
	c.Report = r
	return nil
}

// Returns if it displays the progress bar or not when running the benchmark
func (c *ExportPrinter) ShouldDisplayProgressBar() bool{
	return false
}
