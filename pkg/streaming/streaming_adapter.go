package streaming

import (
	"bytes"
	"context"
	"fmt"
	"google.golang.org/grpc"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"riff-streaming-adapter/streaming_proto"
	"time"
)

type StreamingAdapter struct {
	timeout time.Duration
	server  http.Server
}

func NewStreamingAdapter(timeout time.Duration) *StreamingAdapter {
	return &StreamingAdapter{
		timeout: timeout,
	}
}

func (adapter *StreamingAdapter) Start(httpPort int, grpcConnection *grpc.ClientConn) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", httpPort))
	if err != nil {
		return err
	}
	adapter.server = http.Server{Handler: &AdapterHttpHandler{
		grpcClient: streaming_proto.NewRiffClient(grpcConnection),
		timeout:    adapter.timeout,
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
	grpcClient streaming_proto.RiffClient
	timeout    time.Duration
}

func (handler *AdapterHttpHandler) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	start, next, _ := convertRequest(request)
	client, err := handler.grpcClient.Invoke(context.Background())
	if err != nil {
		_ = writeError(responseWriter, 502, "unreachable gRPC server")
		return
	}
	serverResponse := make(chan *streaming_proto.Signal, 1)
	serverErrors := make(chan error, 1)
	go func() {
		_ = client.Send(start)
		_ = client.Send(next)
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
		_ = writeResponse(responseWriter, convertResponse(signal.GetNext()))
	}
}

func convertRequest(request *http.Request) (*streaming_proto.Signal, *streaming_proto.Signal, error) {
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

func convertResponse(next *streaming_proto.Next) *http.Response {
	result := http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Body:       ioutil.NopCloser(bytes.NewReader(next.Payload)),
		Header:     copyResponseHeaders(next.Headers),
	}
	for key, value := range next.Headers {
		result.Header.Add(key, value)
	}
	return &result
}

func copyResponseHeaders(headers map[string]string) http.Header {
	result := make(map[string][]string)
	for key, value := range headers {
		result[key] = []string{value}
	}
	return result
}

func writeResponse(responseWriter http.ResponseWriter, response *http.Response) error {
	responseWriter.WriteHeader(response.StatusCode)
	if err := response.Header.Write(responseWriter); err != nil {
		return err
	}
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if _, err = responseWriter.Write(responseBody); err != nil {
		return err
	}
	return nil
}
