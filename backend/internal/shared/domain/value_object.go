package domain

// ValueObject is a marker for immutable value objects.
// Implementations must be immutable: no setters, created via constructors.
type ValueObject interface {
	Equals(other ValueObject) bool
}
