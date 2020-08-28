FROM golang:alpine

LABEL maintainer="me+proxyscotch@samjakob.com"

WORKDIR /etc/proxyscotch

COPY . /etc/proxyscotch
RUN ./build.sh linux server

EXPOSE 9159/tcp

# this should be a standard user with the users group on alpine
USER 1000:100

CMD ["sh", "-c", "/etc/proxyscotch/out/linux-server/proxyscotch-server-* --host 0.0.0.0:9159"]
