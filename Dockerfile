# The frontend is built by Node, embedded by Go, and shipped in a distroless
# image that contains only the binary. Node never runs at runtime.
FROM node:24-alpine AS web
RUN npm install -g pnpm@11
WORKDIR /src/web
COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build

FROM golang:1.26 AS build
ARG VERSION=dev
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
COPY --from=web /src/internal/web/dist ./internal/web/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /understudy ./cmd/understudy

FROM gcr.io/distroless/static:nonroot
COPY --from=build /understudy /understudy
ENTRYPOINT ["/understudy"]
