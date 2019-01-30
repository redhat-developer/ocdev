# Validation architecture

User-specified input needs to be validated as early as possible, i.e. before being sent to the remote server so that the user
can benefit from fast feedback. `odo` therefore defines an input validation architecture in order to validate user-specified
values.

## Architecture

`struct`s holding user-specified values can "extend" the `Validatable` type (via embedding) to provide metadata to the 
validation system. In particular, `Validatable` fields allow the developer to specify whether this particular value needs to 
specified (`Required`) and which kind of values it accepts so that the validation system can perform basic validation. 

Additionally, the developer can specify an array of `Validator` functions that also need to be applied to the input value. When
`odo` interacts with the user and expects the user to provide values for some parameters, the developer of the associated 
command can therefore "decorate" their data structure meant to receive the user input with the `Validatable` type so that `odo`
can automatically perform validation of provided values. This is used in conjunction with the library we use to deal with user
interaction (`survey`) which allows developer to specify a `Validator` function to validate values provided by users. Developers
can use the `GetValidatorFor` function to have `odo` automatically create an appropriate validator for the expected value based
on metadata provided via `Validatable`.


## Default validators

`odo` provides default validators in the `validation` package (`validators.go` file) to validate that a value can be converted 
to an `int` (`IntegerValidator`), that the value is a valid Kubernetes name (`NameValidator`) or a so-called `NilValidator`
which is a noop validator used as a default validator when none is provided or can be inferred from provided metadata. More 
validators could be provided, in particular, validators based on `Validatable.Type`, which currently only map `IntegerValidator`
to `Validatable` with `integer` as `Type`.

## Creating a validator

Validators are defined as follows: `type Validator func(interface{}) error`. Therefore, providing new validators is as easy as
providing a function taking an `interface{}` as parameter and returning a non-nil error if validation failed for any reason. If
the value is deemed valid, the function then returns `nil`.

Note: if a plugin system is developed for `odo`, new validators could be a provided via plugins.
