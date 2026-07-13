FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/polygeo ./cmd/polygeo

FROM alpine:3.22
RUN addgroup -S polygeo && adduser -S -G polygeo polygeo
WORKDIR /app
COPY --from=build /out/polygeo /usr/local/bin/polygeo
COPY data ./data
USER polygeo
EXPOSE 8080
ENV POLYGEO_ADDR=:8080
ENV POLYGEO_DATA_DIR=/app/data
ENTRYPOINT ["polygeo"]
