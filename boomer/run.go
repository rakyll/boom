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
	"log"

	"sync"

	"net/http"
	"path/filepath"
	"time"
)

// Run makes all the requests, prints the summary. It blocks until
// all work is done.
func (b *Boomer) Run() {
	b.results = make(chan *result, b.N)
	if b.Output == "" {
		b.bar = newPb(b.N)
	}

	start := time.Now()
	b.run()
	if b.Output == "" {
		b.bar.Finish()
	}

	printReport(b.N, b.results, b.Output, time.Now().Sub(start))
	close(b.results)
}

func (b *Boomer) worker(wg *sync.WaitGroup, ch chan *http.Request) {

	tlsConfig := &tls.Config{}

	if b.Cert != "" && b.Key != "" {
		cert := filepath.Join("", b.Cert)
		key := filepath.Join("", b.Key)
		tlsCert, err := tls.LoadX509KeyPair(cert, key)

		if err != nil {
			log.Fatalf("Failed to load cert")
		}

		tlsConfig = &tls.Config{
			Certificates:       []tls.Certificate{tlsCert},
			InsecureSkipVerify: b.AllowInsecure,
		}
	}

	tr := &http.Transport{
		TLSClientConfig:    tlsConfig,
		DisableCompression: b.DisableCompression,
		DisableKeepAlives:  b.DisableKeepAlives,
		// TODO(jbd): Add dial timeout.
		TLSHandshakeTimeout: time.Duration(b.Timeout) * time.Millisecond,
		Proxy:               http.ProxyURL(b.ProxyAddr),
	}
	client := &http.Client{Transport: tr}
	for req := range ch {
		s := time.Now()
		code := 0
		size := int64(0)
		resp, err := client.Do(req)
		if err == nil {
			size = resp.ContentLength
			code = resp.StatusCode
			resp.Body.Close()
		}
		if b.bar != nil {
			b.bar.Increment()
		}
		wg.Done()

		b.results <- &result{
			statusCode:    code,
			duration:      time.Now().Sub(s),
			err:           err,
			contentLength: size,
		}
	}
}

func (b *Boomer) run() {
	var wg sync.WaitGroup
	wg.Add(b.N)

	var throttle <-chan time.Time
	if b.Qps > 0 {
		throttle = time.Tick(time.Duration(1e6/(b.Qps)) * time.Microsecond)
	}
	jobs := make(chan *http.Request, b.N)
	for i := 0; i < b.C; i++ {
		go func() {
			b.worker(&wg, jobs)
		}()
	}
	for i := 0; i < b.N; i++ {
		if b.Qps > 0 {
			<-throttle
		}
		jobs <- b.Req.Request()
	}
	close(jobs)

	wg.Wait()
}
