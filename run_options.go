package go_monobank

import (
	"encoding/json"
	"fmt"

	"github.com/stremovskyy/go-monobank/log"
)

// RunOption controls behavior of a single API call.
type RunOption func(*runOptions)

// DryRunHandler receives info about a skipped request.
type DryRunHandler func(endpoint string, payload any)

type runOptions struct {
	dryRun       bool
	dryRunHandle DryRunHandler
}

var dryRunLogger = log.NewLogger("Monobank DryRun:")

// DryRun skips the underlying HTTP call.
//
// Optional handler can be provided to inspect payload.
func DryRun(handler ...DryRunHandler) RunOption {
	return func(o *runOptions) {
		o.dryRun = true
		if len(handler) > 0 && handler[0] != nil {
			o.dryRunHandle = handler[0]
			return
		}
		o.dryRunHandle = defaultDryRunHandler
	}
}

func collectRunOptions(opts []RunOption) *runOptions {
	if len(opts) == 0 {
		return &runOptions{}
	}
	r := &runOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(r)
		}
	}
	return r
}

func (o *runOptions) isDryRun() bool {
	return o != nil && o.dryRun
}

func (o *runOptions) handleDryRun(endpoint string, payload any) {
	if o == nil || !o.dryRun {
		return
	}
	if o.dryRunHandle != nil {
		o.dryRunHandle(endpoint, payload)
	}
}

func defaultDryRunHandler(endpoint string, payload any) {
	dryRunLogger.Info("Dry run: skipping request to %s", endpoint)
	if payload == nil {
		dryRunLogger.Info("Dry run payload: <nil>")
		return
	}
	out, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		dryRunLogger.Info("Dry run payload: unable to marshal %T: %v", payload, err)
		return
	}
	dryRunLogger.Info("Dry run payload:\n%s", fmt.Sprintf("%s", out))
}
