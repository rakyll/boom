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

// Package boomer provides commands to run load tests and display results.
package boomer

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	pb "gopkg.in/cheggaaa/pb.v1"

	"golang.org/x/net/http2"
)

type result struct {
	err           error
	statusCode    int
	duration      time.Duration
	contentLength int64
	matchErr      error
}

// Boomer Describes a load test
type Boomer struct {
	// Request is the request to be made.
	Request *http.Request

	RequestBody string

	// N is the total number of requests to make.
	N int

	// C is the concurrency level, the number of concurrent workers to run.
	C int

	// H2 is an option to make HTTP/2 requests
	H2 bool

	// Timeout in seconds.
	Timeout int

	// QPS is the rate limit.
	QPS int

	// DisableCompression is an option to disable compression in response
	DisableCompression bool

	// DisableKeepAlives is an option to prevents re-use of TCP connections between different HTTP requests
	DisableKeepAlives bool

	// Output represents the output type. If "csv" is provided, the
	// output will be dumped as a csv stream.
	Output string

	// The expected content of the response body.
	ResponseBody string

	// Whether to test the content of the response body.
	TestResponse bool

	// ProxyAddr is the address of HTTP proxy server in the format on "host:port".
	// Optional.
	ProxyAddr *url.URL

	results chan *result

	responseEvent chan bool
}

// Run makes all the requests, prints the summary. It blocks until
// all work is done.
func (b *Boomer) Run() {
	b.results = make(chan *result, b.N)
	b.responseEvent = make(chan bool, b.N)

	start := time.Now()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		<-c
		// TODO(jbd): Progress bar should not be finalized.
		newReport(b.N, b.results, b.Output, time.Now().Sub(start)).finalize()
		os.Exit(1)
	}()

	barDone := make(chan bool, 1)
	go func() {
		bar := pb.StartNew(b.N)
		defer func() {
			bar.Finish()
			barDone <- true
		}()
		for done := range b.responseEvent {
			if done {
				break
			} else {
				bar.Increment()
			}
		}
	}()
	b.runWorkers()
	nr := newReport(b.N, b.results, b.Output, time.Now().Sub(start))
	<-barDone
	nr.finalize()
	close(b.results)
}
func strMatch(b1, b2 string) (bool, error) {
	isMatch := b1 == b2
	var err error
	if !isMatch {
		for i := range b1 {
			if i >= len(b2) || b1[i] != b2[i] {
				msg := fmt.Sprintf("First non matching char at index: %v", i)
				err = errors.New(msg)
				break
			}
		}

	}
	return isMatch, err
}

func (b *Boomer) makeRequest(c *http.Client) {
	var (
		s        = time.Now()
		size     int64
		code     int
		matchErr error
	)

	resp, err := c.Do(cloneRequest(b.Request, b.RequestBody))
	if err == nil {
		size = resp.ContentLength
		code = resp.StatusCode
		bodyBuf, _ := ioutil.ReadAll(resp.Body) // overwrite error
		bodyStr := string(bodyBuf)

		if b.TestResponse {
			_, matchErr = strMatch(bodyStr, b.ResponseBody)
		}

		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}
	b.results <- &result{
		statusCode:    code,
		duration:      time.Now().Sub(s),
		err:           err,
		contentLength: size,
		matchErr:      matchErr,
	}
	b.responseEvent <- false
}

func (b *Boomer) runWorker(n int) {
	var throttle <-chan time.Time
	if b.QPS > 0 {
		throttle = time.Tick(time.Duration(1e6/(b.QPS)) * time.Microsecond)
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		DisableCompression: b.DisableCompression,
		DisableKeepAlives:  b.DisableKeepAlives,
		// TODO(jbd): Add dial timeout.
		TLSHandshakeTimeout: time.Duration(b.Timeout) * time.Millisecond,
		Proxy:               http.ProxyURL(b.ProxyAddr),
	}
	if b.H2 {
		http2.ConfigureTransport(tr)
	} else {
		tr.TLSNextProto = make(map[string]func(string, *tls.Conn) http.RoundTripper)
	}
	client := &http.Client{Transport: tr}
	for i := 0; i < n; i++ {
		if b.QPS > 0 {
			<-throttle
		}
		b.makeRequest(client)
	}
}

func (b *Boomer) runWorkers() {
	var wg sync.WaitGroup
	wg.Add(b.C)

	// Ignore the case where b.N % b.C != 0.
	for i := 0; i < b.C; i++ {
		go func() {
			b.runWorker(b.N / b.C)
			wg.Done()
		}()
	}
	wg.Wait()
	b.responseEvent <- true
}

// cloneRequest returns a clone of the provided *http.Request.
// The clone is a shallow copy of the struct and its Header map.
func cloneRequest(r *http.Request, body string) *http.Request {
	// shallow copy of the struct
	r2 := new(http.Request)
	*r2 = *r
	// deep copy of the Header
	r2.Header = make(http.Header, len(r.Header))
	for k, s := range r.Header {
		r2.Header[k] = append([]string(nil), s...)
	}
	r2.Body = ioutil.NopCloser(strings.NewReader(body))
	return r2
}
