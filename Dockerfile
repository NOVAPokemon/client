FROM alpine:latest

RUN apk add --no-cache libc6-compat
ENV executable="executable"
RUN mkdir /service
WORKDIR /service

COPY $executable .
COPY configs.json .

CMD ["sh", "-c", "./$executable"]