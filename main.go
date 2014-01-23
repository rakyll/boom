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
	"flag"
	"fmt"
	"net"
	"net/http"
	gourl "net/url"
	"os"
	"regexp"
	"strings"

	"github.com/rakyll/boom/commands"
)

var (
	flagMethod   = flag.String("m", "GET", "")
	flagHeaders  = flag.String("h", "", "")
	flagD        = flag.String("d", "", "")
	flagType     = flag.String("T", "text/html", "")
	flagAuth     = flag.String("a", "", "")
	flagInsecure = flag.Bool("allow-insecure", false, "")
	flagOutput   = flag.String("o", "", "")

	flagC = flag.Int("c", 50, "")
	flagN = flag.Int("n", 200, "")
	flagQ = flag.Int("q", 0, "")
	flagT = flag.Int("t", 0, "")
)

var usage = `Usage: boom [options...] <url>

Options:
  -n  Number of requests to run.
  -c  Number of requests to run concurrently. Total number of requests cannot
      be smaller than the concurency level.
  -q  Rate limit, in seconds (QPS).
  -o  Output type. If none provided, a summary is printed.
      "csv" is the only supported alternative. Dumps the response
      metrics in comma-seperated values format.

  -m  HTTP method, one of GET, POST, PUT, DELETE, HEAD, OPTIONS.
  -h  Custom HTTP headers, name1:value1;name2:value2.
  -d  HTTP request body.
  -T  Content-type, defaults to "text/html".
  -a  Basic authentication, username:password.

  -allow-insecure Allow bad/expired TLS/SSL certificates.
`

func main() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
	}

	flag.Parse()
	if flag.NArg() < 1 {
		usageAndExit("")
	}

	n := *flagN
	c := *flagC
	q := *flagQ
	t := *flagT

	if n <= 0 || c <= 0 {
		usageAndExit("n and c cannot be smaller than 1.")
	}

	// If total number is smaller than concurrency level,
	// make the total number c.
	if c > n {
		n = c
	}

	url := flag.Args()[0]
	method := strings.ToUpper(*flagMethod)

	ipResolvedUrl, serverName := newURL(url)
	req, _ := http.NewRequest(
		method, ipResolvedUrl.String(), strings.NewReader(*flagD))
	// set content-type
	req.Header.Set("Content-Type", *flagType)

	// set any other additional headers
	if *flagHeaders != "" {
		headers := strings.Split(*flagHeaders, ";")
		for _, h := range headers {
			re := regexp.MustCompile("([\\w|-]+):(.+)")
			matches := re.FindAllStringSubmatch(h, -1)
			if len(matches) < 1 {
				usageAndExit("")
			}
			req.Header.Set(matches[0][1], matches[0][2])
		}
	}

	// set basic auth if set
	if *flagAuth != "" {
		re := regexp.MustCompile("(\\w+):(\\w+)")
		matches := re.FindAllStringSubmatch(*flagAuth, -1)
		if len(matches) < 1 {
			usageAndExit("")
		}
		req.SetBasicAuth(matches[0][1], matches[0][2])
	}

	if *flagOutput != "csv" && *flagOutput != "" {
		usageAndExit("Invalid output type.")
	}

	(&commands.Boom{
		N:              n,
		C:              c,
		Qps:            q,
		Timeout:        t,
		Req:            req,
		OrigServerName: serverName,
		AllowInsecure:  *flagInsecure,
		Output:         *flagOutput}).Run()
}

// Replaces host with an IP and returns the provided
// string URL as a *url.URL.
//
// DNS lookups are not cached in the package level in Go,
// and it's a huge overhead to resolve a host
// before each request in our case. Instead we resolve
// the domain and replace it with the resolved IP to avoid
// lookups during request time. Supported url strings:
//
// <schema>://google.com[:port]
// <schema>://173.194.116.73[:port]
// <schema>://\[2a00:1450:400a:806::1007\][:port]
func newURL(url string) (*gourl.URL, string) {
	uri, err := gourl.ParseRequestURI(url)
	if err != nil {
		usageAndExit(err.Error())
	}

	serverName, port, err := net.SplitHostPort(uri.Host)
	if err != nil {
		serverName = uri.Host
	}

	addrs, err := net.LookupHost(serverName)
	if err != nil {
		usageAndExit("Hostname " + serverName + " cannot be resolved")
	}

	if port != "" {
		// join automatically puts square brackets around the
		// ipv6 IPs.
		uri.Host = net.JoinHostPort(addrs[0], port)
	} else {
		// square brackets are required for ipv6 IPs.
		// otherwise, net.Dial fails with a parsing error.
		uri.Host = fmt.Sprintf("[%s]", addrs[0])
	}
	return uri, serverName
}

func usageAndExit(message string) {
	if message != "" {
		fmt.Fprintf(os.Stderr, message)
		fmt.Fprintf(os.Stderr, "\n\n")
	}
	flag.Usage()
	fmt.Fprintf(os.Stderr, "\n")
	os.Exit(1)
}
