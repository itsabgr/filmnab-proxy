FROM golang:1.20
WORKDIR /src
COPY vendor vendor
COPY go.mod go.sum ./
COPY *.go ./
COPY config.yaml /etc/s3proxy.yaml
RUN go test ./...
RUN go install .
CMD s3proxy -c /etc/s3proxy.yaml
