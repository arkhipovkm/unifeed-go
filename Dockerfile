FROM alpine:3.13.1

RUN apk --no-cache update && apk --no-cache upgrade
RUN apk --no-chache add --update alpine-sdk linux-headers git zlib-dev openssl-dev gperf php cmake
RUN git clone --depth 1 --branch v1.7.0 https://github.com/tdlib/td.git
RUN cd td && rm -rf build && mkdir build
WORKDIR /td/build/
RUN cmake -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX:PATH=../tdlib ..
RUN cmake --build . --target install

FROM golang:1.15.7-alpine3.13
RUN apk --no-cache update && apk --no-cache upgrade
RUN apk --no-chache add --update alpine-sdk linux-headers git zlib-dev openssl-dev
COPY --from=0 /td/tdlib /usr/local
WORKDIR /go/src/github.com/arkhipovkm/unifeed-go
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY ./db ./db/
COPY ./main.go ./
RUN go build
CMD ./unifeed-go