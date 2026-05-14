package app

import (
	"net/http"
	"time"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
)

const (
	fieldConcurrency = iota
	fieldRepeat
	globalFieldCount
)

const (
	fieldName = globalFieldCount + iota
	fieldMethod
	fieldURL
	fieldHeaders
	fieldPayload
	requestFieldCount = fieldPayload - fieldName + 1
)

const sessionDBFile = "sessions.db"

const maxTextareaContentHeight = 10000

type modalMode int

const (
	modalNone modalMode = iota
	modalSaveSession
	modalLoadSession
)

type RequestSpec struct {
	Name      string
	Method    string
	URL       string
	Headers   map[string]string
	Body      []byte
	SourceIdx int
	RunIndex  int
	RunTotal  int
}

type RequestForm struct {
	NameInput   textinput.Model
	MethodInput textinput.Model
	URLInput    textinput.Model
	HeadersArea textarea.Model
	PayloadArea textarea.Model
}

type Result struct {
	StatusCode int
	Duration   time.Duration
	Body       string
	Err        error
	Queued     bool
	Running    bool
	Done       bool
	Name       string
	Method     string
	SourceIdx  int
	RunIndex   int
	RunTotal   int
}

type persistRequest struct {
	Name    string `json:"name"`
	Method  string `json:"method"`
	URL     string `json:"url"`
	Headers string `json:"headers"`
	Payload string `json:"payload"`
}

type allResultsMsg struct {
	Results []Result
}

type runStartMsg struct {
	events chan runEventMsg
}

type runEventMsg interface {
	isRunEventMsg()
}

type resultRunningMsg struct {
	Index int
}

type resultDoneMsg struct {
	Index  int
	Result Result
}

type runCompletedMsg struct{}

func (resultRunningMsg) isRunEventMsg() {}
func (resultDoneMsg) isRunEventMsg()    {}
func (runCompletedMsg) isRunEventMsg()  {}

type SavedSession struct {
	ID           int64
	Name         string
	UpdatedAt    string
	RequestCount int
}

type model struct {
	ConcurrencyInput textinput.Model
	RepeatInput      textinput.Model
	SaveNameInput    textinput.Model
	Forms            []RequestForm
	Results          []Result
	SelectedResult   int
	Viewport         viewport.Model
	SavedSessions    []SavedSession
	SelectedSession  int
	CurrentSession   string
	MouseModeEnabled bool
	ModalMode        modalMode
	FocusIndex       int
	ActiveReq        int
	Running          bool
	RunEvents        chan runEventMsg
	StatusMessage    string
	Width            int
	Height           int
	Client           *http.Client
}
