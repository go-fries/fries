package prints

import "github.com/cheggaaa/pb/v3"

var (
	Full    = pb.Full
	Default = pb.Default
	Simple  = pb.Simple

	FullTemplate    = WithTemplate(Full)
	DefaultTemplate = WithTemplate(Default)
	SimpleTemplate  = WithTemplate(Simple)
)

type ProgressBar struct {
	*pb.ProgressBar

	template pb.ProgressBarTemplate
}

type ProgressBarOption func(*ProgressBar)

func WithTemplate(template pb.ProgressBarTemplate) ProgressBarOption {
	return func(o *ProgressBar) {
		o.template = template
	}
}

func NewProgressBar(total int, opts ...ProgressBarOption) *ProgressBar {
	p := &ProgressBar{}

	for _, opt := range opts {
		opt(p)
	}

	if p.template != "" {
		p.ProgressBar = p.template.Start(total)
	} else {
		p.ProgressBar = pb.StartNew(total)
	}

	return p
}

func WithProgressBar(total int, fc func(bar *ProgressBar), opts ...ProgressBarOption) {
	bar := NewProgressBar(total, opts...)
	fc(bar)
	bar.Finish()
}
