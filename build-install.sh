#!/bin/sh
./run_tests.sh && go install && "$(go env GOPATH)/bin/truenas_incus_ctl" version
