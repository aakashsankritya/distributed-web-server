# Base image for the runtime environment
FROM alpine:3.18

# Set the working directory to /backend
WORKDIR "/backend"

# CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o ./out/webserver .

COPY ./app/out/webserver /backend/webserver

# COPY --from=build ./app/out/webserver /backend/webserver
RUN chmod +x /backend/webserver

EXPOSE 8080
WORKDIR /
# Run the webserver binary when the container starts
CMD ["./backend/webserver"]
# CMD ["tail", "-f", "/dev/null"]
