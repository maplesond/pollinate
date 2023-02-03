# Build the app in the golang container
FROM golang:1.19-alpine as build
WORKDIR /pollinate
COPY go/* ./
RUN go mod download
RUN go build -o pollinate


# Create a simple alpine image with the executable and
# make sure this is executed at startup.  Also make sure
# we are not running as root
FROM alpine
RUN addgroup -S pollinate && adduser -S pollinate -G pollinate
USER pollinate
COPY --from=build /pollinate/pollinate /pollinate
EXPOSE 8000
CMD [ "/pollinate" ]