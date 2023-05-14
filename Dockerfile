FROM alpine:3.14 as certs
RUN apk add ca-certificates

FROM scratch AS kubeconform
LABEL org.opencontainers.image.authors="yann@mandragor.org" \
      org.opencontainers.image.source="https://github.com/yannh/arpicee/" \
      org.opencontainers.image.description="Arpicee - The Remote Procedure Framework" \
      org.opencontainers.image.documentation="https://github.com/yannh/arpicee/" \
      org.opencontainers.image.licenses="Apache License 2.0" \
      org.opencontainers.image.title="arpicee" \
      org.opencontainers.image.url="https://github.com/yannh/arpicee/"
MAINTAINER Yann HAMON <yann@mandragor.org>
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY arpicee /
ENTRYPOINT ["/arpicee-slackbot"]
