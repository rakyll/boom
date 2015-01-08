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

import "testing"

func TestCacheResolution(t *testing.T) {
	c := newCacheDialer()
	tests := []struct {
		net, addr string
		port      int
	}{
		{"tcp", "127.0.0.1:80", 80},
		{"tcp", "[0:0:0:0:0:0:0:1]:80", 80},
		{"tcp", "[::1]:80", 80},
		{"tcp", "localhost:http", 80},
		{"tcp", "localhost:80", 80},
		{"tcp", "localhost:https", 443},
		{"tcp", "localhost:443", 443},
		{"tcp", "localhost:8080", 8080},
		{"tcp4", "localhost:80", 80},
		{"tcp6", "localhost:80", 80},
	}
	for _, test := range tests {
		addr, err := c.ResolveTCPAddr(test.net, test.addr)
		if err != nil {
			t.Errorf("Unexpected error: %v\n", err)
		} else if !addr.IP.IsLoopback() {
			t.Errorf("Expected loopback IP, got: %#v\n", addr)
		} else if addr.Port != test.port {
			t.Errorf("Expected port %d, got: %#d\n", test.port, addr.Port)
		}
	}
}
