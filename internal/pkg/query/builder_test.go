package query

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuilder_BasicSelect(t *testing.T) {
	stmt := From("products").
		Select("product_id", "name", "category").
		Build()

	assert.Equal(t, "SELECT product_id, name, category FROM products", stmt.SQL)
	assert.Empty(t, stmt.Params)
}

func TestBuilder_SelectAllColumns(t *testing.T) {
	stmt := From("products").Build()

	assert.Equal(t, "SELECT * FROM products", stmt.SQL)
	assert.Empty(t, stmt.Params)
}

func TestBuilder_SingleWhereCondition(t *testing.T) {
	stmt := From("products").
		Select("product_id", "name").
		Where(Eq("category", "electronics")).
		Build()

	assert.Equal(t, "SELECT product_id, name FROM products WHERE category = @p0", stmt.SQL)
	assert.Equal(t, map[string]interface{}{
		"p0": "electronics",
	}, stmt.Params)
}

func TestBuilder_MultipleWhereConditions(t *testing.T) {
	stmt := From("products").
		Select("product_id", "name").
		Where(Eq("category", "electronics")).
		Where(Eq("status", "active")).
		Build()

	assert.Equal(t, "SELECT product_id, name FROM products WHERE category = @p0 AND status = @p1", stmt.SQL)
	assert.Equal(t, map[string]interface{}{
		"p0": "electronics",
		"p1": "active",
	}, stmt.Params)
}

func TestBuilder_OrderByAsc(t *testing.T) {
	stmt := From("products").
		Select("product_id", "name").
		OrderBy("created_at", Asc).
		Build()

	assert.Equal(t, "SELECT product_id, name FROM products ORDER BY created_at ASC", stmt.SQL)
	assert.Empty(t, stmt.Params)
}

func TestBuilder_OrderByDesc(t *testing.T) {
	stmt := From("products").
		Select("product_id", "name").
		OrderBy("created_at", Desc).
		Build()

	assert.Equal(t, "SELECT product_id, name FROM products ORDER BY created_at DESC", stmt.SQL)
	assert.Empty(t, stmt.Params)
}

func TestBuilder_Limit(t *testing.T) {
	stmt := From("products").
		Select("product_id", "name").
		Limit(10).
		Build()

	assert.Equal(t, "SELECT product_id, name FROM products LIMIT @limit", stmt.SQL)
	assert.Equal(t, map[string]interface{}{
		"limit": int64(10),
	}, stmt.Params)
}

func TestBuilder_Offset(t *testing.T) {
	stmt := From("products").
		Select("product_id", "name").
		Offset(20).
		Build()

	assert.Equal(t, "SELECT product_id, name FROM products OFFSET @offset", stmt.SQL)
	assert.Equal(t, map[string]interface{}{
		"offset": int64(20),
	}, stmt.Params)
}

func TestBuilder_LimitAndOffset(t *testing.T) {
	stmt := From("products").
		Select("product_id", "name").
		Limit(10).
		Offset(20).
		Build()

	assert.Equal(t, "SELECT product_id, name FROM products LIMIT @limit OFFSET @offset", stmt.SQL)
	assert.Equal(t, map[string]interface{}{
		"limit":  int64(10),
		"offset": int64(20),
	}, stmt.Params)
}

func TestBuilder_CompleteQuery(t *testing.T) {
	stmt := From("products").
		Select("product_id", "name", "category", "status").
		Where(Eq("category", "electronics")).
		Where(Eq("status", "active")).
		OrderBy("created_at", Desc).
		Limit(50).
		Offset(100).
		Build()

	expectedSQL := "SELECT product_id, name, category, status FROM products WHERE category = @p0 AND status = @p1 ORDER BY created_at DESC LIMIT @limit OFFSET @offset"
	assert.Equal(t, expectedSQL, stmt.SQL)
	assert.Equal(t, map[string]interface{}{
		"p0":     "electronics",
		"p1":     "active",
		"limit":  int64(50),
		"offset": int64(100),
	}, stmt.Params)
}

