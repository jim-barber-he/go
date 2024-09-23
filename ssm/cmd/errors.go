package cmd

import (
	"errors"

	"github.com/jim-barber-he/go/util"
)

var (
	errGetSSMParameter   = errors.New("failed to get SSM parameter")
	errPutSSMParameter   = errors.New("failed to put SSM parameter")
	errListSSMParameters = errors.New("failed to list SSM parameters")
	errReadFile          = errors.New("failed to read file")
	errValueRequired     = errors.New("VALUE is required when --file is not used")
	errValueWithFile     = errors.New("VALUE should not be provided when --file is used")
)

// NewBriefAndFullError creates a new error for when the --brief and --full options are both specified.
func NewBriefAndFullError(usage string) error {
	return &util.Error{
		Msg:   "it does not make sense to specify both --brief and --full\n",
		Param: usage,
	}
}
