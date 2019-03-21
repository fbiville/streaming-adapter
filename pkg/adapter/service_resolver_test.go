package adapter_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/http"
	"riff-streaming-adapter/pkg/adapter"
)

var _ = Describe("Service resolver", func() {

	It("resolves Knative service names", func() {
		resolver := &adapter.KnativeServiceResolver{}

		connection, err := resolver.Resolve(&http.Request{Header: http.Header{"X-Riff": {"square/default"}}})
		defer func() {
			Expect(connection.Close()).NotTo(HaveOccurred())
		}()

		Expect(err).NotTo(HaveOccurred())
		Expect(connection).NotTo(BeNil())
	})

	It("fails to resolve unstructured names", func() {
		resolver := &adapter.KnativeServiceResolver{}

		_, err := resolver.Resolve(&http.Request{Header: http.Header{"X-Riff": {"not-gonna-work"}}})

		Expect(err).To(MatchError("\"not-gonna-work\" is invalid: expected name to follow SERVICE_NAME/NAMESPACE structure"))
	})

	It("fails when header is missing", func() {
		resolver := &adapter.KnativeServiceResolver{}

		_, err := resolver.Resolve(&http.Request{Header: http.Header{}})

		Expect(err).To(MatchError("\"X-Riff\" header is missing"))
	})
})
