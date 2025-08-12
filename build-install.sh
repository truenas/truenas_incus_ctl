#!/bin/sh
./run_tests.sh && go install && truenas_incus_ctl version

