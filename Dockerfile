#first stage - builder

FROM golang:latest as builder

COPY . /app
WORKDIR /app

ENV GO111MODULE=auto
RUN GOARCH=amd64 CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o app main.go

#second stage

FROM alpine:latest

RUN apk --no-cache add ca-certificates mailcap && addgroup -S app && adduser -S app -G app

RUN apk update && apk add curl git bash && curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.21.4/bin/linux/amd64/kubectl && chmod 755 kubectl && mv kubectl /bin/kubectl

USER app
WORKDIR /app

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=builder /app/app .

CMD ["./app"]
