package committer

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
)

// CommitPlan is a typed wrapper around Spanner mutations for the Golden Mutation Pattern.
// It collects mutations from multiple sources and applies them atomically.
type CommitPlan struct {
	mutations []*spanner.Mutation
}

// NewPlan creates a new empty CommitPlan.
func NewPlan() *CommitPlan {
	return &CommitPlan{
		mutations: make([]*spanner.Mutation, 0),
	}
}

// Add adds a mutation to the plan.
// Nil mutations are silently ignored for convenience.
func (cp *CommitPlan) Add(mut *spanner.Mutation) {
	if mut != nil {
		cp.mutations = append(cp.mutations, mut)
	}
}

// AddMultiple adds multiple mutations to the plan.
func (cp *CommitPlan) AddMultiple(muts []*spanner.Mutation) {
	for _, mut := range muts {
		cp.Add(mut)
	}
}

// Mutations returns all collected mutations.
func (cp *CommitPlan) Mutations() []*spanner.Mutation {
	return cp.mutations
}

// IsEmpty returns true if the plan has no mutations.
func (cp *CommitPlan) IsEmpty() bool {
	return len(cp.mutations) == 0
}

// Count returns the number of mutations in the plan.
func (cp *CommitPlan) Count() int {
	return len(cp.mutations)
}

// Committer provides transaction execution for CommitPlans.
type Committer struct {
	client *spanner.Client
}

// NewCommitter creates a new Committer.
func NewCommitter(client *spanner.Client) *Committer {
	return &Committer{client: client}
}

// Apply executes the CommitPlan atomically within a Spanner transaction.
func (c *Committer) Apply(ctx context.Context, plan *CommitPlan) error {
	if plan.IsEmpty() {
		return nil // Nothing to commit
	}

	_, err := c.client.Apply(ctx, plan.Mutations())
	if err != nil {
		return fmt.Errorf("failed to apply commit plan: %w", err)
	}

	return nil
}

// ApplyWithReadWriteTransaction executes the CommitPlan within a read-write transaction.
// This is useful when you need to perform reads before building mutations.
func (c *Committer) ApplyWithReadWriteTransaction(ctx context.Context, fn func(context.Context, *spanner.ReadWriteTransaction) error) error {
	_, err := c.client.ReadWriteTransaction(ctx, fn)
	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}
	return nil
}

// ApplyWithVersionCheck executes the CommitPlan with optimistic locking.
// It verifies the version hasn't changed before applying mutations.
// Parameters:
//   - productID: The ID of the product being updated
//   - expectedVersion: The version the aggregate had when it was loaded
//   - plan: The CommitPlan containing mutations to apply
//
// Returns ErrOptimisticLockConflict if the version in the database doesn't match expectedVersion.
func (c *Committer) ApplyWithVersionCheck(ctx context.Context, productID string, expectedVersion int64, plan *CommitPlan) error {
	if plan.IsEmpty() {
		return nil // Nothing to commit
	}

	_, err := c.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Read current version from database
		row, err := txn.ReadRow(ctx, "products", spanner.Key{productID}, []string{"version"})
		if err != nil {
			return fmt.Errorf("failed to read product version: %w", err)
		}

		var currentVersion int64
		if err := row.Column(0, &currentVersion); err != nil {
			return fmt.Errorf("failed to parse version: %w", err)
		}

		// Check if version matches (optimistic lock)
		if currentVersion != expectedVersion {
			return fmt.Errorf("version mismatch: expected %d, got %d (concurrent modification detected)", expectedVersion, currentVersion)
		}

		// Version matches, apply mutations
		return txn.BufferWrite(plan.Mutations())
	})

	if err != nil {
		// Check if it's a version mismatch error
		errMsg := err.Error()
		if strings.Contains(errMsg, "version mismatch") || strings.Contains(errMsg, "concurrent modification") {
			return fmt.Errorf("optimistic lock conflict: %w", err)
		}
		return fmt.Errorf("failed to apply commit plan with version check: %w", err)
	}

	return nil
}
