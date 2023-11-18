FROM golang:1.21 as build
WORKDIR /build
# Copy dependencies list
COPY go.mod go.sum ./

COPY . .

# Installs Go dependencies
RUN go mod download

# Build with optional lambda.norpc tag
RUN go build -tags lambda.norpc -o certmaster certmaster

# Copy artifacts to a clean image
FROM public.ecr.aws/lambda/provided:al2023
COPY --from=build /build/certmaster ./certmaster
ENTRYPOINT [ "./certmaster", "lambda" ]