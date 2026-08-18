FROM golang:1.26 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG COMPONENT

RUN CGO_ENABLED=0 GOOS=linux \
    go build -o /out/app ./cmd/${COMPONENT}

FROM gcr.io/distroless/static-debian12

COPY --from=builder /out/app /app

ENTRYPOINT ["/app"]