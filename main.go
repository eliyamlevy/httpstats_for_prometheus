package main

import (
	"log"
	"net/http"
	"time"

	prometheus "github.com/prometheus/client_golang/prometheus"
	promhttp "github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tkanos/gonfig"
)

func main() {
	//Create config struct
	type Configuration struct {
		Port   string
		URLmap []string
	}
	configuration := Configuration{}

	//Grab Details
	err := gonfig.GetConf("config.json", &configuration)
	if err != nil {
		log.Printf("error: %v", err)
	}

	//Create Client
	client := http.DefaultClient
	client.Timeout = 1 * time.Second

	//Create Summaries
	// dnsLatencyVec
	dnsLatencyVec := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "dns_duration_seconds",
			Help: "Trace dns latency Gauge.",
		},
		[]string{"method", "url"},
	)
	//tcpLatencyVec
	tcpLatencyVec := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tcp_duration_seconds",
			Help: "Trace tcp latency Gauge.",
		}, []string{"method", "url"},
	)
	//requestTimer
	requestTimer := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "request_timer",
			Help: "Request Timer.",
		}, []string{"method", "url"},
	)
	// requestTotalDurationVec
	requestAvgDurationVec := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "request_duration_seconds",
			Help: "Request duration in seconds.",
		},
		[]string{"method"},
	)

	//Register all the metrics with Prometheus
	prometheus.MustRegister(dnsLatencyVec, tcpLatencyVec, requestTimer, requestAvgDurationVec)
	//
	numURLs := len(configuration.URLmap)
	for i := 0; i < numURLs; i++ {
		// Define functions for the available httptrace.ClientTrace hook
		// functions that we want to instrument.
		trace := &promhttp.InstrumentTrace{
			DNSStart: func(t float64) {
				dnsLatencyVec.WithLabelValues("dns_start", configuration.URLmap[i])
			},
			DNSDone: func(t float64) {
				dnsLatencyVec.WithLabelValues("dns_done", configuration.URLmap[i])
			},
			GotConn: func(t float64) {
				tcpLatencyVec.WithLabelValues("TCP_start", configuration.URLmap[i])
			},
			GotFirstResponseByte: func(t float64) {
				tcpLatencyVec.WithLabelValues("TCP_done", configuration.URLmap[i])
			},
		}

		// Wrap the default RoundTripper with middleware.
		roundTripper := promhttp.InstrumentRoundTripperTrace(trace,
			promhttp.InstrumentRoundTripperDuration(requestAvgDurationVec, http.DefaultTransport))

		timer := prometheus.NewTimer(prometheus.ObserverFunc(requestTimer.WithLabelValues("GET", configuration.URLmap[i]).Set))
		// Set the RoundTripper on the client.
		client.Transport = roundTripper

		//Read and close body for each url
		resp, err := client.Get(configuration.URLmap[i])
		if err != nil {
			log.Printf("error: %v", err)
		}
		resp.Body.Close()
		timer.ObserveDuration()
	}
	log.Printf("%+v", configuration)
	// The Handler function provides a default handler to expose metrics
	// via an HTTP server. "/metrics" is the usual endpoint for that.
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(configuration.Port, nil))
}
