package query

import "fmt"

// Condition represents a WHERE clause condition.
// Implementations must generate SQL fragments and parameter maps
// using Spanner's named parameter format (@paramName).
type Condition interface {
	// SQL returns the SQL fragment and parameter map for this condition.
	// paramIndex is used to generate unique parameter names (@p0, @p1, etc.)
	SQL(paramIndex int) (string, map[string]interface{})
}

// eqCondition implements equality comparison (field = value).
type eqCondition struct {
	field string
	value interface{}
}

// Eq creates a WHERE condition for equality comparison.
// Example: Eq("status", "active") generates "status = @p0"
func Eq(field string, value interface{}) Condition {
	return &eqCondition{
		field: field,
		value: value,
	}
}

// SQL generates the SQL fragment for equality comparison.
func (c *eqCondition) SQL(paramIndex int) (string, map[string]interface{}) {
	paramName := fmt.Sprintf("p%d", paramIndex)
	sql := fmt.Sprintf("%s = @%s", c.field, paramName)
	params := map[string]interface{}{
		paramName: c.value,
	}
	return sql, params
}

// IsNull creates a WHERE condition for NULL checks.
// Example: IsNull("discount_percent") generates "discount_percent IS NULL"
// Note: This is a placeholder for future extension.
func IsNull(field string) Condition {
	return &isNullCondition{field: field}
}

// isNullCondition implements IS NULL comparison.
type isNullCondition struct {
	field string
}

// SQL generates the SQL fragment for IS NULL comparison.
func (c *isNullCondition) SQL(paramIndex int) (string, map[string]interface{}) {
	sql := fmt.Sprintf("%s IS NULL", c.field)
	return sql, map[string]interface{}{}
}

// IsNotNull creates a WHERE condition for NOT NULL checks.
// Example: IsNotNull("discount_percent") generates "discount_percent IS NOT NULL"
// Note: This is a placeholder for future extension.
func IsNotNull(field string) Condition {
	return &isNotNullCondition{field: field}
}

// isNotNullCondition implements IS NOT NULL comparison.
type isNotNullCondition struct {
	field string
}

// SQL generates the SQL fragment for IS NOT NULL comparison.
func (c *isNotNullCondition) SQL(paramIndex int) (string, map[string]interface{}) {
	sql := fmt.Sprintf("%s IS NOT NULL", c.field)
	return sql, map[string]interface{}{}
}
