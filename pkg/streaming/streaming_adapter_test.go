package streaming_test

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"riff-streaming-adapter/pkg/streaming"
	"riff-streaming-adapter/streaming_proto"
	"strings"
	"time"
)

var _ = Describe("Streaming adapter", func() {

	const timeout = 200 * time.Millisecond

	Describe("when given proper args", func() {
		var (
			grpcConnection *grpc.ClientConn
			adapter        *streaming.StreamingAdapter
			adapterAddress string
			httpClient     *http.Client
		)

		BeforeEach(func() {
			grpcConnection = openGrpcConnection(NewFrenchizerServer())
			httpPort := findFreePort()

			adapter = streaming.NewStreamingAdapter(timeout)
			err := adapter.Start(httpPort, grpcConnection)
			Expect(err).NotTo(HaveOccurred())

			adapterAddress = fmt.Sprintf("http://localhost:%d", httpPort)
			httpClient = &http.Client{}
		})

		AfterEach(func() {
			assertClose(adapter)
			assertClose(grpcConnection)
		})

		It("adapts and routes single request to gRPC target", func() {
			response, err := httpClient.Do(post(adapterAddress, map[string]string{"Accept": "application/json"}, "2"))

			Expect(err).NotTo(HaveOccurred())
			Expect(response.StatusCode).To(Equal(200))
			Expect(asString(response.Body)).To(Equal(`"deux"`))
		})

		It("adapts and routes several requests to gRPC target", func() {
			response1, _ := httpClient.Do(post(adapterAddress, map[string]string{"Accept": "application/json"}, "1"))
			response2, _ := httpClient.Do(post(adapterAddress, map[string]string{"Accept": "text/plain"}, "2"))

			Expect(response1.StatusCode).To(Equal(200))
			Expect(asString(response1.Body)).To(Equal(`"un"`))
			Expect(response2.StatusCode).To(Equal(200))
			Expect(asString(response2.Body)).To(Equal("deux"))
		})
	})

	Describe("when given wrong arguments", func() {
		var adapter *streaming.StreamingAdapter

		BeforeEach(func() {
			adapter = streaming.NewStreamingAdapter(timeout)
		})

		It("refuses to start on a busy HTTP port", func() {
			listener, err := net.Listen("tcp", ":0")
			Expect(err).NotTo(HaveOccurred())
			defer assertClose(listener)
			grpcConnection := openGrpcConnection(NewFrenchizerServer())
			busyPort := portOf(listener)

			err = adapter.Start(busyPort, grpcConnection)

			Expect(err).To(MatchError(fmt.Sprintf("listen tcp :%d: bind: address already in use", busyPort)))
		})

		It("returns 5xx errors when the gRPC server is unreachable", func() {
			port := findFreePort()
			adapterAddress := fmt.Sprintf("http://localhost:%d", port)
			adapter = streaming.NewStreamingAdapter(timeout)
			err := adapter.Start(port, koGrpcConnection())
			Expect(err).NotTo(HaveOccurred())
			defer assertClose(adapter)
			httpClient := &http.Client{}

			response, err := httpClient.Do(post(adapterAddress, map[string]string{"Accept": "application/json"}, "1"))

			Expect(err).NotTo(HaveOccurred())
			Expect(response.StatusCode).To(Equal(502))
			Expect(asString(response.Body)).To(Equal("unreachable gRPC server"))
		})

		It("returns 5xx errors when the gRPC server fails to be invoked", func() {
			port := findFreePort()
			adapterAddress := fmt.Sprintf("http://localhost:%d", port)
			adapter = streaming.NewStreamingAdapter(timeout)
			grpcConnection := openGrpcConnection(ErroringFrenchizerServer(func(int) error {
				return fmt.Errorf("nope")
			}))
			defer assertClose(grpcConnection)
			err := adapter.Start(port, grpcConnection)
			Expect(err).NotTo(HaveOccurred())
			defer assertClose(adapter)
			httpClient := &http.Client{}

			response, err := httpClient.Do(post(adapterAddress, map[string]string{"Accept": "application/json"}, "1"))

			Expect(err).NotTo(HaveOccurred())
			Expect(response.StatusCode).To(Equal(502))
			Expect(asString(response.Body)).To(Equal("misbehaving gRPC server"))
		})

		It("returns 5xx errors only for failing invocations", func() {
			port := findFreePort()
			adapterAddress := fmt.Sprintf("http://localhost:%d", port)
			adapter = streaming.NewStreamingAdapter(timeout)
			grpcConnection := openGrpcConnection(ErroringFrenchizerServer(func(invocationCount int) error {
				if invocationCount%2 == 1 {
					return fmt.Errorf("nope")
				}
				return nil
			}))
			defer assertClose(grpcConnection)
			err := adapter.Start(port, grpcConnection)
			Expect(err).NotTo(HaveOccurred())
			defer assertClose(adapter)
			httpClient := &http.Client{}

			response1, _ := httpClient.Do(post(adapterAddress, map[string]string{"Accept": "application/json"}, "1"))
			response2, _ := httpClient.Do(post(adapterAddress, map[string]string{"Accept": "text/plain"}, "3"))

			Expect(response1.StatusCode).To(Equal(502))
			Expect(response2.StatusCode).To(Equal(200))
			Expect(asString(response2.Body)).To(Equal("trois"))
		})

		It("returns 5xx errors for invocations that exceeds the adapter timeout", func() {
			port := findFreePort()
			adapterAddress := fmt.Sprintf("http://localhost:%d", port)
			adapter = streaming.NewStreamingAdapter(200 * time.Millisecond)
			grpcConnection := openGrpcConnection(ErroringFrenchizerServer(func(invocationCount int) error {
				time.Sleep(600 * time.Millisecond)
				return nil
			}))
			defer assertClose(grpcConnection)
			err := adapter.Start(port, grpcConnection)
			Expect(err).NotTo(HaveOccurred())
			defer assertClose(adapter)
			httpClient := &http.Client{}

			response, _ := httpClient.Do(post(adapterAddress, map[string]string{"Accept": "application/json"}, "1"))

			Expect(response.StatusCode).To(Equal(504))
		})
	})
})

