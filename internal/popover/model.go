package popover

type Model struct {
	Providers     []Provider `json:"providers"`
	Commands      []Command  `json:"commands"`
	Settings      string     `json:"settings"`
	SettingsTitle string     `json:"settingsTitle"`
	GeneralTitle  string     `json:"generalTitle"`
	SourcesTitle  string     `json:"sourcesTitle"`
	LanguageLabel string     `json:"languageLabel"`
	ChangeSource  string     `json:"changeSource"`
	Refresh       Command    `json:"refresh"`
	Quit          Command    `json:"quit"`
}

type Provider struct {
	Name    string   `json:"name"`
	Message string   `json:"message"`
	Windows []Window `json:"windows"`
}

type Window struct {
	Label       string `json:"label"`
	UsedPercent int    `json:"usedPercent"`
	Usage       string `json:"usage"`
	Remaining   string `json:"remaining"`
	Reset       string `json:"reset"`
	Selected    bool   `json:"selected"`
	Action      string `json:"action"`
	Tooltip     string `json:"tooltip"`
}

type CommandKind string

const (
	ToggleCommand   CommandKind = "toggle"
	SourceCommand   CommandKind = "source"
	LanguageCommand CommandKind = "language"
)

type Command struct {
	Kind    CommandKind `json:"kind"`
	Label   string      `json:"label"`
	Detail  string      `json:"detail"`
	Title   string      `json:"title"`
	Action  string      `json:"action"`
	Checked bool        `json:"checked"`
}
