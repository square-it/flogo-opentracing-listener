package opentracing

import (
	"encoding/json"
	"fmt"
	_ "github.com/apache/thrift/lib/go/thrift"
	_ "github.com/project-flogo/contrib/activity/log"
	_ "github.com/project-flogo/contrib/trigger/rest"
	"github.com/project-flogo/core/app"
	"github.com/project-flogo/core/engine"
	logger "github.com/project-flogo/core/support/log"
	_ "github.com/project-flogo/flow"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"testing"
)

const flogoJSON string = `{
  "name": "flogo-opentracing-sample",
  "type": "flogo:app",
  "version": "0.0.1",
  "appModel": "1.0.0",
  "imports": [
    "github.com/project-flogo/contrib/activity/log",
    "github.com/project-flogo/contrib/trigger/rest",
    "github.com/project-flogo/flow"
  ],
  "triggers": [
    {
      "id": "receive_http_message",
      "type": "rest",
      "name": "Receive HTTP Message",
      "description": "Simple REST Trigger",
      "settings": {
        "port": 9233
      },
      "handlers": [
        {
          "action": {
            "type": "flow",
            "settings": {
              "flowURI": "res://flow:sample_flow"
            }
          },
          "settings": {
            "method": "GET",
            "path": "/test"
          }
        }
      ]
    }
  ],
  "resources": [
    {
      "id": "flow:sample_flow",
      "data": {
        "name": "SampleFlow",
        "tasks": [
          {
            "id": "log_1",
            "name": "Log Message",
            "description": "Simple Log Activity",
            "activity": {
              "type": "log",
              "input": {
                "message": "first log",
                "flowInfo": "false",
                "addToFlow": "false"
              }
            }
          },
          {
            "id": "log_2",
            "name": "Log in a loop",
            "description": "Simple Log Activity",
            "activity": {
              "type": "log",
              "input": {
                "message": "last log",
                "flowInfo": "false",
                "addToFlow": "false"
              }
            }
          }
        ],
        "links": [
          {
            "from": "log_1",
            "to": "log_2"
          }
        ]
      }
    }
  ]
}`

func Benchmark(b *testing.B) {
	startFlogoEngine(flogoJSON)

	benchmarks := []struct {
		name     string
		endpoint string
	}{
		{"simple", "http://localhost:9233/test"},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := http.Get(bm.endpoint)
				if err != nil {
					logger.Error(err)
				}
			}
		})
	}
}

func startFlogoEngine(flogoJSON string) {
	var ready = make(chan bool)

	config := &app.Config{}
	err := json.Unmarshal([]byte(flogoJSON), config)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	go startupEngine(config, ready)
	waitForEngineToStart(ready)
}

func waitForEngineToStart(ready <-chan bool) {
	select {
	case s := <-ready:
		switch s {
		case true:
			logger.RootLogger().Debug("Engine is ready")
		case false:
			logger.RootLogger().Error("Engine did not start")
		}
	}
}

func startupEngine(config *app.Config, ready chan<- bool) int {
	defer func() {
		ready <- false
	}()

	e, err := engine.New(config)
	if err != nil {
		logger.RootLogger().Errorf("Failed to create engine instance due to error: %s", err.Error())
		return 1
	}

	err = e.Start()
	if err != nil {
		logger.RootLogger().Errorf("Failed to start engine due to error: %s", err.Error())
		return 1
	}

	ready <- true

	exitChan := setupSignalHandling()
	code := <-exitChan

	e.Stop()

	return code
}

func setupSignalHandling() chan int {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	exitChan := make(chan int, 1)
	select {
	case s := <-signalChan:
		switch s {
		case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
			exitChan <- 0
		default:
			logger.RootLogger().Debug("Unknown signal.")
			exitChan <- 1
		}
	}
	return exitChan
}
