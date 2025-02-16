FROM golang:1.23

RUN apt update && apt install libpcap-dev -y
WORKDIR /build
COPY . ./
RUN CGO_ENABLED=1 go build -a -tags netgo -ldflags '-w' -o ip-blackcage ./cmd

FROM debian:12
RUN apt update && apt install libpcap-dev -y
COPY --from=0 /build/ip-blackcage /bin/

ENTRYPOINT [ "/bin/ip-blackcage" ]