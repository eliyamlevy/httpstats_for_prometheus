package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	prometheus "github.com/prometheus/client_golang/prometheus"
	promhttp "github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tkanos/gonfig"
)

var confPaths = []string{
	"config.json",                    //test run
	"/etc/goscraper/app/config.json", //inside a dockprom setup
	"/app/config.json",               //inside a docker container
}

func main() {
	//Capture if the program is exited and logs exit signal
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	go func() {
		select {
		case sig := <-c:
			log.Printf("Got %s signal. Aborting...\n", sig)
			os.Exit(1)
		}
	}()

	//Create config struct
	type Configuration struct {
		Port   string
		URLmap []string
	}
	configuration := Configuration{}

	//Grab Details
	for i := 0; i < len(confPaths); i++ {
		err := gonfig.GetConf(confPaths[i], &configuration)
		if err != nil {
			log.Printf("error: %v", err)
		} else if err == nil {
			log.Printf("config file read from " + confPaths[i])
		}
	}

	//Create Client
	client := http.DefaultClient
	client.Timeout = 1 * time.Second
	log.Printf("client created")

	//Create Summaries
	// dnsLatencyVec
	dnsLatencyVec := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "dns_duration_seconds",
			Help: "Trace dns latency.",
		},
		[]string{"method", "url"},
	)
	//TLSHandshakeVec
	TLSHandshakeVec := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "TLS_duration_seconds",
			Help: "Trace TLS latency.",
		},
		[]string{"method", "url"},
	)
	//tcpLatencyVec
	tcpLatencyVec := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tcp_duration_seconds",
			Help: "Trace tcp latency.",
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
	prometheus.MustRegister(dnsLatencyVec, TLSHandshakeVec, tcpLatencyVec, requestTimer, requestAvgDurationVec)

	//Run roundtrip for each URL
	numURLs := len(configuration.URLmap)
	for i := 0; i < numURLs; i++ {
		// Define functions for the available httptrace.ClientTrace hook
		// functions that we want to instrument.
		timer1 := prometheus.NewTimer(prometheus.ObserverFunc(dnsLatencyVec.WithLabelValues("dns_start", configuration.URLmap[i]).Set))
		timer2 := prometheus.NewTimer(prometheus.ObserverFunc(tcpLatencyVec.WithLabelValues("TCP_start", configuration.URLmap[i]).Set))
		timer3 := prometheus.NewTimer(prometheus.ObserverFunc(TLSHandshakeVec.WithLabelValues("TCP_start", configuration.URLmap[i]).Set))
		trace := &promhttp.InstrumentTrace{
			DNSDone: func(t float64) {
				timer1.ObserveDuration()
			},
			ConnectDone: func(t float64) {
				timer2.ObserveDuration()
			},
			TLSHandshakeStart: func(t float64) {
				timer3 = prometheus.NewTimer(prometheus.ObserverFunc(TLSHandshakeVec.WithLabelValues("TCP_start", configuration.URLmap[i]).Set))
			},
			TLSHandshakeDone: func(t float64) {
				timer3.ObserveDuration()
			},
		}

		// Wrap the default RoundTripper with middleware.
		roundTripper := promhttp.InstrumentRoundTripperTrace(trace,
			promhttp.InstrumentRoundTripperDuration(requestAvgDurationVec, http.DefaultTransport))

		timer0 := prometheus.NewTimer(prometheus.ObserverFunc(requestTimer.WithLabelValues("GET", configuration.URLmap[i]).Set))
		// Set the RoundTripper on the client.
		client.Transport = roundTripper
		log.Printf("roundtripper created")

		//Read and close body for each url
		resp, err := client.Get(configuration.URLmap[i])
		if err != nil {
			log.Printf("error: %v", err)
		}
		resp.Body.Close()
		timer0.ObserveDuration()
		log.Printf("GET run with url: " + configuration.URLmap[i])
	}

	log.Printf("Config: " + fmt.Sprintf("%+v", configuration))
	// The Handler function provides a default handler to expose metrics
	// via an HTTP server. "/metrics" is the usual endpoint for that.
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(configuration.Port, nil))
}
