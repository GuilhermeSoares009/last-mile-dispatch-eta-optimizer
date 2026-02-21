FROM golang:1.22 AS build
WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /out/dispatch ./cmd/dispatch

FROM gcr.io/distroless/base-debian12
WORKDIR /
COPY --from=build /out/dispatch /dispatch
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/dispatch"]
