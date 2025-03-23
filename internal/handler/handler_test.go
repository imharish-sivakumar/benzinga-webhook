package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"benzinga-webhook/internal/model"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

type mockBatcher struct {
	entries []model.LogEntry
}

func (m *mockBatcher) Add(entry model.LogEntry) {
	m.entries = append(m.entries, entry)
}
func (m *mockBatcher) Start() {}
func (m *mockBatcher) Stop()  {}

func TestLogPayloadValidation(t *testing.T) {
	core, _ := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)
	validate := validator.New()
	err := validate.RegisterValidation("phoneformat", PhoneValidator)
	assert.Nil(t, err)
	batch := &mockBatcher{}
	h := New(logger, batch, validate)

	tests := []struct {
		name         string
		payload      *model.LogEntry
		expectCode   int
		expectedBody string
		rawBody      string
	}{
		{
			name: "valid request",
			payload: &model.LogEntry{
				UserID: 1,
				Total:  10.5,
				Title:  "valid title",
				Meta: model.Meta{
					Logins: []model.Login{{Time: "2020-08-08T01:52:50Z", IP: "127.0.0.1"}},
					PhoneNumbers: model.PhoneNumbers{
						Home:   "123-4567-891",
						Mobile: "765-4321-912",
					},
				},
				Completed: true,
			},
			expectCode:   http.StatusAccepted,
			expectedBody: `{"status":"Ok"}`,
		},
		{
			name: "invalid request - missing user_id",
			payload: &model.LogEntry{
				Total: 10.5,
				Title: "valid title",
				Meta: model.Meta{
					Logins: []model.Login{{Time: "2020-08-08T01:52:50Z", IP: "127.0.0.1"}},
					PhoneNumbers: model.PhoneNumbers{
						Home:   "123-4567-891",
						Mobile: "765-4321-912",
					},
				},
				Completed: true,
			},
			expectCode:   http.StatusBadRequest,
			expectedBody: `[{"UserID":"is required"}]`,
		},
		{
			name: "invalid request - bad IP address",
			payload: &model.LogEntry{
				UserID: 2,
				Total:  20.0,
				Title:  "invalid ip test",
				Meta: model.Meta{
					Logins: []model.Login{{Time: "2020-08-08T01:52:50Z", IP: "not-an-ip"}},
					PhoneNumbers: model.PhoneNumbers{
						Home:   "111-2222-123",
						Mobile: "222-3333-123",
					},
				},
				Completed: false,
			},
			expectCode:   http.StatusBadRequest,
			expectedBody: `[{"IP":"LogEntry.Meta.Logins[0].IP is invalid"}]`,
		},
		{
			name: "invalid request - malformed timestamp",
			payload: &model.LogEntry{
				UserID: 3,
				Total:  5.0,
				Title:  "bad timestamp",
				Meta: model.Meta{
					Logins: []model.Login{{Time: "08/08/2020", IP: "192.168.1.1"}},
					PhoneNumbers: model.PhoneNumbers{
						Home:   "333-4444-123",
						Mobile: "444-5555-123",
					},
				},
				Completed: true,
			},
			expectCode:   http.StatusBadRequest,
			expectedBody: `[{"Time":"LogEntry.Meta.Logins[0].Time is invalid"}]`,
		},
		{
			name: "invalid request - missing phone",
			payload: &model.LogEntry{
				UserID: 4,
				Total:  15.0,
				Title:  "missing phone",
				Meta: model.Meta{
					Logins: []model.Login{{Time: "2020-08-08T01:52:50Z", IP: "10.0.0.1"}},
					PhoneNumbers: model.PhoneNumbers{
						Home:   "",
						Mobile: "",
					},
				},
				Completed: false,
			},
			expectCode:   http.StatusBadRequest,
			expectedBody: `[{"Home":"is required"},{"Mobile":"is required"}]`,
		},
		{
			name: "invalid request - total is zero",
			payload: &model.LogEntry{
				UserID: 5,
				Total:  0,
				Title:  "zero total",
				Meta: model.Meta{
					Logins: []model.Login{{Time: "2020-08-08T01:52:50Z", IP: "1.1.1.1"}},
					PhoneNumbers: model.PhoneNumbers{
						Home: "123-4512-121", Mobile: "678-9011-123",
					},
				},
				Completed: false,
			},
			expectCode:   http.StatusBadRequest,
			expectedBody: `[{"Total":"is required"}]`,
		},
		{
			name: "invalid request - empty title",
			payload: &model.LogEntry{
				UserID: 6,
				Total:  12.5,
				Title:  "",
				Meta: model.Meta{
					Logins: []model.Login{{Time: "2020-08-08T01:52:50Z", IP: "8.8.8.8"}},
					PhoneNumbers: model.PhoneNumbers{
						Home: "321-1234-123", Mobile: "432-9688-123",
					},
				},
				Completed: true,
			},
			expectCode:   http.StatusBadRequest,
			expectedBody: `[{"Title":"is required"}]`,
		},
		{
			name:         "invalid request - timestamp parsing failure",
			expectedBody: `[{"Time":"LogEntry.Meta.Logins[0].Time is invalid"}]`,
			expectCode:   http.StatusBadRequest,
			payload:      nil,
			rawBody:      `{"user_id":1,"total":9.99,"title":"fail ts","meta":{"logins":[{"time":"not-a-time","ip":"127.0.0.1"}],"phone_numbers":{"home":"555-1212-123","mobile":"555-1212-456"}},"completed":true}`,
		},
		{
			name:         "invalid request body",
			expectedBody: `{"error":"invalid request payload"}`,
			expectCode:   http.StatusBadRequest,
			payload:      nil,
			rawBody:      `{`,
		},
		{
			name: "valid request with multiple logins",
			payload: &model.LogEntry{
				UserID: 7,
				Total:  100.25,
				Title:  "multi-login",
				Meta: model.Meta{
					Logins: []model.Login{
						{Time: "2022-01-01T00:00:00Z", IP: "10.0.0.1"},
						{Time: "2023-01-01T00:00:00Z", IP: "10.0.0.2"},
					},
					PhoneNumbers: model.PhoneNumbers{
						Home: "100-2001-123", Mobile: "200-3001-123",
					},
				},
				Completed: false,
			},
			expectCode:   http.StatusAccepted,
			expectedBody: `{"status":"Ok"}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var body []byte
			var err error
			if tc.rawBody != "" {
				body = []byte(tc.rawBody)
			} else {
				body, err = json.Marshal(tc.payload)
				assert.NoError(t, err)
			}
			r := httptest.NewRequest("POST", "/log", bytes.NewBuffer(body))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.LogPayload(w, r)
			assert.Equal(t, tc.expectCode, w.Code)

			all, err := io.ReadAll(w.Body)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedBody, strings.Trim(string(all), "\n"))
		})
	}

}

func TestHealthz(t *testing.T) {
	core, _ := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)
	validate := validator.New()
	batch := &mockBatcher{}
	h := New(logger, batch, validate)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	h.Healthz(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body, err := io.ReadAll(w.Body)
	assert.NoError(t, err)
	assert.Equal(t, "OK", string(body))
}
