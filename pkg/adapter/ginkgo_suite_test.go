package adapter_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestStreamingAdapter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Streaming Adapter Suite")
}

