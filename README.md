A HTTP statistic exporter for Prometheus written in Go
======
This is a HTTP statistic gathering script written in Go to export metrics to Prometheus

How to check metrics
-----
Run the Script

    $ go run metric1.go

This will continually run on PORT:7777
Go to the /metric extension to see the metrics.
It checks the metrics every time the request is recieved.

Thanks
------
The original idea came from httpstat command (and Dave Cheney's golang implementation), and Taichi Nakashima's package go-httpstat.

ref: https://github.com/davecheney/httpstat https://github.com/tcnksm/go-httpstat
