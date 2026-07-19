package health

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type handlerConfig struct {
	errorDetails bool
}

// HandlerOption configures an HTTP health handler.
type HandlerOption interface {
	applyHandler(*handlerConfig)
}

type handlerOptionFunc func(*handlerConfig)

func (f handlerOptionFunc) applyHandler(c *handlerConfig) {
	f(c)
}

// WithErrorDetails includes checker error messages in HTTP responses. Error
// messages may contain sensitive operational details, so this option should
// only be used on protected endpoints.
func WithErrorDetails() HandlerOption {
	return handlerOptionFunc(func(c *handlerConfig) {
		c.errorDetails = true
	})
}

func newHandlerConfig(options ...HandlerOption) handlerConfig {
	var c handlerConfig
	for _, option := range options {
		if option != nil {
			option.applyHandler(&c)
		}
	}
	return c
}

type httpHandler struct {
	registry     *Registry
	errorDetails bool
}

type httpResponse struct {
	Status   string              `json:"status"`
	Duration string              `json:"duration"`
	Checks   []httpCheckResponse `json:"checks"`
}

type httpCheckResponse struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Duration string `json:"duration"`
	Error    string `json:"error,omitempty"`
}

const (
	statusHealthy   = "healthy"
	statusUnhealthy = "unhealthy"
)

// Handler returns an HTTP handler that checks registry for GET and HEAD
// requests. Healthy reports use status 200 and unhealthy reports use status
// 503.
//
// Handler panics if registry is nil.
func Handler(registry *Registry, options ...HandlerOption) http.Handler {
	if registry == nil {
		panic("health: nil registry")
	}
	c := newHandlerConfig(options...)
	return &httpHandler{
		registry:     registry,
		errorDetails: c.errorDetails,
	}
}

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodHead)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	report := h.registry.Check(r.Context())
	response := h.response(report)
	payload, err := json.Marshal(response)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	payload = append(payload, '\n')

	statusCode := http.StatusOK
	if !report.Healthy() {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
	w.WriteHeader(statusCode)
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write(payload)
}

func (h *httpHandler) response(report Report) httpResponse {
	response := httpResponse{
		Status:   statusHealthy,
		Duration: report.Duration.String(),
		Checks:   make([]httpCheckResponse, len(report.Results)),
	}
	if !report.Healthy() {
		response.Status = statusUnhealthy
	}

	for i, result := range report.Results {
		check := httpCheckResponse{
			Name:     result.Name,
			Status:   statusHealthy,
			Duration: result.Duration.String(),
		}
		if !result.Healthy() {
			check.Status = statusUnhealthy
			if h.errorDetails {
				check.Error = result.Err.Error()
			}
		}
		response.Checks[i] = check
	}
	return response
}
