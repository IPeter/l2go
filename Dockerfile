# syntax=docker/dockerfile:1

# One image for every server; the command picks the role:
#   docker run l2go login -config /etc/l2go/server.json
#   docker run l2go game -id 1 -config /etc/l2go/server.json

FROM golang:1.27 AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY cmd ./cmd
COPY internal ./internal
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/l2go ./cmd/l2go


FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/l2go /usr/local/bin/l2go

# Runtime files live under /opt/l2go, the working directory, so relative
# paths in the config ("scripts") point here.
#   scripts/  Lua scripts, copied as defaults. docker compose mounts
#             ./scripts over them read-only for live reload.
# Coming later:
#   data/     game data (Plan 03), copied into the image:
#             COPY data /opt/l2go/data
COPY scripts /opt/l2go/scripts
WORKDIR /opt/l2go

# 2106: clients to login, 9413: game servers to login, 7777: clients to game.
EXPOSE 2106 9413 7777

ENTRYPOINT ["/usr/local/bin/l2go"]
CMD ["login", "-config", "/etc/l2go/server.json"]
