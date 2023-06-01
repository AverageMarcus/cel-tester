FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.19-alpine as builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM

RUN apk update && apk add -U --no-cache ca-certificates

WORKDIR /app/
ADD go.mod go.sum ./
ADD main.go ./
RUN CGO_ENABLED=0 GOOS=js GOARCH=wasm go build -ldflags="-w -s" -o lib.wasm ./main.go

FROM nginx:1.25
WORKDIR /usr/share/nginx/html
COPY --from=builder /app/lib.wasm /usr/share/nginx/html/lib.wasm
ADD index.html wasm_exec.js ./
RUN chown -R nginx:nginx ./
