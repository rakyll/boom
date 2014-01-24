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

package main

import (
	"testing"
)

type mockDnsResolver struct {
	Addr string
}

func (r *mockDnsResolver) Lookup(host string) ([]string, error) {
	return []string{r.Addr}, nil
}

func TestParseUrl_IPv4(t *testing.T) {
	defaultDnsResolver = &mockDnsResolver{Addr: "127.0.0.1"}
	u, s := newURL("http://google.com")
	t.Log(u.String())
	if s != "google.com" {
		t.Errorf("Original server name doesn't match with google.com, %v is found.", s)
	}
	if u.String() != "http://[127.0.0.1]" {
		t.Errorf("URL is expected to be http://127.0.0.1, %v is found.", u)
	}
}

func TestParseUrl_IPv4AndPort(t *testing.T) {
	defaultDnsResolver = &mockDnsResolver{Addr: "127.0.0.1"}
	u, s := newURL("http://google.com:80")
	if s != "google.com" {
		t.Errorf("Original server name doesn't match with google.com, %v is found.", s)
	}
	if u.String() != "http://127.0.0.1:80" {
		t.Errorf("URL is expected to be http://127.0.0.1, %v is found.", u)
	}
}

func TestParseUrl_IPv6(t *testing.T) {
	defaultDnsResolver = &mockDnsResolver{Addr: "2a00:1450:400a:806::1007"}
	u, s := newURL("http://google.com")
	if s != "google.com" {
		t.Errorf("Original server name doesn't match with google.com, %v is found.", s)
	}
	if u.String() != "http://[2a00:1450:400a:806::1007]" {
		t.Errorf("URL is expected to be http://[2a00:1450:400a:806::1007], %v is found.", u)
	}
}

func TestParseUrl_IPv6AndPort(t *testing.T) {
	defaultDnsResolver = &mockDnsResolver{Addr: "2a00:1450:400a:806::1007"}
	u, s := newURL("http://google.com:80")
	if s != "google.com" {
		t.Errorf("Original server name doesn't match with google.com, %v is found.", s)
	}
	if u.String() != "http://[2a00:1450:400a:806::1007]:80" {
		t.Errorf("URL is expected to be http://[2a00:1450:400a:806::1007]:80, %v is found.", u)
	}
}
