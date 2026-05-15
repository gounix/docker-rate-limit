/*
MIT License

Copyright (c) 2026 gounix

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package data

import (
	"sync"
	"time"
	"errors"
	"docker-rate-limit/logger"
)

type dataT struct {
	mu sync.Mutex
	limit string
	remaining string
	source string
	timestamp time.Time
	initialized bool
}

var data = dataT{ initialized: false }

func Put(limit string, remaining string, source string) {
    data.mu.Lock()
    defer data.mu.Unlock()

    data.initialized = true
    data.limit = limit
    data.remaining = remaining
    data.source = source
    data.timestamp = time.Now()
    logger.Info("data.Put", "limit", data.limit, "remaining", data.remaining, "source", data.source)
}

func Get() (string, string, string, error) {
    data.mu.Lock()
    defer data.mu.Unlock()

    if data.initialized {
	    logger.Info("data.Get", "limit", data.limit, "remaining", data.remaining, "source", data.source)
	    return data.limit, data.remaining, data.source, nil
    } else {
	    logger.Info("data.Get not initialized")
	    return data.limit, data.remaining, data.source, errors.New("not initialized")
    }
}

func Alive(interval int64) bool {
	now := time.Now()
	diff := now.Sub(data.timestamp)

	// consider the producer dead after it missed 2 intervals
	isOK := diff.Seconds() < float64(2 * interval)
	logger.Info("data.Alive", "age", diff.Seconds(), "OK", isOK)

	return isOK
}

