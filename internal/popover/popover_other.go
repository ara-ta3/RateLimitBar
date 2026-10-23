//go:build !darwin

package popover

const Available = false

var Events <-chan string

func Update(Model) {}
func Toggle()      {}
func Close()       {}

func ShowSettings() {}

func UpdateStatus(Status) {}
