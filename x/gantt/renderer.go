package gantt

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/go-fries/fries/x/gantt/v3/internal/parser"
	"github.com/go-fries/fries/x/gantt/v3/internal/render"
)

const defaultFilePerm = 0o644

type rendererImpl struct{}

var defaultRenderer = rendererImpl{}

// Render 实现
func (rendererImpl) Render(ctx context.Context, in Input) (RenderResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if in.Source == "" {
		return RenderResult{}, fmt.Errorf("source is empty")
	}
	if in.OutputPath == "" && in.Writer == nil {
		return RenderResult{}, fmt.Errorf("output target missing (OutputPath or Writer)")
	}

	var model parser.Model
	var err error
	if in.FromFile {
		if _, statErr := os.Stat(in.Source); statErr != nil {
			return RenderResult{}, fmt.Errorf("source file: %w", statErr)
		}
		model, err = parser.ParseFile(in.Source)
	} else {
		model, err = parser.Parse(in.Source)
	}
	if err != nil {
		return RenderResult{}, err
	}

	theme := MergeTheme(DefaultTheme(), in.Theme)
	colors := render.ThemeFromHex(
		theme.Background,
		theme.Grid,
		theme.TaskFill,
		theme.TaskBorder,
		theme.TaskText,
		theme.Text,
		theme.Milestone,
		theme.TodayLine,
	)

	opt := render.Options{
		Width:    in.Width,
		Height:   in.Height,
		Scale:    in.Scale,
		Theme:    colors,
		FontPath: in.FontPath,
	}

	imgBytes, err := render.RenderModel(ctx, model, opt)
	if err != nil {
		return RenderResult{}, err
	}

	res := RenderResult{Bytes: imgBytes}
	if in.OutputPath != "" {
		if writeErr := os.WriteFile(in.OutputPath, imgBytes, defaultFilePerm); writeErr != nil {
			return RenderResult{}, fmt.Errorf("write output: %w", writeErr)
		}
		res.OutputPath = in.OutputPath
	}
	if in.Writer != nil {
		if _, err := in.Writer.Write(imgBytes); err != nil {
			return RenderResult{}, fmt.Errorf("write to writer: %w", err)
		}
	}

	return res, nil
}

// Errors 定义
var (
	ErrInvalidInput = errors.New("invalid input")
)
