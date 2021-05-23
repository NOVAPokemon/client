FROM debian:latest

ENV executable="executable"
ENV multiclient="multiclient"
RUN mkdir /service && mkdir /logs && mkdir /logs/failure_logs
WORKDIR /service

COPY location_tags.json .
COPY delays_config.json .
COPY client_delays.json .
COPY cells_to_region.json .
COPY location_weights.json .
COPY regions_to_area.json .
COPY $executable .
COPY $multiclient .
COPY configs.json .
COPY lats.txt .
COPY locations.json .

CMD ["sh", "-c", "./$multiclient"]