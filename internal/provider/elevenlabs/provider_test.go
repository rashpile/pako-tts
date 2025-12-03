package elevenlabs

import (
	"testing"
)

func TestNewProvider(t *testing.T) {
	provider := NewProvider("test-api-key", true)

	if provider == nil {
		t.Fatal("Expected non-nil provider")
	}
	if provider.client == nil {
		t.Error("Expected client to be initialized")
	}
	if !provider.isDefault {
		t.Error("Expected isDefault to be true")
	}
}

func TestProvider_Name(t *testing.T) {
	provider := NewProvider("test-api-key", true)

	name := provider.Name()

	if name != "elevenlabs" {
		t.Errorf("Expected name 'elevenlabs', got %s", name)
	}
}

func TestProvider_MaxConcurrent(t *testing.T) {
	provider := NewProvider("test-api-key", true)

	maxConcurrent := provider.MaxConcurrent()

	if maxConcurrent != 4 {
		t.Errorf("Expected maxConcurrent 4, got %d", maxConcurrent)
	}
}

func TestProvider_ActiveJobs(t *testing.T) {
	provider := NewProvider("test-api-key", true)

	activeJobs := provider.ActiveJobs()

	if activeJobs != 0 {
		t.Errorf("Expected activeJobs 0, got %d", activeJobs)
	}
}

func TestGetFloatValue(t *testing.T) {
	tests := []struct {
		name        string
		ptr         *float64
		defaultVal  float64
		expected    float64
	}{
		{"nil pointer", nil, 0.5, 0.5},
		{"non-nil pointer", ptrFloat(0.8), 0.5, 0.8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getFloatValue(tt.ptr, tt.defaultVal)
			if result != tt.expected {
				t.Errorf("Expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestGetBoolValue(t *testing.T) {
	tests := []struct {
		name       string
		ptr        *bool
		defaultVal bool
		expected   bool
	}{
		{"nil pointer true default", nil, true, true},
		{"nil pointer false default", nil, false, false},
		{"non-nil pointer true", ptrBool(true), false, true},
		{"non-nil pointer false", ptrBool(false), true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getBoolValue(tt.ptr, tt.defaultVal)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func ptrFloat(f float64) *float64 {
	return &f
}

func ptrBool(b bool) *bool {
	return &b
}
