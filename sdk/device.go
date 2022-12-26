package sdk

import (
	"context"
)

type Device struct {
}

func (dw *deviceWrapper) FetchDevice(ctx context.Context) Device {
	out := Device{}

	return out
}
