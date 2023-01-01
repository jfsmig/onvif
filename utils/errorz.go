package utils

type constError string

func (e constError) Error() string { return string(e) }

var ErrUnreachable = constError("unreachable device")
var ErrHttp = constError("http request error")
var ErrNotOnvif = constError("not an OnVif device")
var ErrUnsupportedPTZ = constError("unsupported PTZ")
var ErrUnsupportedCall = constError("unsupported call")
