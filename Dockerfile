FROM golang:1.18-alpine AS builder

RUN apk update && \
    apk add --no-cache --update make bash git ca-certificates && \
    update-ca-certificates

WORKDIR /go/src/otc-rds-operator

COPY . .

RUN make build

FROM alpine:3.15

RUN addgroup -S appgroup && adduser -S appuser -G appgroup -h /home/appuser
WORKDIR /home/appuser

COPY --from=builder /go/src/otc-rds-operator/bin/otc-rds-operator /home/appuser/otc-rds-operator
USER appuser

CMD [ "/home/appuser/otc-rds-operator" ]
