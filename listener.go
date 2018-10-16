package opentracing

import (
	"github.com/TIBCOSoftware/flogo-contrib/action/flow/instance"
	"github.com/opentracing/opentracing-go"
	"sync"
)

var (
	lock = &sync.RWMutex{}
	spans map[string]opentracing.Span
)

func init() {
	initFromEnvVars()

	spans = make(map[string]opentracing.Span)

	instance.RegisterFlowEventListener(opentracingFlowEventListener)
	instance.RegisterTaskEventListener(opentracingTaskEventListener)
}

func startFlowSpan(flowEventContext *instance.FlowEventContext) {
	span := opentracing.StartSpan(flowEventContext.Name())
	span.SetTag("type", "flogo:flow")

	lock.Lock()
	defer lock.Unlock()
	spans[flowEventContext.ID()] = span
}

func finishFlowSpan(flowEventContext *instance.FlowEventContext) {
	lock.Lock()
	defer lock.Unlock()
	span := spans[flowEventContext.ID()]

	span.Finish()
}

func startTaskSpan(taskEventContext *instance.TaskEventContext) {
	flowSpan := spans[taskEventContext.FlowID()]

	span := opentracing.StartSpan(taskEventContext.Name(), opentracing.ChildOf(flowSpan.Context()))
	span.SetTag("type", "flogo:activity")

	lock.Lock()
	defer lock.Unlock()
	spans[taskEventContext.FlowID() + taskEventContext.Name()] = span
}

func finishTaskSpan(taskEventContext *instance.TaskEventContext) {
	lock.Lock()
	defer lock.Unlock()
	span := spans[taskEventContext.FlowID() + taskEventContext.Name()]

	span.Finish()
}

func opentracingFlowEventListener(flowEventContext *instance.FlowEventContext) {
	status := flowEventContext.Status()

	switch status {
	//case instance.Created:
	case instance.Started:
		startFlowSpan(flowEventContext)
	//case instance.Cancelled:
	case instance.Completed:
		finishFlowSpan(flowEventContext)
	//case instance.Failed:
	}

}

func opentracingTaskEventListener(taskEventContext *instance.TaskEventContext) {
	status := taskEventContext.Status()

	switch status {
	//case instance.Created:
	//case instance.Scheduled:
	//case instance.Skipped:
	case instance.Started:
		startTaskSpan(taskEventContext)
	//case instance.Failed:
	case instance.Completed:
		finishTaskSpan(taskEventContext)
	//case instance.Waiting:
	}
}
