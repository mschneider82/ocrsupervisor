FROM golang:alpine as builder
RUN mkdir -p /ocrsupervisor
ADD . /ocrsupervisor
WORKDIR /ocrsupervisor
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ocrsupervisor .

FROM alpine
RUN mkdir -p /app
COPY --from=builder /ocrsupervisor/ocrsupervisor /app
RUN chmod +x /app/ocrsupervisor
ENTRYPOINT [ "/app/ocrsupervisor" ]
