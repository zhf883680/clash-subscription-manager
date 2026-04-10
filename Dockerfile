FROM golang:1.25 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -o /out/clash-subscription-manager .

FROM alpine:3.21

WORKDIR /app

COPY --from=build /out/clash-subscription-manager /app/clash-subscription-manager
COPY templates /app/templates
COPY static /app/static
COPY config.yaml /app/config.yaml

RUN mkdir -p /app/data

EXPOSE 8080

CMD ["/app/clash-subscription-manager"]
