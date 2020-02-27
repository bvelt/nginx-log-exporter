GOOS=linux GOARCH=amd64 go build -ldflags="-w -s"
docker build -t betteries/nginx-log-exporter:latest .
docker run -d -p 9999:9999 -v /var/log/nginx:/nginxlogs betteries/nginx-log-exporter:latest
