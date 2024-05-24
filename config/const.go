package config

import "errors"

var ErrTLE = errors.New("time limit exceeded")
var ErrOOM = errors.New("out of memory")
var ErrFile = errors.New("file operation with no permission")

const FolderNameLen = 20
const OmitStringLen = int64(4096)

var DefaultEnv = []string{"PATH=/bin:/usr/bin"}
