FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /office-pong .

FROM alpine:3.20
COPY --from=build /office-pong /office-pong
EXPOSE 8080
ENTRYPOINT ["/office-pong"]
