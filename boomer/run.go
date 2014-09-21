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
	"crypto/tls"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"time"
)

var errTimeout = errors.New("timeout")

func (b *Boomer) Run() {
	b.results = make(chan *result, b.N)
	if b.Output == "" {
		b.bar = newPb(b.N)
	}
	b.rpt = newReport(b.N, b.results, b.Output)
	b.run()
}

func (b *Boomer) worker(ch chan *http.Request) {
	host, _, _ := net.SplitHostPort(b.Req.OriginalHost)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: b.AllowInsecure, ServerName: host},
	}
	if b.ProxyAddr != "" {
		tr.Dial = func(network string, addr string) (conn net.Conn, err error) {
			return net.Dial(network, b.ProxyAddr)
		}
	}
	client := &http.Client{Transport: tr}
	for req := range ch {
		b.results <- doTimeout(client, req, tr, b.Timeout)
		if b.bar != nil {
			b.bar.Increment()
		}
	}
}

func doTimeout(client *http.Client, req *http.Request, tr *http.Transport, timeout time.Duration) *result {
	if timeout == 0 {
		return do(client, req)
	}
	if timeout < 0 {
		return &result{err: errTimeout}
	}

	t := time.NewTicker(timeout)
	defer t.Stop()
	resc := make(chan *result, 1)
	go func() {
		resc <- do(client, req)
	}()
	select {
	case <-t.C:
		tr.CancelRequest(req)
		return &result{err: errTimeout}
	case r := <-resc:
		return r
	}
}

func do(client *http.Client, req *http.Request) *result {
	s := time.Now()
	resp, err := client.Do(req)
	r := &result{err: err}
	if resp != nil {
		r.statusCode = resp.StatusCode
		// consume the whole body
		r.contentLength, r.err = io.Copy(ioutil.Discard, resp.Body)
		// cleanup body, so the socket can be reusable
		resp.Body.Close()
	}
	r.duration = time.Since(s)
	return r
}

func (b *Boomer) run() {
	var wg sync.WaitGroup
	wg.Add(b.C)

	var throttle <-chan time.Time
	if b.Qps > 0 {
		throttle = time.Tick(time.Duration(1e6/(b.Qps)) * time.Microsecond)
	}

	start := time.Now()
	jobs := make(chan *http.Request, b.N)
	// Start workers.
	for i := 0; i < b.C; i++ {
		go func() {
			b.worker(jobs)
			wg.Done()
		}()
	}

	// Start sending jobs to the workers.
	for i := 0; i < b.N; i++ {
		if b.Qps > 0 {
			<-throttle
		}
		jobs <- b.Req.Request()
	}
	close(jobs)

	wg.Wait()
	if b.bar != nil {
		b.bar.Finish()
	}
	b.rpt.finalize(time.Now().Sub(start))
}
