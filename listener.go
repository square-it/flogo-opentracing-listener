package opentracing

import (
	"sync"

	flowevent "github.com/TIBCOSoftware/flogo-contrib/action/flow/event"
	"github.com/TIBCOSoftware/flogo-lib/core/event"
	"github.com/TIBCOSoftware/flogo-lib/logger"
	"github.com/opentracing/opentracing-go"

	_ "github.com/apache/thrift/lib/go/thrift" // required to define the dependency as a constraint in Gopkg.toml
)

var (
	lock  = &sync.RWMutex{}
	spans map[string]opentracing.Span
)

type OpenTracingListener struct {
	name   string
	logger logger.Logger
}

func (otl *OpenTracingListener) Name() string {
	return otl.name
}

func (otl *OpenTracingListener) HandleEvent(evt *event.EventContext) error {
	// Handle flowevent events and ignore remaining
	switch t := evt.GetEvent().(type) {
	case flowevent.FlowEvent:
		otl.logger.Debugf("Name: %s, ID: %s, Status: %s ", t.FlowName(), t.FlowID(), t.FlowStatus())
		switch t.FlowStatus() {
		case flowevent.STARTED:
			startFlowSpan(t)
		case flowevent.COMPLETED:
			finishFlowSpan(t)
		}
	case flowevent.TaskEvent:
		otl.logger.Debugf("Name: %s, FID: %s, Status: %s ", t.TaskName(), t.FlowID(), t.TaskStatus())
		switch t.TaskStatus() {
		case flowevent.STARTED:
			startTaskSpan(t)
		case flowevent.COMPLETED:
			finishTaskSpan(t)
		}
          default:
                   otl.logger.Debugf("Event of type %T is not supported",t)
	}

	return nil
}

func init() {
	initFromEnvVars()

	spans = make(map[string]opentracing.Span)

	event.RegisterEventListener(&OpenTracingListener{name: "OpenTracingListener", logger: logger.GetLogger("open-tracing-listener")}, []string{flowevent.FLOW_EVENT_TYPE, flowevent.TASK_EVENT_TYPE})
}

func startFlowSpan(flowEvent flowevent.FlowEvent) {
	span := opentracing.StartSpan(flowEvent.FlowName(), opentracing.StartTime(flowEvent.Time()))
	span.SetTag("type", "flogo:flowevent")

	lock.Lock()
	defer lock.Unlock()
	spans[flowEvent.FlowID()] = span
}

func finishFlowSpan(flowEvent flowevent.FlowEvent) {
	lock.Lock()
	defer lock.Unlock()
	span := spans[flowEvent.FlowID()]

	span.FinishWithOptions(opentracing.FinishOptions{FinishTime: flowEvent.Time()})
}

func startTaskSpan(taskEvent flowevent.TaskEvent) {
	lock.Lock()
	defer lock.Unlock()
	flowSpan := spans[taskEvent.FlowID()]

	span := opentracing.StartSpan(taskEvent.TaskName(), opentracing.ChildOf(flowSpan.Context()), opentracing.StartTime(taskEvent.Time()))
	span.SetTag("type", "flogo:activity")

	spans[taskEvent.FlowID()+taskEvent.TaskName()] = span
}

func finishTaskSpan(taskEvent flowevent.TaskEvent) {
	lock.Lock()
	defer lock.Unlock()
	span := spans[taskEvent.FlowID()+taskEvent.TaskName()]

	span.FinishWithOptions(opentracing.FinishOptions{FinishTime: taskEvent.Time()})
}
