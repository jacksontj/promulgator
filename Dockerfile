FROM       golang:alpine as builder

COPY . /go/src/github.com/jacksontj/promxy
RUN cd /go/src/github.com/jacksontj/promxy/cmd/promxy && CGO_ENABLED=0 go build -mod=vendor
RUN cd /go/src/github.com/jacksontj/promxy/cmd/remote_write_exporter && CGO_ENABLED=0 go build -mod=vendor

FROM alpine:3.11.6
MAINTAINER Thomas Jackson <jacksontj.89@gmail.com>
EXPOSE     8082

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/src/github.com/jacksontj/promxy/cmd/promxy/promxy /bin/promxy
COPY --from=builder /go/src/github.com/jacksontj/promxy/cmd/remote_write_exporter/remote_write_exporter /bin/remote_write_exporter

USER       nobody

ENTRYPOINT [ "/bin/promxy" ]

