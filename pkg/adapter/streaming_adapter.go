package adapter

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"riff-streaming-adapter/streaming"
	"time"
)

// visible for tests
type StreamingAdapter struct {
	ServiceResolver ServiceResolver
	Timeout         time.Duration
	server          http.Server
}

func NewStreamingAdapter(timeout time.Duration) *StreamingAdapter {
	return &StreamingAdapter{
		ServiceResolver: &PassthroughResolver{},
		Timeout:         timeout,
	}
}

func (adapter *StreamingAdapter) Start(httpPort int) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", httpPort))
	if err != nil {
		return err
	}
	adapter.server = http.Server{Handler: &AdapterHttpHandler{
		ServiceResolver: adapter.ServiceResolver,
		timeout:         adapter.Timeout,
	}}
	go func() {
		if err = adapter.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			_, _ = fmt.Fprintf(os.Stderr, "error when starting server: %v", err)
		}
	}()
	return nil
}

func (adapter *StreamingAdapter) Close() error {
	if err := adapter.server.Close(); err != nil {
		return err
	}
	return nil
}

// Implementation of http.Handler that also acts as a gRPC client
type AdapterHttpHandler struct {
	ServiceResolver ServiceResolver
	timeout         time.Duration
}

func (handler *AdapterHttpHandler) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	start, next, _ := convertRequest(request)
	connection, err := handler.ServiceResolver.Resolve(request)
	if err != nil {
		_ = writeError(responseWriter, 502, "unreachable gRPC server")
		return
	}
	defer func() {
		_ = connection.Close()
	}()
	riffClient := streaming.NewRiffClient(connection)
	client, err := riffClient.Invoke(context.Background())
	if err != nil {
		_ = writeError(responseWriter, 502, "unreachable gRPC server")
		return
	}
	serverResponse := make(chan *streaming.Signal, 1)
	serverErrors := make(chan error, 1)
	go func() {
		_ = client.Send(start)
		_ = client.Send(next)
		_ = client.CloseSend()
		signal, err := client.Recv()
		if err != nil {
			serverErrors <- err
		} else {
			serverResponse <- signal
		}
	}()
	select {
	case <-time.After(handler.timeout):
		_ = writeError(responseWriter, 504, "upstream gRPC server did not respond in time")
	case err = <-serverErrors:
		_ = writeError(responseWriter, 502, "misbehaving gRPC server")
	case signal := <-serverResponse:
		_ = writeResponse(responseWriter, signal.GetNext())
	}
}

func convertRequest(request *http.Request) (*streaming.Signal, *streaming.Signal, error) {
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		return nil, nil, err
	}
	err = request.Body.Close()
	if err != nil {
		return nil, nil, err
	}
	start := NewStartSignal(request.Header.Get("Accept"))
	next := NewNextSignal(copyRequestHeaders(request.Header, "Accept"), body)
	return start, next, nil
}

func copyRequestHeaders(headers http.Header, excludedHeader string) map[string]string {
	result := make(map[string]string)
	for key, values := range headers {
		if key == excludedHeader {
			continue
		}
		for _, value := range values {
			result[key] = value
		}
	}
	return result
}

func writeError(responseWriter http.ResponseWriter, statusCode int, reason string) error {
	responseWriter.WriteHeader(statusCode)
	responseWriter.Header().Add("Content-Type", "text/plain")
	if _, err := responseWriter.Write([]byte(reason)); err != nil {
		return err
	}
	return nil
}

func writeResponse(responseWriter http.ResponseWriter, next *streaming.Next) error {
	responseWriter.WriteHeader(200)
	for key, value := range next.Headers {
		if key == "Content-Length" {
			continue
		}
		responseWriter.Header().Add(key, value)
	}
	_, err := responseWriter.Write(next.Payload)
	return err
}
