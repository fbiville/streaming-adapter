package streaming

import "riff-streaming-adapter/streaming_proto"

func NewStartSignal(header string) *streaming_proto.Signal {
	return &streaming_proto.Signal{
		Value: &streaming_proto.Signal_Start{
			Start: &streaming_proto.Start{Accept: header},
		},
	}
}

func NewNextSignal(headers map[string]string, body []byte) *streaming_proto.Signal {
	return &streaming_proto.Signal{
		Value: &streaming_proto.Signal_Next{
			Next: &streaming_proto.Next{
				Headers: headers,
				Payload: body,
			},
		},
	}
}

