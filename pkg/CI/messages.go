package CI

const (
	Log    = "Log"
	Status = "Status"
	Build  = "Build"
)

type Message struct {
	Kind  string `json:"Kind"`
	Build int    `json:"Build"`
}

type LogsMessage struct {
	*Message
	Logs string `json:"Logs"`
}

type StatusMessage struct {
	*Message
	Success bool `json:"Success"`
}

type BuildMessage struct {
	*Message
}

type PRRequestParameter map[string]string

func NewPRRequestParameter(name, value string) PRRequestParameter {
	p := make(PRRequestParameter)
	p["name"] = name
	p["value"] = value
	return p
}

type PRRequestMessage struct {
	Project    string               `json:"project"`
	Token      string               `json:"token"`
	Parameters []PRRequestParameter `json:"parameter"`
}

func NewPRRequestMessage(project, prno, token string) *PRRequestMessage {
	prm := &PRRequestMessage{
		Project: project,
		Token:   token,
	}
	prp := NewPRRequestParameter("PR_NO", prno)
	prm.Parameters = append(prm.Parameters, prp)
	return prm
}

func NewBuildMessage(build int) *BuildMessage {
	return &BuildMessage{
		Message: &Message{
			Kind:  Build,
			Build: build,
		},
	}
}

func NewLogsMessage(build int) *LogsMessage {
	return &LogsMessage{
		Message: &Message{
			Kind:  Log,
			Build: build,
		},
	}
}

func NewStatusMessage(build int) *StatusMessage {
	return &StatusMessage{
		Message: &Message{
			Kind:  Status,
			Build: build,
		},
	}
}

func (m *Message) IsLog() bool {
	return m.Kind == Log
}

func (m *Message) IsStatus() bool {
	return m.Kind == Status
}

func (m *Message) IsBuild() bool {
	return m.Kind == Build
}