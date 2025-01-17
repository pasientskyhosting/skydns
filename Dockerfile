FROM alpine:3.15.4
MAINTAINER Miek Gieben <miek@miek.nl> (@miekg)

RUN apk --update add bind-tools && rm -rf /var/cache/apk/*

ADD skydns skydns

EXPOSE 53 53/udp
ENTRYPOINT ["/skydns"]
