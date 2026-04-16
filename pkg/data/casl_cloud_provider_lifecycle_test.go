package data

import "testing"

func TestCASLCloudProviderShutdownCancelsLifecycleContext(t *testing.T) {
	provider := NewCASLCloudProvider("", "", 0)
	ctx := provider.lifecycleContext()

	provider.Shutdown()

	select {
	case <-ctx.Done():
	default:
		t.Fatal("expected provider lifecycle context to be canceled")
	}
}
