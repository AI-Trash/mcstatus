FROM alpine:latest

ARG TARGETPLATFORM

RUN apk --no-cache add ca-certificates tzdata

COPY $TARGETPLATFORM/mcstatus /usr/local/bin/mcstatus
RUN chmod +x /usr/local/bin/mcstatus
EXPOSE 3001

ENTRYPOINT ["/usr/local/bin/mcstatus"]
