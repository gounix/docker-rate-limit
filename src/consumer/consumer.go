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

package consumer

import (
	"fmt"
	"net/http"
	"docker-rate-limit/data"
	"docker-rate-limit/logger"
)

const promHeader = `# HELP ratelimit_limit docker hub ratelimit
# TYPE ratelimit_limit gauge
# HELP ratelimit_remaining docker hub ratelimit remaining
# TYPE ratelimit_remaining gauge`

var interval int64

func logRequest(r *http.Request) {
	fmt.Printf("%s: %s %s %s\n", r.Host, r.Method, r.URL.Path, r.UserAgent())
	logger.Info("consumer.logRequest", "host", r.Host, "method", r.Method, "url", r.URL.Path, "user agent", r.UserAgent())
}

func sendPromLines(w http.ResponseWriter, limit string, remaining string, source string) {

	str := fmt.Sprintf("ratelimit_limit{account=\"%s\"} %s\n", source, limit)
	fmt.Fprintf(w, str)
	logger.Info("consumer.sendPromLines", "reply", str)

	str = fmt.Sprintf("ratelimit_remaining{account=\"%s\"} %s\n", source, remaining)
	fmt.Fprintf(w, str)
	logger.Info("consumer.sendPromLines", "reply", str)
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {

	logRequest(r)
	limit, remaining, source, err := data.Get()
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
	} else {
		fmt.Fprintln(w, promHeader)
		sendPromLines(w, limit, remaining, source)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	logRequest(r)
	if data.Alive(interval) {
		fmt.Fprintf(w, "OK")
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func Get(port int64, intrvl int64) {
	interval = intrvl
	http.HandleFunc("/metrics", metricsHandler)
	http.HandleFunc("/health", healthHandler)
	addr := fmt.Sprintf(":%d", port)
	http.ListenAndServe(addr, nil)
}