func TestBuilder_Count(t *testing.T) {
	builder := From("products").
		Select("product_id", "name", "category").
		Where(Eq("category", "electronics")).
		Where(Eq("status", "active")).
		OrderBy("created_at", Desc).
		Limit(50).
		Offset(100)

	// Main query
	mainStmt := builder.Build()
	assert.Contains(t, mainStmt.SQL, "SELECT product_id, name, category FROM products")
	assert.Contains(t, mainStmt.SQL, "LIMIT @limit")
	assert.Contains(t, mainStmt.SQL, "OFFSET @offset")

	// Count query - should reuse WHERE but not pagination/ordering
	countStmt := builder.Count().Build()
	assert.Equal(t, "SELECT COUNT(*) FROM products WHERE category = @p0 AND status = @p1", countStmt.SQL)
	assert.Equal(t, map[string]interface{}{
		"p0": "electronics",
		"p1": "active",
	}, countStmt.Params)

	// Verify original builder is unchanged (immutability)
	mainStmt2 := builder.Build()
	assert.Equal(t, mainStmt.SQL, mainStmt2.SQL)
}

func TestBuilder_CountWithoutFilters(t *testing.T) {
	stmt := From("products").
		Select("product_id", "name").
		Count().
		Build()

	assert.Equal(t, "SELECT COUNT(*) FROM products", stmt.SQL)
	assert.Empty(t, stmt.Params)
}

func TestBuilder_Immutability(t *testing.T) {
	base := From("products").Select("product_id")

	// Add different WHERE conditions
	stmt1 := base.Where(Eq("status", "active")).Build()
	stmt2 := base.Where(Eq("category", "electronics")).Build()

	// Both should have their own conditions
	assert.Contains(t, stmt1.SQL, "status = @p0")
	assert.NotContains(t, stmt1.SQL, "category")

	assert.Contains(t, stmt2.SQL, "category = @p0")
	assert.NotContains(t, stmt2.SQL, "status")
}

func TestBuilder_EmptyWhere(t *testing.T) {
	stmt := From("products").
		Select("product_id", "name").
		OrderBy("created_at", Desc).
		Build()

	assert.Equal(t, "SELECT product_id, name FROM products ORDER BY created_at DESC", stmt.SQL)
	assert.Empty(t, stmt.Params)
}

func TestBuilder_OnlyWhereNoOrderOrPagination(t *testing.T) {
	stmt := From("products").
		Select("product_id").
		Where(Eq("status", "active")).
		Build()

	assert.Equal(t, "SELECT product_id FROM products WHERE status = @p0", stmt.SQL)
	assert.Equal(t, map[string]interface{}{
		"p0": "active",
	}, stmt.Params)
}

func TestCondition_Eq(t *testing.T) {
	cond := Eq("status", "active")
	sql, params := cond.SQL(0)

	assert.Equal(t, "status = @p0", sql)
	assert.Equal(t, map[string]interface{}{
		"p0": "active",
	}, params)
}

func TestCondition_EqWithDifferentParamIndex(t *testing.T) {
	cond := Eq("category", "electronics")
	sql, params := cond.SQL(5)

	assert.Equal(t, "category = @p5", sql)
	assert.Equal(t, map[string]interface{}{
		"p5": "electronics",
	}, params)
}

func TestCondition_IsNull(t *testing.T) {
	cond := IsNull("discount_percent")
	sql, params := cond.SQL(0)

	assert.Equal(t, "discount_percent IS NULL", sql)
	assert.Empty(t, params)
}

func TestCondition_IsNotNull(t *testing.T) {
	cond := IsNotNull("discount_percent")
	sql, params := cond.SQL(0)

	assert.Equal(t, "discount_percent IS NOT NULL", sql)
	assert.Empty(t, params)
}

func TestBuilder_String(t *testing.T) {
	builder := From("products").
		Select("product_id", "name").
		Where(Eq("status", "active"))

	str := builder.String()
	require.NotEmpty(t, str)
	assert.Contains(t, str, "SQL:")
	assert.Contains(t, str, "Params:")
	assert.Contains(t, str, "products")
}

func TestBuilder_WhereWithIsNull(t *testing.T) {
	stmt := From("products").
		Select("product_id", "name").
		Where(Eq("status", "active")).
		Where(IsNull("discount_percent")).
		Build()

	assert.Equal(t, "SELECT product_id, name FROM products WHERE status = @p0 AND discount_percent IS NULL", stmt.SQL)
	assert.Equal(t, map[string]interface{}{
		"p0": "active",
	}, stmt.Params)
}

func TestBuilder_MultipleSelectCalls(t *testing.T) {
	stmt := From("products").
		Select("product_id", "name").
		Select("category", "status").
		Build()

	assert.Equal(t, "SELECT product_id, name, category, status FROM products", stmt.SQL)
	assert.Empty(t, stmt.Params)
}
