package popover

type Model struct {
	Providers []Provider `json:"providers"`
	Commands  []Command  `json:"commands"`
	Settings  string     `json:"settings"`
	Refresh   Command    `json:"refresh"`
	Quit      Command    `json:"quit"`
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

type Command struct {
	Title   string `json:"title"`
	Action  string `json:"action"`
	Checked bool   `json:"checked"`
}
