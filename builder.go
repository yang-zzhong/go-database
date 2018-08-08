package querybuilder

import (
	str "strings"
)

//
// Quote's callback type
//
type QuoteWhere func(builder *Builder)

const (
	DESC string = "DESC"
	ASC  string = "ASC"
)

const (
	t_WHERE = iota
	t_RAW
	t_QUOTE_BEGIN
	t_QUOTE_END
	t_AND
	t_OR
)

type Condition struct {
	t  int    // condition type
	id string // condition id
}

type Builder struct {
	// users
	tableName string

	conditions []Condition

	wheres map[string]Where

	/**
	 * ["name", "age"]
	 * ["*"]
	 */
	selects []column

	/**
	 * ["age" => "desc"]
	 */
	orders map[string]string

	whereFactory *WhereFactory

	values []interface{}

	limit int

	offset int

	modifier Modifier

	group string

	replace bool

	isPrepare bool
}

func NewBuilder(modifier Modifier) *Builder {
	builder := new(Builder)
	builder.modifier = modifier
	builder.Init()
	return builder
}

func (builder *Builder) Init() {
	builder.conditions = []Condition{}
	builder.wheres = make(map[string]Where)
	builder.selects = []column{{"*", false}}
	builder.orders = make(map[string]string)
	builder.whereFactory = NewWF(builder.modifier)
	builder.values = []interface{}{}
	builder.limit = -1
	builder.offset = -1
	builder.replace = true
}

func (builder *Builder) SetIsPrepare(isPrepare bool) *Builder {
	builder.isPrepare = isPrepare
	return builder
}

func (builder *Builder) From(tableName string) *Builder {
	builder.tableName = tableName
	return builder
}

func (builder *Builder) OrderBy(field string, order string) *Builder {
	builder.orders[field] = order
	return builder
}

func (builder *Builder) Limit(limit int) *Builder {
	builder.limit = limit
	return builder
}

func (builder *Builder) Offset(offset int) *Builder {
	builder.offset = offset
	return builder
}

func (builder *Builder) Params() []interface{} {
	result := []interface{}{}
	for _, val := range builder.values {
		result = append(result, val)
	}

	return result
}

func (builder *Builder) ForQuery() string {
	builder.values = []interface{}{}
	columns := builder.QuotedColumns()
	sql := "SELECT " + str.Join(columns, ", ") + " FROM " + builder.QuotedTableName()
	if len(builder.conditions) > 0 {
		sql += " WHERE " + builder.handleWhere()
	}
	if len(builder.orders) > 0 {
		sql += handleOrderBy(builder.orders)
	}
	if builder.group != "" {
		sql += " " + builder.group
	}
	if builder.limit > -1 {
		sql += " LIMIT " + builder.modifier.PrePh()
		builder.values = append(builder.values, builder.limit)
	}
	if builder.offset > -1 {
		sql += " OFFSET " + builder.modifier.PrePh()
		builder.values = append(builder.values, builder.offset)
	}
	if builder.replace {
		sql = replace(builder.modifier, sql)
	}
	return builder.toSql(sql)
}

func (builder *Builder) QuotedTableName() string {
	return builder.modifier.QuoteName(builder.tableName)
}

func (builder *Builder) QuotedColumns() []string {
	result := make([]string, len(builder.selects))
	for i, item := range builder.selects {
		fieldName := item.fieldName
		if item.quote {
			fieldName = builder.modifier.QuoteName(item.fieldName)
		}
		result[i] = fieldName
	}

	return result
}

func (builder *Builder) ForRemove() string {
	builder.values = []interface{}{}
	sql := "DELETE FROM " + builder.QuotedTableName()
	if len(builder.conditions) > 0 {
		sql += " WHERE " + builder.handleWhere()
	}
	if builder.limit > -1 {
		sql += " LIMIT " + builder.modifier.PrePh()
		builder.values = append(builder.values, builder.limit)
	}
	if builder.offset > -1 {
		sql += " OFFSET " + builder.modifier.PrePh()
		builder.values = append(builder.values, builder.offset)
	}

	return builder.toSql(replace(builder.modifier, sql))
}

func (builder *Builder) ForInsert(data []map[string]interface{}) string {
	sql := "INSERT INTO " + builder.QuotedTableName()
	fields := []string{}
	values := []string{}
	builder.values = []interface{}{}
	for i, row := range data {
		rowValue := []string{}
		for field, value := range row {
			if i == 0 {
				fields = append(fields, builder.modifier.QuoteName(field))
			}
			rowValue = append(rowValue, builder.modifier.PrePh())
			builder.values = append(builder.values, value)
		}
		values = append(values, "("+str.Join(rowValue, ", ")+")")
	}
	sql += "(" + str.Join(fields, ", ") + ")"
	sql += " VALUES" + str.Join(values, ", ")

	return builder.toSql(replace(builder.modifier, sql))
}

func (builder *Builder) ForUpdate(data map[string]interface{}) string {
	builder.values = []interface{}{}
	sql := "UPDATE " + builder.QuotedTableName() + " SET "
	length := len(data)
	i := 1
	for field, value := range data {
		sql += builder.modifier.QuoteName(field) + "=" + builder.modifier.PrePh()
		builder.values = append(builder.values, value)
		if i < length {
			sql += ", "
		}
		i++
	}
	if len(builder.conditions) > 0 {
		sql += " WHERE " + builder.handleWhere()
	}
	if builder.limit > -1 {
		sql += " LIMIT " + builder.modifier.PrePh()
		builder.values = append(builder.values, builder.limit)
	}
	if builder.offset > -1 {
		sql += " OFFSET " + builder.modifier.PrePh()
		builder.values = append(builder.values, builder.offset)
	}

	return builder.toSql(replace(builder.modifier, sql))
}

func (builder *Builder) toSql(sql string) string {
	if !builder.isPrepare {
		return sql
	}
	for _, val := range builder.Params() {
		sql = str.Replace(sql, builder.modifier.Ph(""), val.(string), 1)
	}
	return sql
}

func handleOrderBy(orders map[string]string) string {
	sql := " ORDER BY "
	length := len(orders)
	i := 1
	for field, order := range orders {
		sql += field + " " + order
		if i < length {
			sql += ", "
		}
		i++
	}

	return sql
}
