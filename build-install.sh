#!/bin/sh
go test ./cmd && go build && ./test/test_home_fallback.sh && go install && truenas_incus_ctl version