func openGrpcConnection(server streaming_proto.RiffServer) *grpc.ClientConn {
	grpcServer := grpc.NewServer()
	streaming_proto.RegisterRiffServer(grpcServer, server)
	listener := makeListener(":0")
	go func() {
		err := grpcServer.Serve(listener)
		Expect(err).To(BeNil(), "gRPC server should start")
	}()

	connection, err := grpc.Dial(listener.Addr().String(), grpc.WithInsecure())
	Expect(err).NotTo(HaveOccurred())
	return connection
}

func koGrpcConnection() *grpc.ClientConn {
	connection, _ := grpc.Dial(fmt.Sprintf("localhost:%d", findFreePort()), grpc.WithInsecure())
	return connection
}

func post(url string, headers map[string]string, body string) *http.Request {
	request, err := http.NewRequest("POST", url, strings.NewReader(body))
	Expect(err).To(BeNil(), "should create HTTP request")
	for key, value := range headers {
		request.Header.Add(key, value)
	}
	return request
}

func asString(body io.ReadCloser) string {
	result, err := ioutil.ReadAll(body)
	Expect(err).NotTo(HaveOccurred())
	return string(result)
}

func findFreePort() int {
	listener := makeListener(":0")
	defer assertClose(listener)
	return portOf(listener)
}

func portOf(listener net.Listener) int {
	return listener.Addr().(*net.TCPAddr).Port
}

func makeListener(port string) net.Listener {
	listener, err := net.Listen("tcp", port)
	if err != nil {
		Expect(err).To(BeNil(), "listener should start")
	}
	return listener
}

func assertClose(closer io.Closer) {
	Expect(closer.Close()).NotTo(HaveOccurred())
}
