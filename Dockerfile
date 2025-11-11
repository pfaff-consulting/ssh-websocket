FROM golang:1.25.3-alpine3.22 AS build
WORKDIR /build
COPY . .
RUN go build -o /ssh-websocket

FROM scratch AS final
COPY --from=build /ssh-websocket /ssh-websocket

EXPOSE 2280
ENTRYPOINT [ "/ssh-websocket" ]
