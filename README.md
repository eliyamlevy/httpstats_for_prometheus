A HTTP statistic exporter for Prometheus written in Go
======
This is a HTTP statistic gathering script written in Go to export metrics to Prometheus.

The config file allows for adding multiple links, and changing the port. Default port is ":7777", just remember to change it in the dockerfile as well if you want a different port.

To Build:
-------

$ go build -o goscraper main.go

Thanks
------
The original idea came from httpstat command (and Dave Cheney's golang implementation), and Taichi Nakashima's package go-httpstat.

ref: https://github.com/davecheney/httpstat https://github.com/tcnksm/go-httpstat
