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
			URL:    server.URL,
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
			URL:    server.URL,
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
	var uri, contentType, some, method, auth, host string
	handler := func(w http.ResponseWriter, r *http.Request) {
		uri = r.RequestURI
		method = r.Method
		contentType = r.Header.Get("Content-type")
		host = r.Host
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
			URL:      server.URL,
			Header:   header,
			Host:     "example.com",
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
	if host != "example.com" {
		t.Errorf("Host header is not properly set")
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
			URL:    server.URL,
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
