package utils

// Makes a ternary operation that returns one of two values based on a boolean condition.
func Ternary[T any](cond bool, a, b T) T {
	if cond {
		return a
	}
	return b
}
