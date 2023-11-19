FROM golang:1.21 as build

WORKDIR /build

# Copy dependencies list
COPY go.mod go.sum ./

# Installs Go dependencies
RUN go mod download

COPY . .

# Build with optional lambda.norpc tag
RUN go build -tags lambda.norpc -o certmaster github.com/poundifdef/certmaster

# Copy artifacts to a clean image
FROM public.ecr.aws/lambda/provided:al2023
COPY --from=build /build/certmaster ./certmaster
ENTRYPOINT ["./certmaster", "lambda"]