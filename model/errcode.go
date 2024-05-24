package model

type ErrorCode int8

const (
	OK ErrorCode = iota
	CE
	IE
	RE
)
