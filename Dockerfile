FROM golang:alpine

LABEL maintainer="marcel.parciak@gmail.com"

WORKDIR /etc/postwoman-proxy

COPY . /etc/postwoman-proxy
RUN ./build.sh linux server

EXPOSE 9159/tcp

# this should be a standard user with the users group on alpine
USER 1000:100

CMD ["sh", "-c", "/etc/postwoman-proxy/out/linux-server/postwoman-proxy-server --host 0.0.0.0:9159"]
