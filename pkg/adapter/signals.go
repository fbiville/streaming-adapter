package adapter

import "riff-streaming-adapter/streaming"

func NewStartSignal(header string) *streaming.Signal {
	return &streaming.Signal{
		Value: &streaming.Signal_Start{
			Start: &streaming.Start{Accept: header},
		},
	}
}

func NewNextSignal(headers map[string]string, body []byte) *streaming.Signal {
	return &streaming.Signal{
		Value: &streaming.Signal_Next{
			Next: &streaming.Next{
				Headers: headers,
				Payload: body,
			},
		},
	}
}
