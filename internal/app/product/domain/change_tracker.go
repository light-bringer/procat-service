package domain

// ChangeTracker tracks which fields have been modified in a domain aggregate.
// This allows repositories to optimize updates by only persisting changed fields.
type ChangeTracker struct {
	dirtyFields map[string]bool
}

// NewChangeTracker creates a new ChangeTracker.
func NewChangeTracker() *ChangeTracker {
	return &ChangeTracker{
		dirtyFields: make(map[string]bool),
	}
}

// MarkDirty marks a field as dirty (modified).
func (ct *ChangeTracker) MarkDirty(field string) {
	ct.dirtyFields[field] = true
}

// Dirty checks if a field has been modified.
func (ct *ChangeTracker) Dirty(field string) bool {
	return ct.dirtyFields[field]
}

// Clear clears all dirty field markers.
func (ct *ChangeTracker) Clear() {
	ct.dirtyFields = make(map[string]bool)
}

// HasChanges returns true if any field has been modified.
func (ct *ChangeTracker) HasChanges() bool {
	return len(ct.dirtyFields) > 0
}

// DirtyFields returns a slice of all dirty field names.
func (ct *ChangeTracker) DirtyFields() []string {
	fields := make([]string, 0, len(ct.dirtyFields))
	for field := range ct.dirtyFields {
		fields = append(fields, field)
	}
	return fields
}
