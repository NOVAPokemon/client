#!/bin/sh

timeout=$TIMEOUT
measurement_counts=$MEASURE_COUNTS

bwm-ng -t "$timeout" -I eth0 -o csv -c "$measurement_counts" -u bits -T rate -F /bandwidth_stats/"$CLIENT_ID".csv -D 1