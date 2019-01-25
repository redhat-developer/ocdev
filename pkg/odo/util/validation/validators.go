package validation

import (
	"fmt"
	"gopkg.in/AlecAivazis/survey.v1"
	"strconv"
)

func NameValidator(name interface{}) error {
	s := name.(string)
	return ValidateName(s)
}

// Validator is a function that validates that the provided interface conforms to expectations or return an error
type Validator func(interface{}) error

// NilValidator always validates
func NilValidator(interface{}) error { return nil }

func IntegerValidator(ans interface{}) error {
	s := ans.(string)
	_, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("invalid integer value '%s': %s", s, err)
	}
	return nil
}

// GetValidatorFor retrieves a validator for the specified validatable, first validating its required state, then its value
// based on type then any additional validators in the order specified by Validatable.AdditionalValidators
func GetValidatorFor(prop Validatable) Validator {
	v, _ := internalGetValidatorFor(prop)
	return v
}

// internalGetValidatorFor exposed for testing purposes
func internalGetValidatorFor(prop Validatable) (validator Validator, chain []survey.Validator) {
	// make sure we don't run into issues when composing validators
	validatorChain := make([]survey.Validator, 0, 5)

	if prop.Required {
		validatorChain = append(validatorChain, survey.Required)
	}

	switch prop.Type {
	case "integer":
		validatorChain = append(validatorChain, survey.Validator(IntegerValidator))
	}

	for i := range prop.AdditionalValidators {
		validatorChain = append(validatorChain, survey.Validator(prop.AdditionalValidators[i]))
	}

	if len(validatorChain) > 0 {
		return Validator(survey.ComposeValidators(validatorChain...)), validatorChain
	}

	return NilValidator, validatorChain
}
