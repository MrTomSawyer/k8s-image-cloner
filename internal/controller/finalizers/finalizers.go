package finalizers

// RemoveFinalizer removes the specified finalizer (s) from the slice of finalizers (slice).
func RemoveFinalizer(slice []string, s string) []string {
	result := []string{}
	for _, item := range slice {
		if item != s {
			result = append(result, item)
		}
	}
	return result
}
