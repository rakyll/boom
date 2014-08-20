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
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestN(t *testing.T) {
	var count int64
	handler := func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&count, int64(1))
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	boomer := &Boomer{
		Req: &ReqOpts{
			Method: "GET",
			Url:    server.URL,
		},
		N: 20,
		C: 2,
	}
	boomer.Run()
	if count != 20 {
		t.Errorf("Expected to boom 20 times, found %v", count)
	}
}

func TestQps(t *testing.T) {
	var wg sync.WaitGroup
	var count int64
	handler := func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&count, int64(1))
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	boomer := &Boomer{
		Req: &ReqOpts{
			Method: "GET",
			Url:    server.URL,
		},
		N:   20,
		C:   2,
		Qps: 1,
	}
	wg.Add(1)
	time.AfterFunc(time.Second, func() {
		if count > 1 {
			t.Errorf("Expected to boom 1 times, found %v", count)
		}
		wg.Done()
	})
	go boomer.Run()
	wg.Wait()
}

func TestRequest(t *testing.T) {
	var uri, contentType, some, method, auth string
	handler := func(w http.ResponseWriter, r *http.Request) {
		uri = r.RequestURI
		method = r.Method
		contentType = r.Header.Get("Content-type")
		some = r.Header.Get("X-some")
		auth = r.Header.Get("Authorization")
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	header := make(http.Header)
	header.Add("Content-type", "text/html")
	header.Add("X-some", "value")
	boomer := &Boomer{
		Req: &ReqOpts{
			Method:   "PUT",
			Url:      server.URL,
			Header:   header,
			Username: "username",
			Password: "password",
		},
		N: 1,
		C: 1,
	}
	boomer.Run()
	if uri != "/" {
		t.Errorf("Uri is expected to be /, %v is found", uri)
	}
	if contentType != "text/html" {
		t.Errorf("Content type is expected to be text/html, %v is found", contentType)
	}
	if some != "value" {
		t.Errorf("X-some header is expected to be value, %v is found", some)
	}
	if auth != "Basic dXNlcm5hbWU6cGFzc3dvcmQ=" {
		t.Errorf("Basic authorization is not properly set")
	}
}

func TestBody(t *testing.T) {
	var count int64
	handler := func(w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)
		if string(body) == "Body" {
			atomic.AddInt64(&count, int64(1))
		}
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	boomer := &Boomer{
		Req: &ReqOpts{
			Method: "POST",
			Url:    server.URL,
			Body:   "Body",
		},
		N: 10,
		C: 1,
	}
	boomer.Run()
	if count != 10 {
		t.Errorf("Expected to boom 10 times, found %v", count)
	}
}

func TestContentLengthIfExists(t *testing.T) {
	buf := make([]byte, 20)
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Write(buf)
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	boomer := &Boomer{
		Req: &ReqOpts{
			Method: "GET",
			Url:    server.URL,
		},
		N: 10,
		C: 1,
	}
	boomer.Run()

	if exp := int64(len(buf) * boomer.N); boomer.rpt.sizeTotal != exp {
		t.Errorf("Expected Total Data Received %d bytes, found %d", exp, boomer.rpt.sizeTotal)
	}
}

func TestContentLengthIfDontExists(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	boomer := &Boomer{
		Req: &ReqOpts{
			Method: "GET",
			Url:    server.URL,
		},
		N: 10,
		C: 1,
	}
	boomer.Run()

	if boomer.rpt.sizeTotal != 0 {
		t.Errorf("Expected Total Data Received 0 bytes, found %v", boomer.rpt.sizeTotal)
	}
}

func testTimeout(t *testing.T, handler func(delay time.Duration, w http.ResponseWriter)) {
	var count int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var delay time.Duration
		if atomic.AddInt64(&count, 1)%2 == 0 {
			delay = 200 * time.Millisecond
		}
		handler(delay, w)
	}))
	defer server.Close()

	boomer := &Boomer{
		Req: &ReqOpts{
			Method: "GET",
			Url:    server.URL,
		},
		Timeout: 100 * time.Millisecond,
		N:       20,
		C:       5,
	}
	boomer.Run()

	errs := 0
	expErrStr := errTimeout.Error()
	for errStr, i := range boomer.rpt.errors {
		if errStr != expErrStr {
			t.Errorf("Expected %q error, got %q", expErrStr, errStr)
		}
		errs += i
	}
	if exp := boomer.N / 2; errs != exp {
		t.Errorf("Expected %d errors, found %d", exp, errs)
	}
}

func TestTimeoutHeaders(t *testing.T) {
	t.Parallel()
	testTimeout(t, func(delay time.Duration, w http.ResponseWriter) {
		time.Sleep(delay)
	})
}

func TestTimeoutBody(t *testing.T) {
	t.Parallel()
	testTimeout(t, func(delay time.Duration, w http.ResponseWriter) {
		// Write enough data to make the buffers flush at least once. 4k is currently enough
		// but this behavior is hidden in the http package, so let's throw some extra at it
		// in the hope that the test stays somewhat resilient to future underlying changes.
		buf := make([]byte, 8<<10)
		w.Write(buf)
		time.Sleep(delay)
	})
}
