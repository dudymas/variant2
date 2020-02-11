package app

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/zclconf/go-cty/cty"

	"strconv"
	"strings"
)

type PendingOption struct {
	Spec OptionSpec
	Type cty.Type
}

func makeQuestions(pendingOptions []PendingOption) ([]*survey.Question, map[string]survey.Transformer, error) {
	qs := []*survey.Question{}

	strTransformers := map[string]survey.Transformer{}

	for _, op := range pendingOptions {
		name := op.Spec.Name

		var msg string

		var description string

		if op.Spec.Description != nil {
			description = *op.Spec.Description
		}

		msg = name

		var validate survey.Validator

		var transform survey.Transformer

		var prompt survey.Prompt

		switch op.Type {
		case cty.String:
			prompt = &survey.Input{
				Message: msg,
				Help:    description,
			}
		case cty.Number:
			prompt = &survey.Input{
				Message: msg,
				Help:    description,
			}

			transform = func(ans interface{}) (newAns interface{}) {
				i, _ := strconv.Atoi(ans.(string))
				return i
			}

			validate = func(ans interface{}) error {
				switch v := ans.(type) {
				case string:
					if _, err := strconv.Atoi(v); err != nil {
						return fmt.Errorf("option %q: %v", name, err)
					}
				default:
					return fmt.Errorf("option %q: number: unexpected type of input %T", name, v)
				}

				return nil
			}
		case cty.Bool:
			prompt = &survey.Confirm{
				Message: msg,
				Help:    description,
				Default: false,
			}
		case cty.List(cty.String):
			prompt = &survey.Multiline{
				Message: msg,
				Help:    description,
			}

			transform = func(ans interface{}) (newAns interface{}) {
				lines := strings.Split(ans.(string), "\n")
				return lines
			}

			validate = func(ans interface{}) error {
				switch v := ans.(type) {
				case string:
				default:
					return fmt.Errorf("option %q: list(string): unexpected type of input %T", name, v)
				}

				return nil
			}
		case cty.List(cty.Number):
			prompt = &survey.Multiline{
				Message: msg,
				Help:    description,
			}

			transform = func(ans interface{}) (newAns interface{}) {
				lines := strings.Split(ans.(string), "\n")

				var ints []int

				for _, line := range lines {
					i, _ := strconv.Atoi(line)
					ints = append(ints, i)
				}

				return ints
			}

			validate = func(ans interface{}) error {
				switch v := ans.(type) {
				case string:
					vs := strings.Split(v, "\n")

					for _, a := range vs {
						_, err := strconv.Atoi(a)
						if err != nil {
							return fmt.Errorf("option %q: list(number): atoi: %w", name, err)
						}
					}
				default:
					return fmt.Errorf("option %q: list(number): unexpected type of input %T", name, v)
				}

				return nil
			}
		default:
			return nil, nil, fmt.Errorf("option %q: unexpected type %q", op.Spec.Name, op.Type.FriendlyName())
		}

		validators := []survey.Validator{survey.Required}

		if validate != nil {
			validators = append(validators, validate)
		}

		qs = append(qs, &survey.Question{
			Name:     name,
			Prompt:   prompt,
			Validate: survey.ComposeValidators(validators...),
		})

		if transform != nil {
			strTransformers[name] = transform
		}
	}

	return qs, strTransformers, nil
}

func setOpts(opts map[string]cty.Value, pendingOptions []PendingOption) error {
	qs, strTransformers, err := makeQuestions(pendingOptions)
	if err != nil {
		return err
	}

	res := make(map[string]interface{})

	if err := survey.Ask(qs, &res); err != nil {
		return err
	}

	for k, v := range res {
		t, ok := strTransformers[k]

		var ans interface{}

		if ok {
			ans = t(v)
		} else {
			ans = v
		}

		switch v := ans.(type) {
		case int:
			opts[k] = cty.NumberIntVal(int64(v))
		case string:
			opts[k] = cty.StringVal(v)
		case []string:
			vs := []cty.Value{}
			for _, s := range v {
				vs = append(vs, cty.StringVal(s))
			}

			opts[k] = cty.ListVal(vs)
		case []int:
			vs := []cty.Value{}
			for _, i := range v {
				vs = append(vs, cty.NumberIntVal(int64(i)))
			}

			opts[k] = cty.ListVal(vs)
		case bool:
			opts[k] = cty.BoolVal(v)
		default:
			return fmt.Errorf("option %q: parsing answer: unexpected type %T", k, v)
		}
	}

	return nil
}
