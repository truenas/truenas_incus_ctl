#!/bin/sh
go test ./cmd && go build && go install && truenas_incus_ctl version

