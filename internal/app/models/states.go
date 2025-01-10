package models

type State int

const (
	Default State = iota
	StateTextSupport
)

var userStates = make(map[int64]State)
