package query

import (
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
)

// Direction represents ORDER BY direction.
type Direction int

const (
	// Asc represents ascending order.
	Asc Direction = iota
	// Desc represents descending order.
	Desc
)

// Builder constructs SQL SELECT queries for Cloud Spanner.
// It provides a fluent API for building queries with WHERE clauses,
// ORDER BY, LIMIT, and OFFSET. Auto-generates parameter names to
// prevent manual synchronization errors.
type Builder struct {
	table        string
	selectCols   []string
	whereClauses []Condition
	orderByCol   string
	orderByDir   Direction
	limitVal     int64
	offsetVal    int64
	paramCounter int
}

// From creates a new Builder for the specified table.
func From(table string) *Builder {
	return &Builder{
		table:        table,
		selectCols:   []string{},
		whereClauses: []Condition{},
		paramCounter: 0,
	}
}

// Select specifies the columns to retrieve.
// Call this method to avoid duplicating column lists.
func (b *Builder) Select(columns ...string) *Builder {
	newBuilder := b.clone()
	newBuilder.selectCols = append(newBuilder.selectCols, columns...)
	return newBuilder
}

// Where adds a WHERE condition.
// Multiple calls are combined with AND logic.
func (b *Builder) Where(condition Condition) *Builder {
	newBuilder := b.clone()
	newBuilder.whereClauses = append(newBuilder.whereClauses, condition)
	return newBuilder
}

// OrderBy specifies the column and direction for sorting.
func (b *Builder) OrderBy(column string, direction Direction) *Builder {
	newBuilder := b.clone()
	newBuilder.orderByCol = column
	newBuilder.orderByDir = direction
	return newBuilder
}

// Limit sets the maximum number of rows to return.
func (b *Builder) Limit(limit int64) *Builder {
	newBuilder := b.clone()
	newBuilder.limitVal = limit
	return newBuilder
}

// Offset sets the number of rows to skip.
func (b *Builder) Offset(offset int64) *Builder {
	newBuilder := b.clone()
	newBuilder.offsetVal = offset
	return newBuilder
}

// Count returns a new builder that generates a COUNT(*) query
// with the same FROM and WHERE clauses.
// This eliminates duplication when you need both result rows and total count.
func (b *Builder) Count() *Builder {
	newBuilder := b.clone()
	newBuilder.selectCols = []string{"COUNT(*)"}
	// Clear pagination for count query
	newBuilder.limitVal = 0
	newBuilder.offsetVal = 0
	newBuilder.orderByCol = ""
	return newBuilder
}

// Build constructs the final spanner.Statement with SQL and parameters.
func (b *Builder) Build() spanner.Statement {
	var sql strings.Builder
	params := make(map[string]interface{})

	// SELECT clause
	sql.WriteString("SELECT ")
	if len(b.selectCols) == 0 {
		sql.WriteString("*")
	} else {
		sql.WriteString(strings.Join(b.selectCols, ", "))
	}

	// FROM clause
	sql.WriteString(" FROM ")
	sql.WriteString(b.table)

	// WHERE clause
	if len(b.whereClauses) > 0 {
		sql.WriteString(" WHERE ")
		whereParts := make([]string, 0, len(b.whereClauses))
		paramIndex := 0
		for _, condition := range b.whereClauses {
			fragment, condParams := condition.SQL(paramIndex)
			whereParts = append(whereParts, fragment)
			for k, v := range condParams {
				params[k] = v
			}
			paramIndex += len(condParams)
		}
		sql.WriteString(strings.Join(whereParts, " AND "))
	}

	// ORDER BY clause
	if b.orderByCol != "" {
		sql.WriteString(" ORDER BY ")
		sql.WriteString(b.orderByCol)
		if b.orderByDir == Desc {
			sql.WriteString(" DESC")
		} else {
			sql.WriteString(" ASC")
		}
	}

	// LIMIT clause
	if b.limitVal > 0 {
		sql.WriteString(" LIMIT @limit")
		params["limit"] = b.limitVal
	}

	// OFFSET clause
	if b.offsetVal > 0 {
		sql.WriteString(" OFFSET @offset")
		params["offset"] = b.offsetVal
	}

	return spanner.Statement{
		SQL:    sql.String(),
		Params: params,
	}
}

// clone creates a shallow copy of the builder for immutability.
func (b *Builder) clone() *Builder {
	newBuilder := &Builder{
		table:        b.table,
		selectCols:   make([]string, len(b.selectCols)),
		whereClauses: make([]Condition, len(b.whereClauses)),
		orderByCol:   b.orderByCol,
		orderByDir:   b.orderByDir,
		limitVal:     b.limitVal,
		offsetVal:    b.offsetVal,
		paramCounter: b.paramCounter,
	}
	copy(newBuilder.selectCols, b.selectCols)
	copy(newBuilder.whereClauses, b.whereClauses)
	return newBuilder
}

// String returns a human-readable representation for debugging.
func (b *Builder) String() string {
	stmt := b.Build()
	return fmt.Sprintf("SQL: %s\nParams: %v", stmt.SQL, stmt.Params)
}
