FROM debian:latest

ENV executable="executable"
ENV multiclient="multiclient"
RUN mkdir /service && mkdir /logs && mkdir /logs/failure_logs
WORKDIR /service

COPY $executable .
COPY $multiclient .
COPY configs.json .

CMD ["sh", "-c", "./$multiclient"]