package main

import (
	"fmt"
	"io"
	"os"
	"riff-streaming-adapter/pkg/adapter"
	"strconv"
	"time"
)

func main() {
	//httpPort, err := mandatoryIntEnvVar("HTTP_PORT")
	//if err != nil {
	//	panic(err)
	//}
	//httpTimeout, err := mandatoryIntEnvVar("HTTP_TIMEOUT_MILLISECONDS")
	//if err != nil {
	//	panic(err)
	//}
	httpTimeout := 5 * time.Minute
	httpPort := 8888
	timeout := time.Duration(int64(time.Millisecond) * int64(httpTimeout))
	adapter := adapter.NewStreamingAdapter(timeout)
	err := adapter.Start(httpPort)
	if err != nil {
		panic(err)
	}
	defer logClose(adapter)
	<-make(chan struct{})
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
