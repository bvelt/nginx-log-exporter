FROM golang:latest
ADD nginx-log-exporter .
ADD config.yml .
EXPOSE 9999
ENTRYPOINT [ "./nginx-log-exporter" ]