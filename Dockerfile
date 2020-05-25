FROM debian:latest

ENV executable="executable"
RUN mkdir /service
WORKDIR /service

COPY $executable .
COPY configs.json .

CMD ["sh", "-c", "./$executable"]