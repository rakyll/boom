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
	"net"
	"sync"
)

type cacheDialer struct {
	mu    sync.RWMutex
	cache map[string]*net.TCPAddr
}

func newCacheDialer() *cacheDialer {
	return &cacheDialer{
		cache: make(map[string]*net.TCPAddr),
	}
}

func (c *cacheDialer) Dial(network, address string) (net.Conn, error) {
	switch network {
	case "tcp", "tcp4", "tcp6":
		addr, err := c.ResolveTCPAddr(network, address)
		if err != nil {
			return nil, err
		}
		return net.DialTCP(network, nil, addr)
	default:
		// There's no need to bother with other networks for now.
		return nil, net.UnknownNetworkError(network)
	}
}

func (c *cacheDialer) ResolveTCPAddr(network, address string) (*net.TCPAddr, error) {
	key := network + ":" + address
	c.mu.RLock()
	addr := c.cache[key]
	c.mu.RUnlock()
	if addr != nil {
		return addr, nil
	}
	// net package coalesces concurrent dns lookups into one
	addr, err := net.ResolveTCPAddr(network, address)
	if err == nil {
		c.mu.Lock()
		c.cache[key] = addr
		c.mu.Unlock()
	}
	return addr, err
}
