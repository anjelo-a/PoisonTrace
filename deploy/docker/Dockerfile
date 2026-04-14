FROM golang:1.22 AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/scanner ./cmd/scanner

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/scanner /scanner
ENTRYPOINT ["/scanner"]
