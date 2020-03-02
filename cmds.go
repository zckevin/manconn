package manconn

import (
	"context"
	"time"

	"github.com/uber/tchannel-go"
	"github.com/uber/tchannel-go/json"
)

type LatencyConfig struct {
	Sid    string
	Type   string
	Config interface{}
}

type ConnConfig struct {
	Latency LatencyConfig
}

type Response struct {
	Success bool
	Err     string
}

func onError(ctx context.Context, err error) {
	log.Errorf("tchannel's onError handler is triggered: %v.", err)
}

func listenAndHandle(s *tchannel.Channel, hostPort string) {
	if err := s.ListenAndServe(hostPort); err != nil {
		log.Errorf("tchannel error start error: %v.", err)
	}
	log.Infof("tchannel server start at %s", hostPort)
}

func (d *Dispatcher) acceptCmds() {
	ch, err := tchannel.NewChannel("CommandService", &tchannel.ChannelOptions{})
	if err != nil {
		log.Errorf("Could not create new channel: %v.", err)
	}

	json.Register(ch, json.Handlers{
		"setLatency": d.setLatencyHandler,
	}, onError)

	listenAndHandle(ch, "127.0.0.1:18082")
}

func (d *Dispatcher) setLatencyHandler(ctx json.Context, config *LatencyConfig) (*Response, error) {
	log.Debugf("sid: %v, config: %v", config.Sid, config.Config)
	d.mu.Lock()
	defer d.mu.Unlock()
	if st, ok := d.streams[config.Sid]; ok {
		var latency LatencyAdder
		c := config.Config.(map[string]interface{})

		switch config.Type {
		case "DummyLatencyAdder":
			latency = &DummyLatencyAdder{
				Latency: time.Duration(uint64(c["Latency"].(float64))),
			}
		case "TimeRangeLatencyAdder":
			latency = NewTimeRangeLatencyAdder(
				time.Duration(uint64(c["Old"].(float64))),
				time.Duration(uint64(c["Range"].(float64))),
				time.Duration(uint64(c["New"].(float64))),
			)
		default:
			return &Response{
				Success: false,
				Err:     "Could not find type: " + config.Type,
			}, nil
		}
		st.downlink.setLatency(latency)
		st.uplink.setLatency(latency)
		return &Response{
			Success: true,
		}, nil
	} else {
		return &Response{
			Success: false,
			Err:     "Could not find stream " + config.Sid,
		}, nil
	}
}
