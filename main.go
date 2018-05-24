package main

import (
	"log"
	"net/http"
	"time"

	prometheus "github.com/prometheus/client_golang/prometheus"
	promhttp "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	url = "http://www.google.com"
)

func main() {
	//Create Client
	client := http.DefaultClient
	client.Timeout = 1 * time.Second

	//Create Summaries
	// dnsLatencyVec
	dnsLatencyVec := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "dns_duration_seconds",
			Help: "Trace dns latency summary.",
		},
		[]string{"event"},
	)
	//tcpLatencyVec
	tcpLatencyVec := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "tcp_duration_seconds",
			Help: "Trace tcp latency summary.",
		},
		[]string{"event"},
	)
	// requestTotalDurationVec
	requestTotalDurationVec := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "request_duration_seconds",
			Help: "Request duration in seconds.",
		},
		[]string{"event"},
	)

	//Register all the metrics with Prometheus
	prometheus.MustRegister(dnsLatencyVec, tcpLatencyVec, requestTotalDurationVec)

	// Define functions for the available httptrace.ClientTrace hook
	// functions that we want to instrument.
	trace := &promhttp.InstrumentTrace{
		DNSStart: func(t float64) {
			dnsLatencyVec.WithLabelValues("dns_start")
		},
		DNSDone: func(t float64) {
			dnsLatencyVec.WithLabelValues("dns_done")
		},
		GotConn: func(t float64) {
			tcpLatencyVec.WithLabelValues("TCP_start")
		},
		GotFirstResponseByte: func(t float64) {
			tcpLatencyVec.WithLabelValues("TCP_done")
		},
	}
	// Wrap the default RoundTripper with middleware.
	roundTripper := promhttp.InstrumentRoundTripperTrace(trace, promhttp.InstrumentRoundTripperDuration(requestTotalDurationVec, http.DefaultTransport))

	// Set the RoundTripper on the client.
	client.Transport = roundTripper

	//Read and close body
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("error: %v", err)
	}
	defer resp.Body.Close()

	// The Handler function provides a default handler to expose metrics
	// via an HTTP server. "/metrics" is the usual endpoint for that.
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":7777", nil))
}
