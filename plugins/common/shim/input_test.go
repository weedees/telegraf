package shim

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/influxdata/telegraf"
)

func TestInputShimTimer(t *testing.T) {
	stdoutBytes := bytes.NewBufferString("")
	stdout = stdoutBytes

	stdin, _ = io.Pipe() // hold the stdin pipe open

	metricProcessed, _ := runInputPlugin(t, 10*time.Millisecond)

	<-metricProcessed
	for stdoutBytes.Len() == 0 {
		time.Sleep(10 * time.Millisecond)
	}

	out := string(stdoutBytes.Bytes())
	require.Contains(t, out, "\n")
	metricLine := strings.Split(out, "\n")[0]
	require.Equal(t, "measurement,tag=tag field=1i 1234000005678", metricLine)
}

func TestInputShimStdinSignalingWorks(t *testing.T) {
	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()

	stdin = stdinReader
	stdout = stdoutWriter

	metricProcessed, exited := runInputPlugin(t, 40*time.Second)

	stdinWriter.Write([]byte("\n"))

	<-metricProcessed

	r := bufio.NewReader(stdoutReader)
	out, err := r.ReadString('\n')
	require.NoError(t, err)
	require.Equal(t, "measurement,tag=tag field=1i 1234000005678\n", out)

	stdinWriter.Close()
	go ioutil.ReadAll(r)
	// check that it exits cleanly
	<-exited
}

func runInputPlugin(t *testing.T, interval time.Duration) (metricProcessed chan bool, exited chan bool) {
	metricProcessed = make(chan bool, 1)
	exited = make(chan bool, 1)
	inp := &testInput{
		metricProcessed: metricProcessed,
	}

	shim := New()
	shim.AddInput(inp)
	go func() {
		err := shim.Run(interval)
		require.NoError(t, err)
		exited <- true
	}()
	return metricProcessed, exited
}

type testInput struct {
	metricProcessed chan bool
}

func (i *testInput) SampleConfig() string {
	return ""
}

func (i *testInput) Description() string {
	return ""
}

func (i *testInput) Gather(acc telegraf.Accumulator) error {
	acc.AddFields("measurement",
		map[string]interface{}{
			"field": 1,
		},
		map[string]string{
			"tag": "tag",
		}, time.Unix(1234, 5678))
	i.metricProcessed <- true
	return nil
}

func (i *testInput) Start(acc telegraf.Accumulator) error {
	return nil
}

func (i *testInput) Stop() {
}

type serviceInput struct {
	ServiceName string `toml:"service_name"`
	SecretToken string `toml:"secret_token"`
	SecretValue string `toml:"secret_value"`
}

func (i *serviceInput) SampleConfig() string {
	return ""
}

func (i *serviceInput) Description() string {
	return ""
}

func (i *serviceInput) Gather(acc telegraf.Accumulator) error {
	acc.AddFields("measurement",
		map[string]interface{}{
			"field": 1,
		},
		map[string]string{
			"tag": "tag",
		}, time.Unix(1234, 5678))

	return nil
}

func (i *serviceInput) Start(acc telegraf.Accumulator) error {
	return nil
}

func (i *serviceInput) Stop() {
}
