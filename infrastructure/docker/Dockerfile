FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY apps ./apps
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -o /wisp ./apps/api && CGO_ENABLED=0 go build -trimpath -o /wisp-ingest ./apps/ingest

FROM alpine:3.23
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 wisp
COPY --from=build /wisp /usr/local/bin/wisp
COPY --from=build /wisp-ingest /usr/local/bin/wisp-ingest
USER wisp
EXPOSE 8090
ENTRYPOINT ["wisp"]
CMD ["serve"]
