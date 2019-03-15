package streaming_test // visible for tests only

import (
	"fmt"
	"io"
	"riff-streaming-adapter/streaming_proto"
)

// gRPC server that translates (some) digits into French according to the requested representation (text or JSON).
// It makes sure that only a sequence of exactly 1 START and 1 NEXT signal is received for every invocation
// It also allows for errors to be injected based on the count of Invoke invocations of this instance.
type frenchizerServer struct {
	invokeCount int
	InjectError func(int) error
}

func ErroringFrenchizerServer(injector func(int) error) *frenchizerServer {
	return &frenchizerServer{InjectError: injector}
}

func NewFrenchizerServer() *frenchizerServer {
	return &frenchizerServer{InjectError: func(i int) error {
		return nil
	}}
}

func (frenchizer *frenchizerServer) Invoke(server streaming_proto.Riff_InvokeServer) error {
	var receivedSignals = map[string]bool{"start": false, "next": false}
	var accept *string

	for true {
		signal, err := server.Recv()
		if err == io.EOF {
			return nil
		}

		if value := signal.GetStart(); value != nil {
			if receivedSignals["start"] {
				return fmt.Errorf("start must only be sent once by the client")
			}
			frenchizer.invokeCount++
			if err := frenchizer.InjectError(frenchizer.invokeCount); err != nil {
				return err
			}
			receivedSignals["start"] = true
			accept = &value.Accept
		}
		if value := signal.GetNext(); value != nil {
			if receivedSignals["next"] {
				return fmt.Errorf("next must only be sent once by the client")
			}
			receivedSignals["next"] = true
			payload, err := frenchizer.onNext(accept, value)
			if err != nil {
				return err
			}
			err = server.Send(nextSignal(payload))
			if err != nil {
				return err
			}
			return nil
		}
		if !receivedSignals["start"] && !receivedSignals["next"] {
			return fmt.Errorf("unsupported signal value %v", signal.GetValue())
		}
	}
	return nil
}

func (frenchizer *frenchizerServer) onNext(accept *string, value *streaming_proto.Next) (string, error) {
	if accept == nil {
		return "", fmt.Errorf("start must be sent before next")
	}
	payload := string(value.Payload)
	result, err := frenchizer.makeMagicHappen(*accept, payload)
	if err != nil {
		return "", err
	}
	return result, nil
}

func (frenchizer *frenchizerServer) makeMagicHappen(acceptHeader string, requestPayload string) (string, error) {
	var value string
	switch requestPayload {
	case "1":
		value = "un"
	case "2":
		value = "deux"
	case "3":
		value = "trois"
	default:
		return "", fmt.Errorf("unexpected request payload %v", requestPayload)
	}

	switch acceptHeader {
	case "application/json":
		return fmt.Sprintf(`"%s"`, value), nil
	case "text/plain":
		return value, nil
	default:
		return "", fmt.Errorf("unsupported accept header %v", acceptHeader)
	}
}

func nextSignal(payload string) *streaming_proto.Signal {
	value := streaming_proto.Signal_Next{Next: &streaming_proto.Next{Payload: []byte(payload)}}
	return &streaming_proto.Signal{Value: &value}
}
