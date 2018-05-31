FROM golang:latest
RUN go get github.com/prometheus/client_golang/prometheus
RUN go get github.com/tkanos/gonfig
RUN mkdir /app
ADD . /app/
WORKDIR /app
RUN go build -o main main.go
CMD ["/app/main"]
