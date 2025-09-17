package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/diagnosis/luxsuv-bookings/pkg/logger"
)

type ServiceProxy struct {
	baseURL string
	client  *http.Client
}

func NewServiceProxy(baseURL string) *ServiceProxy {
	return &ServiceProxy{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (p *ServiceProxy) ProxyRequest(ctx context.Context, method, path string, body []byte, headers map[string]string) (*http.Response, error) {
	url := p.baseURL + path
	
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Copy headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	
	// Add request ID for tracing
	if requestID := ctx.Value(logger.RequestIDKey); requestID != nil {
		req.Header.Set("X-Request-ID", requestID.(string))
	}
	
	// Add correlation headers for service mesh
	req.Header.Set("X-Gateway-Forwarded", "true")
	req.Header.Set("X-Gateway-Service", "luxsuv-gateway")
	
	logger.DebugContext(ctx, "Proxying request", 
		"method", method, 
		"url", url,
		"headers", headers,
	)
	
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	
	return resp, nil
}

func (p *ServiceProxy) Get(ctx context.Context, path string, headers map[string]string) (*http.Response, error) {
	return p.ProxyRequest(ctx, "GET", path, nil, headers)
}

func (p *ServiceProxy) Post(ctx context.Context, path string, body []byte, headers map[string]string) (*http.Response, error) {
	return p.ProxyRequest(ctx, "POST", path, body, headers)
}

func (p *ServiceProxy) Patch(ctx context.Context, path string, body []byte, headers map[string]string) (*http.Response, error) {
	return p.ProxyRequest(ctx, "PATCH", path, body, headers)
}

func (p *ServiceProxy) Delete(ctx context.Context, path string, headers map[string]string) (*http.Response, error) {
	return p.ProxyRequest(ctx, "DELETE", path, nil, headers)
}