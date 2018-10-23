package opentracing

import (
	"encoding/json"
	"fmt"
	"github.com/TIBCOSoftware/flogo-lib/app"
	"github.com/TIBCOSoftware/flogo-lib/engine"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"testing"

	_ "github.com/TIBCOSoftware/flogo-contrib/action/flow"
	_ "github.com/TIBCOSoftware/flogo-contrib/activity/log"
	_ "github.com/TIBCOSoftware/flogo-contrib/trigger/rest"
	"github.com/TIBCOSoftware/flogo-lib/logger"
	_ "github.com/apache/thrift/lib/go/thrift"
	_ "github.com/square-it/flogo-contrib-activities/sleep"
)

const flogoJSON string = `{
  "name": "flogo-opentracing-sample",
  "type": "flogo:app",
  "version": "0.0.1",
  "appModel": "1.0.0",
  "triggers": [
    {
      "id": "receive_http_message",
      "ref": "github.com/TIBCOSoftware/flogo-contrib/trigger/rest",
      "name": "Receive HTTP Message",
      "description": "Simple REST Trigger",
      "settings": {
        "port": 9233
      },
      "handlers": [
        {
          "action": {
            "ref": "github.com/TIBCOSoftware/flogo-contrib/action/flow",
            "data": {
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
              "ref": "github.com/TIBCOSoftware/flogo-contrib/activity/log",
              "input": {
                "message": "first log",
                "flowInfo": "false",
                "addToFlow": "false"
              }
            }
          },
          {
            "id": "sleep_1",
            "name": "Sleep",
            "description": "Sleep Activity",
            "activity": {
              "ref": "github.com/debovema/flogo-contrib-activities/sleep",
              "input": {
                "duration": "5s"
              }
            }
          },
          {
            "id": "log_2",
            "type": "iterator",
            "name": "Log in a loop",
            "description": "Simple Log Activity",
            "settings": {
              "iterate": "2"
            },
            "activity": {
              "ref": "github.com/TIBCOSoftware/flogo-contrib/activity/log",
              "input": {
                "flowInfo": "false",
                "addToFlow": "false"
              },
              "mappings": {
                "input": [
                  {
                    "type": "assign",
                    "value": "$current.iteration.value",
                    "mapTo": "message"
                  }
                ]
              }
            }
          }
        ],
        "links": [
          {
            "from": "log_1",
            "to": "sleep_1"
          },
          {
            "from": "sleep_1",
            "to": "log_2"
          }
        ]
      }
    }
  ]
}`

func Benchmark(b *testing.B) {
	startFlogoEngine()

	benchmarks := []struct {
		name     string
		endpoint string
	}{
		{"simple", "http://localhost:9233/test"},
		{"error", "http://localhost:9233/notfound"},
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

func startFlogoEngine() {
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
			logger.Debug("Engine is ready")
		}
	}
}

func startupEngine(config *app.Config, ready chan<- bool) int {
	defer func() {
		ready <- false
	}()

	e, err := engine.New(config)
	if err != nil {
		log.Errorf("Failed to create engine instance due to error: %s", err.Error())
		return 1
	}

	err = e.Start()
	if err != nil {
		log.Errorf("Failed to start engine due to error: %s", err.Error())
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
			logger.Debug("Unknown signal.")
			exitChan <- 1
		}
	}
	return exitChan
}
