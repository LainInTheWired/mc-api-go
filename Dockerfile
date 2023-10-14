FROM golang:latest

WORKDIR /go/src

COPY ./ .

RUN go install github.com/cosmtrek/air@latest

CMD ["air", "-c", ".air.toml"]
