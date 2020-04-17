FROM golang:alpine

LABEL maintainer="marcel.parciak@mgmail.com"
ENV TOKEN ""

WORKDIR /etc/postwoman-proxy

COPY . /etc/postwoman-proxy
RUN ./build.sh linux server

EXPOSE 9159/tcp

CMD ["sh", "-c", "/etc/postwoman-proxy/out/linux-server/postwoman-proxy-server --host 0.0.0.0:9159 --token $TOKEN"]
