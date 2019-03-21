package adapter

import (
	"fmt"
	"google.golang.org/grpc"
	"net/http"
	"strings"
)

type ServiceResolver interface {
	Resolve(request *http.Request) (*grpc.ClientConn, error)
}

type KnativeServiceResolver struct{}

func (*KnativeServiceResolver) Resolve(request *http.Request) (*grpc.ClientConn, error) {
	name := request.Header.Get("X-Riff")
	if name == "" {
		return nil, fmt.Errorf("%q header is missing", "X-Riff")
	}
	coordinates := strings.SplitN(name, "/", 2)
	if len(coordinates) != 2 {
		return nil, fmt.Errorf("%q is invalid: expected name to follow SERVICE_NAME/NAMESPACE structure", name)
	}
	host := fmt.Sprintf("%s.%s.svc.cluster.local", coordinates[0], coordinates[1])

	return grpc.Dial(host, grpc.WithInsecure(), grpc.WithAuthority(request.Header.Get("X-Riff-Authority")))
}

type PassthroughResolver struct{}

func (*PassthroughResolver) Resolve(request *http.Request) (*grpc.ClientConn, error) {
	host := request.Header.Get("X-Riff")
	if host == "" {
		return nil, fmt.Errorf("%q header is missing", "X-Riff")
	}
	return grpc.Dial(host, grpc.WithInsecure(), grpc.WithAuthority(request.Header.Get("X-Riff-Authority")))
}
