FROM golang:latest

WORKDIR /go/src

COPY src/ .
RUN apt-get update && apt-get -y install vim

RUN go install github.com/cosmtrek/air@latest

# CMD ["air", "-c", ".air.toml"]
