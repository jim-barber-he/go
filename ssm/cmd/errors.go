package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

var (
	errGetSSMParameter = errors.New("failed to get SSM parameter")
	errInvalidDataType = errors.New(
		"invalid data-type specified. Must be one of: text, aws:ec2:image, or aws:ssm:integration",
	)
	errInvalidEnv        = errors.New("invalid environment specified")
	errInvalidParamCombo = errors.New("invalid parameter combination")
	errInvalidTier       = errors.New("invalid tier specified")
	errListSSMParameters = errors.New("failed to list SSM parameters")
	errPutSSMParameter   = errors.New("failed to put SSM parameter")
	errReadFile          = errors.New("failed to read file")
	errValueRequired     = errors.New("VALUE is required when --file is not used")
	errValueWithFile     = errors.New("VALUE should not be provided when --file is used")
)

// newBriefAndFullError creates a new error for when the --brief and --full options are both specified.
func newBriefAndFullError() error {
	return fmt.Errorf("%w: it does not make sense to specify both --brief and --full", errInvalidParamCombo)
}

// newEnvUsageError creates a new error for when the --env flag is specified with an invalid flag.
func newEnvUsageError() error {
	return fmt.Errorf("%w: Cannot use --env with --full, --json, nor --verbose", errInvalidParamCombo)
}

// newFullAndVerboseError creates a new error for when the --full and --verbose options are both specified.
func newFullAndVerboseError() error {
	return fmt.Errorf("%w: it does not make sense to specify both --full and --verbose", errInvalidParamCombo)
}

// newInvalidEnvError creates a new error for when an invalid environment is specified.
func newInvalidEnvError(env string) error {
	return fmt.Errorf("%w: %s", errInvalidEnv, env)
}

// newInvalidTierError creates a new error for when an invalid SSM parameter store tier is specified.
func newInvalidTierError() error {
	ssmTiers := types.ParameterTier("").Values()
	ssmTiersStr := make([]string, len(ssmTiers))

	for i, tier := range ssmTiers {
		ssmTiersStr[i] = string(tier)
	}

	return fmt.Errorf("%w: Must be one of: %s", errInvalidTier, strings.Join(ssmTiersStr, ", "))
}
