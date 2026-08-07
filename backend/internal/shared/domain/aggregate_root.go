package domain

// AggregateRoot marks a DDD aggregate root. All access to entities inside
// the aggregate goes through the root.
type AggregateRoot struct {
	Entity
}
