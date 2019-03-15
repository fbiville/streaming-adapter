package main

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"riff-streaming-adapter/pkg/adapter"
	"strconv"
	"time"
)

func main() {
	httpPort, err := mandatoryIntEnvVar("HTTP_PORT")
	if err != nil {
		panic(err)
	}
	httpTimeout, err := mandatoryIntEnvVar("HTTP_TIMEOUT_MILLISECONDS")
	if err != nil {
		panic(err)
	}
	timeout := time.Duration(int64(time.Millisecond) * int64(httpTimeout))
	streamingAdapter := adapter.NewStreamingAdapter(timeout)
	err = streamingAdapter.Start(httpPort)
	if err != nil {
		panic(err)
	}
	defer logClose(streamingAdapter)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
}

func mandatoryIntEnvVar(name string) (int, error) {
	result, err := mandatoryStringEnvVar(name)
	if err != nil {
		return -1, err
	}
	return strconv.Atoi(result)
}

func mandatoryStringEnvVar(name string) (string, error) {
	result, found := os.LookupEnv(name)
	if !found {
		return "", fmt.Errorf("envvar %s not found", name)
	}
	return result, nil
}

func logClose(closeable io.Closer) {
	if err := closeable.Close(); err != nil {
		panic(err)
	}
}
