FROM golang:1.25-bookworm AS builder

WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -trimpath -ldflags="-s -w" -o /bin/ticketowl ./cmd/ticketowl

# ---
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /bin/ticketowl /ticketowl
COPY migrations/ /migrations/

EXPOSE 8082
USER nonroot:nonroot

ENTRYPOINT ["/ticketowl"]
