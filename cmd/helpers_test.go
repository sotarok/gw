package cmd

import "bytes"

// Helper function to create a bool pointer
func boolPtr(b bool) *bool {
	return &b
}

// Helper functions

func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}
