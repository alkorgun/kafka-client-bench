#!/bin/bash

set -e

clickhouse-client --queries-file=/docker-entrypoint-initdb.d/schema.sql
