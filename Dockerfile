# build stage

FROM golang:1.21-alpine3.17 AS build_stage

RUN apk add --no-cache ca-certificates git

COPY .ignore/netrc /root/.netrc
RUN chmod 600 /root/.netrc

WORKDIR /app

COPY . .

RUN go mod download

EXPOSE 8080

RUN CGO_ENABLED=0 GOOS=linux go build -o /noble-cctp-relayer

# deploy stage

FROM alpine:latest

WORKDIR /

COPY --from=build_stage /noble-cctp-relayer /noble-cctp-relayer

ENTRYPOINT ["./noble-cctp-relayer", "start", "--config", "./config/testnet.yaml"]