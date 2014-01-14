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
	"crypto/tls"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

func (b *Boom) Run() {
	b.Req.Header.Add("cache-control", "no-cache")
	b.init()
	b.run()
	b.rpt.finalize()
}

func (b *Boom) init() {
	if b.Client == nil {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: b.AllowInsecure},
		}
		b.Client = &http.Client{Transport: tr}
	}
	b.results = make(chan *result, b.C)
	b.bar = newPb(b.N)
	b.rpt = newReport(b.N)
}

func (b *Boom) worker(wg *sync.WaitGroup) {
	defer wg.Done()
	for _ = range b.jobs {
		s := time.Now()
		resp, err := b.Client.Do(b.Req)
		code := 0
		if resp != nil {
			code = resp.StatusCode
		}
		b.results <- &result{
			statusCode: code,
			duration:   time.Now().Sub(s),
			err:        err,
		}
		if resp != nil {
			io.Copy(ioutil.Discard, resp.Body)
			resp.Body.Close()
		}
		b.bar.Increment()
	}
}

func (b *Boom) run() {
	var wg sync.WaitGroup
	// Start throttler if rate limit is specified.
	var throttle <-chan time.Time
	if b.Qps > 0 {
		throttle = time.Tick(time.Duration(1e6/b.Qps) * time.Microsecond)
	}
	var timeout <-chan time.Time
	if b.Timeout > 0 {
		timeout = time.After(time.Duration(b.Timeout) * time.Second)
	}
	finished := make(chan bool)
	// Start collecting results.
	go func() {
		for res := range b.results {
			b.rpt.update(res)
		}
		finished <- true
	}()
	// Send N requests.
	b.jobs = make(chan bool, b.C)
	// Start C workers to consume.
	for i := 0; i < b.C; i++ {
		wg.Add(1)
		go b.worker(&wg)
	}
requestLoop:
	for i := 0; i < b.N; i++ {
		select {
		case <-timeout:
			break requestLoop

		default:
			if b.Qps > 0 {
				<-throttle
			}
			b.jobs <- true
		}
	}
	close(b.jobs)
	// Wait until all workers are done.
	wg.Wait()
	b.bar.Finish()
	close(b.results)
	// Wait until all results are processed.
	<-finished
}
