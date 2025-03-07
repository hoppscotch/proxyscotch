FROM golang:1.23.7-alpine3.21 AS builder

WORKDIR /etc/proxyscotch

COPY . /etc/proxyscotch
RUN ./build.sh linux server



# The actual final container
FROM alpine:3.21.3
LABEL maintainer="support@hoppscotch.io"

EXPOSE 9159/tcp

COPY --from=builder /etc/proxyscotch/out/linux-server/proxyscotch-server-linux-* /usr/bin/proxyscotch

# this should be a standard user with the users group on alpine
USER 1000:100

CMD ["proxyscotch", "--host", "0.0.0.0:9159"]
