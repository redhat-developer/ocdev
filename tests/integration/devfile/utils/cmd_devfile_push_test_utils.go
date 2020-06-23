package utils

import (
	"encoding/json"
	"strings"

	"github.com/openshift/odo/pkg/machineoutput"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// AnalyzePushConsoleOutput analyzes the output of 'odo push -o json' for the machine readable event push test above.
func AnalyzePushConsoleOutput(pushConsoleOutput string) {

	lines := strings.Split(strings.Replace(pushConsoleOutput, "\r\n", "\n", -1), "\n")

	var entries []machineoutput.MachineEventLogEntry

	// Ensure that all lines can be correctly parsed into their expected JSON structure
	for _, line := range lines {

		if len(strings.TrimSpace(line)) == 0 {
			continue
		}

		// fmt.Println("Processing output line: " + line)

		lineWrapper := machineoutput.MachineEventWrapper{}

		err := json.Unmarshal([]byte(line), &lineWrapper)
		Expect(err).NotTo(HaveOccurred())

		entry, err := lineWrapper.GetEntry()
		Expect(err).NotTo(HaveOccurred())

		entries = append(entries, entry)

	}

	// Ensure we pass a sanity test on the minimum expected entries
	if len(entries) < 4 {
		Fail("Expected at least 4 entries, corresponding to command/action execution.")
	}

	// Ensure that all logText entries are wrapped inside commandExecutionBegin and commandExecutionComplete entries (e.g. no floating logTexts)
	insideCommandExecution := false
	for _, entry := range entries {

		if entry.GetType() == machineoutput.TypeDevFileCommandExecutionBegin {
			insideCommandExecution = true
		}

		if entry.GetType() == machineoutput.TypeDevFileCommandExecutionComplete {
			insideCommandExecution = false
		}

		if entry.GetType() == machineoutput.TypeLogText {
			Expect(insideCommandExecution).To(Equal(true))
		}

	}

	// Ensure that the log output has the given structure:
	// - look for the expected devbuild events, then look for the expected devrun events.
	expectedEventOrder := []struct {
		entryType   machineoutput.MachineEventLogEntryType
		commandName string
	}{
		// first the devbuild command (and its action) should run
		{
			machineoutput.TypeDevFileCommandExecutionBegin,
			"devbuild",
		},
		{
			// at least one logged line of text
			machineoutput.TypeLogText,
			"",
		},
		{
			machineoutput.TypeDevFileCommandExecutionComplete,
			"devbuild",
		},
		// next the devbuild command (and its action) should run
		{
			machineoutput.TypeDevFileCommandExecutionBegin,
			"devrun",
		},
		{
			// at least one logged line of text
			machineoutput.TypeLogText,
			"",
		},
		{
			machineoutput.TypeDevFileCommandExecutionComplete,
			"devrun",
		},
	}
	currIndex := -1
	for _, nextEventOrder := range expectedEventOrder {
		entry, newIndex := findNextEntryByType(currIndex, nextEventOrder.entryType, entries)
		Expect(entry).NotTo(BeNil())
		Expect(newIndex).To(BeNumerically(">=", 0))
		Expect(newIndex).To(BeNumerically(">", currIndex)) // monotonically increasing index

		// We should see devbuild for the first set of events, then devrun
		commandName := machineoutput.GetCommandName(entry)
		Expect(commandName).To(Equal(nextEventOrder.commandName))

		currIndex = newIndex
	}

}

// findNextEntryByType locates the next entry of a given type within a slice. Currently used for test purposes only.
func findNextEntryByType(initialIndex int, typeToFind machineoutput.MachineEventLogEntryType, entries []machineoutput.MachineEventLogEntry) (machineoutput.MachineEventLogEntry, int) {

	for index, entry := range entries {
		if index < initialIndex {
			continue
		}

		if entry.GetType() == typeToFind {
			return entry, index
		}
	}

	return nil, -1

}
