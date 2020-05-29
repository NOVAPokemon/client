FROM debian:latest

ENV executable="executable"
ENV multiclient="multiclient"
RUN mkdir /service
WORKDIR /service

COPY $executable .
COPY $multiclient .
COPY configs.json .

CMD ["sh", "-c", "./$multiclient"]