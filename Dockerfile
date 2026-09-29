FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

COPY mcstatus /usr/local/bin/mcstatus

EXPOSE 3001

ENTRYPOINT ["/usr/local/bin/mcstatus"]
