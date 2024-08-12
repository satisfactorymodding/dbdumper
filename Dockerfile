FROM alpine
RUN apk add --no-cache postgresql16-client
COPY dbdumper /
ENTRYPOINT ["/dbdumper"]