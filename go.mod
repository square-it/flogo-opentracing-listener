module github.com/square-it/flogo-opentracing-listener

require (
	github.com/apache/thrift v0.12.0
	github.com/opentracing/opentracing-go v1.0.2
	github.com/openzipkin-contrib/zipkin-go-opentracing v0.3.5
	github.com/project-flogo/contrib v0.9.0-alpha.3.0.20190211153431-680ebf186e58
	github.com/project-flogo/core v0.9.0-alpha.4.0.20190213174534-066cf88d3782
	github.com/project-flogo/flow v0.9.0-alpha.3.0.20190211150821-b5f5b5d71381
	github.com/uber/jaeger-client-go v2.15.0+incompatible
	github.com/uber/jaeger-lib v1.5.0
)

replace (
	github.com/project-flogo/core => github.com/skothari-tibco/core v0.0.0-20190215153801-2f9dd9f3fbb7
	github.com/project-flogo/flow => github.com/skothari-tibco/flow v0.9.0-alpha.4.0.20190219031632-b046643f1e87
)
