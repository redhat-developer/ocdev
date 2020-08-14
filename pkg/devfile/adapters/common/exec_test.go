package common

import (
	"reflect"
	"testing"
)

func TestCreateConsoleOutputWriterAndChannel(t *testing.T) {

	tests := []struct {
		Name  string
		Input []string
	}{
		{
			Name:  "Close channel with no text sent",
			Input: []string{},
		},
		{
			Name:  "Close channel with a single line of text sent",
			Input: []string{"hi"},
		},
		{
			Name:  "Close channel with multiple lines of text sent",
			Input: []string{"hi"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {

			inputWriter, outputChan := CreateConsoleOutputWriterAndChannel()

			// Write input text
			for _, toSend := range tt.Input {
				inputWriter.Write([]byte(toSend + "\n"))
			}

			// Close and wait for result
			inputWriter.Close()
			out := <-outputChan

			// Ouput text read from channel should exactly match input text
			if len(out) != len(tt.Input) {
				t.Fatal("Output response did not match input", out)
			}
			if !reflect.DeepEqual(tt.Input, out) {
				t.Fatal("Output response did not match input", out)
			}

		})
	}

}
