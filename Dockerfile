FROM golang:latest
RUN go get github.com/prometheus/client_golang/prometheus
RUN go get github.com/tkanos/gonfig
RUN mkdir /app
ADD . /app/
WORKDIR /app
RUN ls -R /app
RUN go run /app/main.go
