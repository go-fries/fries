package prints

import (
	"bufio"
	"os"
	"strings"

	"golang.org/x/term"
)

type Prompt struct {
	question string

	defaultAnswer string
	secret        bool
	trimSpace     bool
}

type PromptOption func(*Prompt)

func WithDefaultAnswer(defaultAnswer string) PromptOption {
	return func(p *Prompt) {
		p.defaultAnswer = defaultAnswer
	}
}

func WithSecret() PromptOption {
	return func(p *Prompt) {
		p.secret = true
	}
}

func WithTrimSpace(flag bool) PromptOption {
	return func(p *Prompt) {
		p.trimSpace = flag
	}
}

func NewPrompt(question string, opts ...PromptOption) *Prompt {
	p := &Prompt{
		question:  question,
		trimSpace: true,
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

func (p *Prompt) Ask() (string, error) {
	if err := p.output(); err != nil {
		return "", err
	}

	input, err := p.input()
	if err != nil {
		return "", err
	}

	if p.trimSpace {
		input = strings.TrimSpace(input)
	}

	if input == "" {
		return p.defaultAnswer, nil
	}
	return input, nil
}

func (p *Prompt) output() error {
	if _, err := Infof("%s ", p.question); err != nil {
		return err
	}

	if p.defaultAnswer != "" {
		if _, err := Warnf("[%s] ", p.defaultAnswer); err != nil {
			return err
		}
	}

	if _, err := Linef("\n> "); err != nil {
		return err
	}

	return nil
}

func (p *Prompt) input() (string, error) {
	// support secret
	if p.secret {
		input, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return "", err
		}
		NewLine() //nolint:errcheck // add a new line
		return string(input), nil
	}

	// normal input
	reader := bufio.NewReader(os.Stdin)
	return reader.ReadString('\n')
}

func Ask(question string, defaults ...string) (string, error) {
	var opts []PromptOption
	if len(defaults) > 0 {
		opts = append(opts, WithDefaultAnswer(defaults[0]))
	}

	return NewPrompt(question, opts...).Ask()
}

func Secret(question string, defaults ...string) (string, error) {
	var opts []PromptOption
	if len(defaults) > 0 {
		opts = append(opts, WithDefaultAnswer(defaults[0]))
	}
	opts = append(opts, WithSecret(), WithTrimSpace(false))

	return NewPrompt(question, opts...).Ask()
}
