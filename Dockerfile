FROM debian:stable-slim AS builder

RUN apt update \
	&& apt install --no-install-recommends -y make git ca-certificates \
	libncurses6 libtinfo6 libc6 autoconf automake g++ \
	&& apt clean \
	&& rm -rf /var/lib/apt/lists/*
RUN git clone https://github.com/vgropp/bwm-ng.git
WORKDIR /bwm-ng
RUN ./autogen.sh && make && make install


FROM debian:stable-slim

COPY --from=builder /usr/local/bin/bwm-ng /usr/local/bin/
ENV executable="executable"
ENV multiclient="multiclient"
WORKDIR /
RUN mkdir /service && mkdir /logs && mkdir /logs/failure_logs
WORKDIR /service

COPY start_recording.sh .
RUN chmod +x start_recording.sh
COPY location_tags.json .
COPY delays_config.json .
COPY client_delays.json .
COPY cells_to_region.json .
COPY location_weights.json .
COPY regions_to_area.json .
COPY $executable .
COPY $multiclient .
COPY configs.json .
COPY run_script.sh .
COPY lats.txt .
COPY locations.json .
RUN chmod +x run_script.sh

CMD ["sh", "-c", "./run_script.sh"]