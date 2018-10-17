package opentracing

import (
	"sync"

	"github.com/TIBCOSoftware/flogo-contrib/action/flow/instance"
	"github.com/TIBCOSoftware/flogo-lib/core/events"
	"github.com/TIBCOSoftware/flogo-lib/logger"
	"github.com/opentracing/opentracing-go"
)

var (
	lock = &sync.RWMutex{}
	spans map[string]opentracing.Span
)

type OpenTracingListener struct {
	name string
	logger logger.Logger
}

func (otl *OpenTracingListener) Name() string {
	return otl.name
}

func (otl *OpenTracingListener) EventTypes() []string {
	return []string{instance.FLOW_EVENT_TYPE}
}

func (otl *OpenTracingListener) HandleEvent(evt *events.EventContext) error {
	// Handle flow events and ignore remaining
	if evt.GetType() == instance.FLOW_EVENT_TYPE {
		switch t := evt.GetEvent().(type) {
		case instance.FlowEvent:
			otl.logger.Infof("Name: %s, ID: %s, Status: %s ", t.Name(), t.ID(), t.Status())
			switch t.Status() {
			case instance.STARTED:
				startFlowSpan(t)
			case instance.COMPLETED:
				finishFlowSpan(t)
			}
		case instance.TaskEvent:
			switch t.Status() {
			case instance.STARTED:
				startTaskSpan(t)
			case instance.COMPLETED:
				finishTaskSpan(t)
			}
			otl.logger.Infof("Name: %s, FID: %s, Status: %s ", t.Name(), t.FlowID(), t.Status())
		}
	}
	return nil
}

func init() {
	initFromEnvVars()

	spans = make(map[string]opentracing.Span)

	events.RegisterEventListener(&OpenTracingListener{name: "OpenTracingListener", logger: logger.GetLogger("open-tracing-listener")})
}

func startFlowSpan(flowEvent instance.FlowEvent) {
	span := opentracing.StartSpan(flowEvent.Name(), opentracing.StartTime(flowEvent.Time()))
	span.SetTag("type", "flogo:flow")

	lock.Lock()
	defer lock.Unlock()
	spans[flowEvent.ID()] = span
}

func finishFlowSpan(flowEvent instance.FlowEvent) {
	lock.Lock()
	defer lock.Unlock()
	span := spans[flowEvent.ID()]

	span.FinishWithOptions(opentracing.FinishOptions{FinishTime: flowEvent.Time()})
}

func startTaskSpan(taskEvent instance.TaskEvent) {
	lock.Lock()
	defer lock.Unlock()
	flowSpan := spans[taskEvent.FlowID()]

	span := opentracing.StartSpan(taskEvent.Name(), opentracing.ChildOf(flowSpan.Context()), opentracing.StartTime(taskEvent.Time()))
	span.SetTag("type", "flogo:activity")

	spans[taskEvent.FlowID() + taskEvent.Name()] = span
}

func finishTaskSpan(taskEvent instance.TaskEvent) {
	lock.Lock()
	defer lock.Unlock()
	span := spans[taskEvent.FlowID() + taskEvent.Name()]

	span.FinishWithOptions(opentracing.FinishOptions{FinishTime: taskEvent.Time()})
}

