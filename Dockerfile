FROM golang:1.24.2-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ./bin/auth-service ./cmd

FROM scratch

COPY --from=builder /app/bin/auth-service /bin/server

LABEL maintainer="vadimdominik2005@gmailcom"
LABEL version="1.0.0"
LABEL description="Auth Service API"

ENTRYPOINT [ "/bin/server" ]
CMD [ "-configPath=./config.yml" ]
