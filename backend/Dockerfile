FROM public.ecr.aws/docker/library/golang:1.25 AS build
WORKDIR /app
COPY go.mod go.sum ./
ENV GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
ENV GOSUMDB=off
RUN go mod download
COPY . .
RUN go build -o /bin/didi-api ./cmd/api && \
    go build -o /bin/didi-gateway ./cmd/gateway && \
    go build -o /bin/didi-auth-service ./cmd/auth-service && \
    go build -o /bin/didi-user-service ./cmd/user-service && \
    go build -o /bin/didi-driver-service ./cmd/driver-service && \
    go build -o /bin/didi-location-service ./cmd/location-service && \
    go build -o /bin/didi-order-service ./cmd/order-service && \
    go build -o /bin/didi-dispatch-service ./cmd/dispatch-service && \
    go build -o /bin/didi-payment-service ./cmd/payment-service && \
    go build -o /bin/didi-realtime-service ./cmd/realtime-service && \
    go build -o /bin/didi-admin-service ./cmd/admin-service

FROM public.ecr.aws/docker/library/debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends curl ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=build /bin/didi-api /bin/didi-api
COPY --from=build /bin/didi-gateway /bin/didi-gateway
COPY --from=build /bin/didi-auth-service /bin/didi-auth-service
COPY --from=build /bin/didi-user-service /bin/didi-user-service
COPY --from=build /bin/didi-driver-service /bin/didi-driver-service
COPY --from=build /bin/didi-location-service /bin/didi-location-service
COPY --from=build /bin/didi-order-service /bin/didi-order-service
COPY --from=build /bin/didi-dispatch-service /bin/didi-dispatch-service
COPY --from=build /bin/didi-payment-service /bin/didi-payment-service
COPY --from=build /bin/didi-realtime-service /bin/didi-realtime-service
COPY --from=build /bin/didi-admin-service /bin/didi-admin-service
EXPOSE 8080
CMD ["/bin/didi-api"]
