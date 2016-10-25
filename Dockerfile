FROM alpine:latest

# Install bash and curl
RUN apk add --update bash curl && rm -rf /var/cache/apk/*

ADD trigg.sh /usr/bin/trigg
RUN chmod +x /usr/bin/trigg

ADD check_args.sh /usr/bin/check_args.sh
RUN chmod +x /usr/bin/check_args.sh

ADD app /usr/bin/app
RUN chmod +x /usr/bin/app

RUN mkdir -p /config
VOLUME /config

EXPOSE 80

ENTRYPOINT ["/usr/bin/app"]
CMD ["-listen-addr", "0.0.0.0:80", "-configdir", "/config"]
