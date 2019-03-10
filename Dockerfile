FROM golang:alpine as builder
WORKDIR /home/compiler
RUN apk --update add ca-certificates
COPY ./app .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -mod vendor

FROM scratch as runner
LABEL maintainer="Bogdan Condurache <bogdan@condurache.me>"
ENV PATH=/bin
COPY --from=builder /home/compiler/online-compiler /bin/compiler
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
ENTRYPOINT [ "compiler" ]
