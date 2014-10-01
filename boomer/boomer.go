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
	"github.com/rakyll/pb"
	"net/http"
	"strings"
	"time"
)

type result struct {
	err           error
	statusCode    int
	duration      time.Duration
	contentLength int64
}

type ReqOpts struct {
	Method   string
	Url      string
	Header   http.Header
	Body     string
	Username string
	Password string
	// Request host is an resolved IP. TLS/SSL handshakes may require
	// the original server name, keep it to initate the TLS client.
	OriginalHost string
}

// Creates a req object from req options
func (r *ReqOpts) Request() *http.Request {
	req, _ := http.NewRequest(r.Method, r.Url, strings.NewReader(r.Body))
	req.Header = r.Header

	// update the Host value in the Request - this is used as the host header in any subsequent request
	req.Host = r.OriginalHost

	if r.Username != "" && r.Password != "" {
		req.SetBasicAuth(r.Username, r.Password)
	}
	return req
}

type Boomer struct {
	// Request to make.
	Req *ReqOpts
	// Total number of requests to make.
	N int
	// Concurrency level, the number of concurrent workers to run.
	C int
	// Timeout.
	Timeout time.Duration
	// Rate limit.
	Qps int
	// Option to allow insecure TLS/SSL certificates.
	AllowInsecure bool

	// Output type
	Output string

	// Optional address of HTTP proxy server as host:port
	ProxyAddr string

	bar     *pb.ProgressBar
	rpt     *report
	results chan *result
}

func newPb(size int) (bar *pb.ProgressBar) {
	bar = pb.New(size)
	bar.Format("Bom !")
	bar.Start()
	return
}
