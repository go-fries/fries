package queue

import (
	"reflect"
	"sync"
)

var (
	typeRegistry   = make(map[string]reflect.Type)
	typeRegistryMu sync.RWMutex
)

// RegisterPayload registers a payload type for serialization/deserialization
// This is required when using drivers that need to serialize payloads (e.g., Redis)
// For in-memory drivers (channel), this is optional but recommended
func RegisterPayload[T any]() {
	typeRegistryMu.Lock()
	defer typeRegistryMu.Unlock()

	var t T
	typeName := reflect.TypeOf(t).String()
	typeRegistry[typeName] = reflect.TypeOf(t)
}

// newPayloadInstance creates a new instance of the registered type
// Returns nil if the type is not registered
func newPayloadInstance(typeName string) any {
	typeRegistryMu.RLock()
	defer typeRegistryMu.RUnlock()

	t, ok := typeRegistry[typeName]
	if !ok {
		return nil
	}

	// Create a new instance
	if t.Kind() == reflect.Ptr {
		return reflect.New(t.Elem()).Interface()
	}
	return reflect.New(t).Elem().Interface()
}

// isTypeRegistered checks if a type is registered
func isTypeRegistered(typeName string) bool {
	typeRegistryMu.RLock()
	defer typeRegistryMu.RUnlock()

	_, ok := typeRegistry[typeName]
	return ok
}

// registeredTypes returns all registered type names (for debugging)
func registeredTypes() []string {
	typeRegistryMu.RLock()
	defer typeRegistryMu.RUnlock()

	types := make([]string, 0, len(typeRegistry))
	for name := range typeRegistry {
		types = append(types, name)
	}
	return types
}
