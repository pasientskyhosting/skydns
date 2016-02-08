// Copyright (c) 2014 The SkyDNS Authors. All rights reserved.
// Use of this source code is governed by The MIT License (MIT) that can be
// found in the LICENSE file.

package metrics

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/miekg/dns"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	Port      = os.Getenv("PROMETHEUS_PORT")
	Path      = envOrDefault("PROMETHEUS_PATH", "/metrics")
	Namespace = os.Getenv("PROMETHEUS_NAMESPACE")
	Subsystem = envOrDefault("PROMETHEUS_SUBSYSTEM", "skydns")

	requestCount    *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	responseSize    *prometheus.HistogramVec
	errorCount      *prometheus.CounterVec
	cacheMiss       *prometheus.CounterVec
)

type (
	System    string
	Cause     string
	CacheType string
)

var (
	Auth    System = "auth"
	Cache   System = "cache"
	Rec     System = "recursive"
	Reverse System = "reverse"
	Stub    System = "stub"

	Nxdomain  Cause = "nxdomain"
	Nodata    Cause = "nodata"
	Truncated Cause = "truncated"
	Refused   Cause = "refused"
	Overflow  Cause = "overflow"
	Loop      Cause = "loop"
	Fail      Cause = "servfail"

	Response  CacheType = "response"
	Signature CacheType = "signature"
)

func define() {
	requestCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "dns_request_count",
		Help:      "Counter of DNS requests made.",
	}, []string{"system"})

	requestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "dns_request_duration",
		Help:      "Histogram of the time (in seconds) each request took to resolve.",
		Buckets:   append([]float64{0.001, 0.003}, prometheus.DefBuckets...),
	}, []string{"system"})

	responseSize = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "dns_response_size",
		Help:      "Size of the returns response in bytes.",
		// 4k increments after 4096
		Buckets: []float64{0, 512, 1024, 1500, 2048, 4096,
			8192, 12288, 16384, 20480, 24576, 28672, 32768, 36864,
			40960, 45056, 49152, 53248, 57344, 61440, 65536,
		},
	}, []string{"system"})

	errorCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "dns_error_count",
		Help:      "Counter of DNS requests resulting in an error.",
	}, []string{"system", "cause"})

	cacheMiss = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "dns_cache_miss_count",
		Help:      "Counter of DNS requests that result in a cache miss.",
	}, []string{"cache"})

	println("define")
	println(requestDuration)
	println(responseSize)
	println("define, done")
}

// Metrics registers the DNS metrics to Prometheus, and starts the internal metrics
// server if the environment variable PROMETHEUS_PORT is set.
func Metrics() error {
	// We do this in a function instead of using var + init(), because we want to
	// able to set Namespace and/or Subsystem.
	println(requestDuration)
	println(responseSize)
	println(requestCount)
	define()

	prometheus.MustRegister(requestCount)
	prometheus.MustRegister(requestDuration)
	prometheus.MustRegister(responseSize)
	prometheus.MustRegister(errorCount)
	prometheus.MustRegister(cacheMiss)
	println(requestDuration)
	println(responseSize)
	println(requestCount)
	println("after registger")

	if Port == "" {
		println("NO PORT")
		return nil
	}

	_, err := strconv.Atoi(Port)
	if err != nil {
		fmt.Errorf("bad port for prometheus: %s", Port)
	}

	http.Handle(Path, prometheus.Handler())
	go func() {
		fmt.Errorf("%s", http.ListenAndServe(":"+Port, nil))
	}()
	return nil
}

func Duration(resp *dns.Msg, start time.Time, sys System) {
	rlen := float64(0)
	if resp != nil {
		rlen = float64(resp.Len())
	}
	println(string(sys), start.String())
	println(requestCount)
	println(requestDuration)
	println(responseSize)
	requestDuration.WithLabelValues(string(sys)).Observe(float64(time.Since(start)) / float64(time.Second))
	responseSize.WithLabelValues(string(sys)).Observe(rlen)
}

func RequestCount(req *dns.Msg, sys System) {
	requestCount.WithLabelValues(string(sys)).Inc()
	println(requestCount)
}

func ErrorCount(resp *dns.Msg, sys System) {
	if resp == nil {
		return
	}

	switch resp.Rcode {
	case dns.RcodeServerFailure:
		errorCount.WithLabelValues(string(sys), string(Fail)).Inc()
	case dns.RcodeRefused:
		errorCount.WithLabelValues(string(sys), string(Refused)).Inc()
	case dns.RcodeNameError:
		errorCount.WithLabelValues(string(sys), string(Nxdomain)).Inc()
	// nodata ??
	}

}

func CacheMiss(ca CacheType) { cacheMiss.WithLabelValues(string(ca)).Inc() }

func envOrDefault(env, def string) string {
	e := os.Getenv(env)
	if e != "" {
		return e
	}
	return def
}