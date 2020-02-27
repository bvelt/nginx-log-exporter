FROM alpine:latest
ADD nginx-log-exporter /nginx-log-exporter
ADD config.yml /config.yml
EXPOSE 9999
ENTRYPOINT [ "/nginx-log-exporter" ]