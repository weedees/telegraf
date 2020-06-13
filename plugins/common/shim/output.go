package shim

import (
	"bufio"
	"fmt"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/parsers"
)

// AddOutput adds the input to the shim. Later calls to Run() will run this.
func (s *Shim) AddOutput(output telegraf.Output) error {
	if p, ok := output.(telegraf.Initializer); ok {
		err := p.Init()
		if err != nil {
			return fmt.Errorf("failed to init input: %s", err)
		}
	}

	s.Output = output
	return nil
}

func (s *Shim) RunOutput() error {
	// TODO: ? Need to support multiple parsers, but not clear how.
	parser, err := parsers.NewInfluxParser()
	if err != nil {
		return fmt.Errorf("Failed to create new parser: %w", err)
	}

	err = s.Output.Connect()
	if err != nil {
		return fmt.Errorf("failed to start processor: %w", err)
	}
	defer s.Output.Close()

	var m telegraf.Metric

	scanner := bufio.NewScanner(stdin)
	for scanner.Scan() {
		m, err = parser.ParseLine(scanner.Text())
		if err != nil {
			fmt.Fprintf(stderr, "Failed to parse metric: %s\b", err)
		}
		if err = s.Output.Write([]telegraf.Metric{m}); err != nil {
			fmt.Fprintf(stderr, "Failed to write metric: %s\b", err)
		}
	}

	return nil
}
