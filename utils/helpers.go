package utils

func NotInSlice[T comparable](target T, slice []T) bool {
	for _, item := range slice {
		if item == target {
			return false
		}
	}
	return true
}
