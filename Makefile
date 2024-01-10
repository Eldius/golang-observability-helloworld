
hello:
	$(eval COLLECTOR_TRACE_HOST := $(shell ./fetch_ports.sh otel-collector 55689 observability))
	$(eval COLLECTOR_METRICS_HOST := $(shell ./fetch_ports.sh otel-collector 55690 observability))

	HELLO_TELEMETRY_METRICS_ENDPOINT=$(COLLECTOR_METRICS_HOST) \
	HELLO_TELEMETRY_TRACES_ENDPOINT=$(COLLECTOR_TRACE_HOST)  go \
		run \
		./cmd/hello
