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
	"config.json",                //test run
	"/etc/goscraper/config.json", //inside a dockprom setup
	"/app/config.json",           //inside a docker container
}

var path = "log.txt"

//Configuration create config struct
type Configuration struct {
	ScrapeInterval int
	Timeout        int
	Port           string
	URLmap         []string
}

var configuration = Configuration{}

//Create Summaries
// goscraper_dns_latency_gauge gauge vector
var goscraper_dns_latency_gauge = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "goscraper_dns_latency_gauge",
		Help: "Trace dns latency.",
	},
	[]string{"method", "url"},
)

//goscraper_tls_handshake_latency gauge vector
var goscraper_tls_handshake_latency = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "goscraper_tls_handshake_latency",
		Help: "Trace TLS latency.",
	},
	[]string{"method", "url"},
)

//goscraper_tcp_latency_gauge gauge vector
var goscraper_tcp_latency_gauge = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "goscraper_tcp_latency_gauge",
		Help: "Trace tcp latency.",
	}, []string{"method", "url"},
)

//goscraper_total_request_time gauge vector
var goscraper_total_request_time = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "goscraper_total_request_time",
		Help: "Request Timer.",
	}, []string{"method", "url"},
)

// goscraper_request_duration_summary summary vector
var goscraper_request_duration_summary = prometheus.NewSummaryVec(
	prometheus.SummaryOpts{
		Name: "goscraper_request_duration_summary",
		Help: "Request duration in seconds.",
	},
	[]string{"method"},
)

func init() {
	//Create/Find log folder and add new log file
	createFindDir()

	//Register all the metrics with Prometheus
	prometheus.MustRegister(goscraper_dns_latency_gauge, goscraper_tls_handshake_latency, goscraper_tcp_latency_gauge, goscraper_total_request_time, goscraper_request_duration_summary)

	//Capture if the program is exited and logs exit signal
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	go func() {
		select {
		case sig := <-c:
			log.Printf("Got %s signal. Aborting...\n", sig)
			appendFile(fmt.Sprintf("Got %s signal. Aborting...\n", sig), path)
			os.Exit(1)
		}
	}()

	//Grab Details
	log.Printf("confPath %v", confPaths)
	for i := 0; i < len(confPaths); i++ {
		err := gonfig.GetConf(confPaths[i], &configuration)
		if err != nil {
			log.Printf("error: %v", err)
		} else if err == nil {
			log.Printf("config file read from " + confPaths[i])
		}
	}
	log.Printf("Config: " + fmt.Sprintf("%+v", configuration))
	appendFile("Config: "+fmt.Sprintf("%+v", configuration), path)

}

func main() {
	//Set client timeout
	http.DefaultClient.Timeout = time.Duration(configuration.Timeout) * time.Second

	//go func that runs the runRequest func and sleeps based off of config
	funcRan := 0
	go func() {
		for {
			runRequest()
			funcRan++
			log.Printf("Function run %d time(s)", funcRan)
			appendFile(fmt.Sprintf("Function run %d time(s)", funcRan), path)
			time.Sleep(time.Duration(configuration.ScrapeInterval) * time.Second)
		}
	}()

	// The Handler function provides a default handler to expose metrics
	// via an HTTP server. "/metrics" is the usual endpoint for that.
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(configuration.Port, nil)
}

//Take in URL, GET request it, and parse through all the information
func readAndClose(url string) {
	//Read and close body for each url
	resp, err := http.DefaultClient.Get(url)
	if err != nil {
		log.Printf("error: %v", err)
	} else if err == nil {
		resp.Body.Close()
	}
}

//Create Trace
func createTrace(url string) promhttp.InstrumentTrace {
	// Define functions for the available httptrace.ClientTrace hook
	// functions that we want to instrument.
	timer1 := prometheus.NewTimer(prometheus.ObserverFunc(goscraper_dns_latency_gauge.WithLabelValues("dns_start", url).Set))
	timer2 := prometheus.NewTimer(prometheus.ObserverFunc(goscraper_tcp_latency_gauge.WithLabelValues("TCP_start", url).Set))
	timer3 := prometheus.NewTimer(prometheus.ObserverFunc(goscraper_tls_handshake_latency.WithLabelValues("TCP_start", url).Set))
	trace := &promhttp.InstrumentTrace{
		DNSDone: func(t float64) {
			timer1.ObserveDuration()
		},
		ConnectDone: func(t float64) {
			timer2.ObserveDuration()
		},
		TLSHandshakeStart: func(t float64) {
			timer3 = prometheus.NewTimer(prometheus.ObserverFunc(goscraper_tls_handshake_latency.WithLabelValues("TCP_start", url).Set))
		},
		TLSHandshakeDone: func(t float64) {
			timer3.ObserveDuration()
		},
	}
	return *trace
}

func runRequest() {
	//Run roundtrip for each URL
	for i := range configuration.URLmap {
		//Create trace for each URL
		trace := createTrace(configuration.URLmap[i])
		log.Printf("trace created")
		appendFile("trace created", path)

		// Wrap the default RoundTripper with middleware.
		roundTripper := promhttp.InstrumentRoundTripperTrace(&trace,
			promhttp.InstrumentRoundTripperDuration(goscraper_request_duration_summary, http.DefaultTransport))

		// Set the RoundTripper on the client.
		http.DefaultClient.Transport = roundTripper
		log.Printf("roundtripper created")
		appendFile("roundtripper created", path)

		//Time the request
		timer0 := prometheus.NewTimer(prometheus.ObserverFunc(goscraper_total_request_time.WithLabelValues("GET", configuration.URLmap[i]).Set))
		readAndClose(configuration.URLmap[i])
		timer0.ObserveDuration()
		log.Printf("GET run with url: " + configuration.URLmap[i])
		appendFile("GET run with url: "+configuration.URLmap[i], path)
	}
}

func createFindDir() {
	//Check if log folder exists
	if _, err := os.Stat("logs"); os.IsNotExist(err) {
		os.Mkdir("logs", os.ModePerm)
		log.Printf("Log Folder Created")
		createFile()
	} else {
		log.Printf("Log Folder Found")
		createFile()
	}
}

func createFile() {
	// detect if log file exists
	var _, err = os.Stat(path)

	// create log file if does not exists
	if os.IsNotExist(err) {
		path = "logs/log" + time.Now().Format("2006-01-0215:04:05") + ".txt"
		var file, err = os.Create(path)
		if err != nil {
			return
		}
		defer file.Close()
		log.Println("done creating file", path)
	} else {
		log.Println("file found", path)
	}
}

//function to append strings to a text file
func appendFile(str, path string) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("failed opening file: %s", err)
	}
	defer file.Close()
	t := time.Now().Format("2006-01-0215:04:05")
	_, err = file.WriteString(t + " " + str + "\n")
	if err != nil {
		log.Fatalf("failed writing to file: %s", err)
	}
}
