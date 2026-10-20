FROM golang:1.27-bookworm AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/frontend .

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/frontend ./frontend
COPY templates ./templates
COPY static ./static
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/frontend"]
