package cmd

import (
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/jim-barber-he/go/util"
)

var (
	errGetSSMParameter = errors.New("failed to get SSM parameter")
	errInvalidDataType = errors.New(
		"invalid data-type specified. Must be one of: text, aws:ec2:image, or aws:ssm:integration",
	)
	errListSSMParameters = errors.New("failed to list SSM parameters")
	errPutSSMParameter   = errors.New("failed to put SSM parameter")
	errReadFile          = errors.New("failed to read file")
	errValueRequired     = errors.New("VALUE is required when --file is not used")
	errValueWithFile     = errors.New("VALUE should not be provided when --file is used")
)

// newBriefAndFullError creates a new error for when the --brief and --full options are both specified.
func newBriefAndFullError(usage string) error {
	return &util.Error{
		Msg:   "it does not make sense to specify both --brief and --full\n",
		Param: usage,
	}
}

// newBriefAndJSONError creates a new error for when the --brief and --json options are both specified.
func newBriefAndJSONError(usage string) error {
	return &util.Error{
		Msg:   "it does not make sense to specify both --brief and --json\n",
		Param: usage,
	}
}

// newInvalidEnvError creates a new error for when an invalid environment is specified.
func newInvalidEnvError(env string) error {
	return &util.Error{
		Msg:   "invalid environment: ",
		Param: env,
	}
}

// newInvalidTierError creates a new error for when an invalid SSM parameter store tier is specified.
func newInvalidTierError() error {
	ssmTiers := types.ParameterTier("").Values()
	ssmTiersStr := make([]string, len(ssmTiers))

	for i, tier := range ssmTiers {
		ssmTiersStr[i] = string(tier)
	}

	return errors.New("invalid tier specified. Must be one of: " + strings.Join(ssmTiersStr, ", "))
}
