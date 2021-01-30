FROM golang:1.15.6-alpine AS build_deps

RUN apk add --no-cache git

WORKDIR /workspace
ENV GO111MODULE=on

COPY go.mod .
COPY go.sum .

RUN go mod download

FROM build_deps AS build

COPY . .

RUN CGO_ENABLED=0 go build -o webhook -ldflags '-w -extldflags "-static"' .

FROM build_deps AS build_dlv

RUN apk add --no-cache make

WORKDIR /dlv

RUN git clone https://github.com/go-delve/delve.git .

RUN make install

FROM alpine:3.12

RUN apk add --no-cache ca-certificates

COPY --from=build /workspace/webhook /usr/local/bin/webhook

COPY --from=build_dlv /go/bin/dlv /usr/local/bin/dlv

ENTRYPOINT ["dlv","--listen=:40000","--headless=true","--api-version=2","--accept-multiclient","exec","/usr/local/bin/webhook","--"]
