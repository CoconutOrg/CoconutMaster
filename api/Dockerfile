FROM mcr.microsoft.com/oss/go/microsoft/golang:1.25-azurelinux3.0

ENV CGO_ENABLED=1
ENV GOEXPERIMENT=systemcrypto
ENV GODEBUG=fips140=on

WORKDIR /app

COPY . .

RUN go mod tidy
RUN go build -tags=requirefips -o ./build/api/main cmd/*.go

CMD ["./build/api/main"]
