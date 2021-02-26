FROM arkhipovkm/tdlib:amd64-1.7.0
FROM golang:1.15.7-alpine3.12
RUN apk --no-cache add alpine-sdk linux-headers git zlib-dev openssl-dev
COPY --from=0 /td/tdlib /usr/local
WORKDIR /go/src/github.com/arkhipovkm/unifeed-go
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY ./db ./db/
COPY ./main.go ./
RUN go build
CMD ./unifeed-go