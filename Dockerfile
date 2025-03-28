FROM golang:1.19-alpine as builder

# Add Maintainer Info
LABEL maintainer="Sam Zhou <sam@mixmedia.com>"

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app
RUN go version \
 && export GO111MODULE=on \
 && export GOPROXY=https://goproxy.io,direct \
 && go mod vendor \
 && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bedrock-claude-proxy \
 && echo "{}" > config.json


######## Start a new stage from scratch #######
FROM golang:1.19-alpine

RUN apk add --update libintl \
    && apk add --no-cache ca-certificates tzdata dumb-init python3 py3-pip \
    && apk add --virtual build_deps gettext  \
    && cp /usr/bin/envsubst /usr/local/bin/envsubst \
    && apk del build_deps

WORKDIR /app

COPY scripts /app/scripts

RUN pip3 install -r /app/scripts/requirements.txt

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/bedrock-claude-proxy .
COPY --from=builder /app/webroot .
COPY --from=builder /app/config.json .

ENV HTTP_LISTEN=0.0.0.0:3000 \
 WEB_ROOT=/app/webroot \
 AWS_BEDROCK_ACCESS_KEY= \
 AWS_BEDROCK_SECRET_KEY= \
 AWS_BEDROCK_REGION= \
 AWS_BEDROCK_MODEL_MAPPINGS="claude-instant-1.2=anthropic.claude-instant-v1,claude-2.0=anthropic.claude-v2,claude-2.1=anthropic.claude-v2:1,claude-3-sonnet-20240229=anthropic.claude-3-sonnet-20240229-v1:0,claude-3-opus-20240229=anthropic.claude-3-opus-20240229-v1:0,claude-3-haiku-20240307=anthropic.claude-3-haiku-20240307-v1:0,claude-3-7-sonnet-20250219=us.anthropic.claude-3-7-sonnet-20250219-v1:0,claude-3-5-sonnet-20241022=anthropic.claude-3-5-sonnet-20241022-v2:0,claude-3-5-haiku-20241022:anthropic.claude-3-5-haiku-20241022-v1:0" \
 AWS_BEDROCK_ANTHROPIC_VERSION_MAPPINGS="2023-06-01=bedrock-2023-05-31" \
 AWS_BEDROCK_ANTHROPIC_DEFAULT_MODEL=anthropic.claude-v2 \
 AWS_BEDROCK_ANTHROPIC_DEFAULT_VERSION=bedrock-2023-05-31 \
 DB_HOST=mysql \
 DB_PORT=3306 \
 DB_USER=bedrock \
 DB_PASSWORD=bedrock_password \
 DB_NAME=bedrock \
 LOG_LEVEL=INFO

EXPOSE 3000

ENTRYPOINT ["dumb-init", "--"]

CMD envsubst < /app/config.json > /app/temp.json \
 && /app/bedrock-claude-proxy -c /app/temp.json

