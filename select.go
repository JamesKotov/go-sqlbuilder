// Copyright 2018 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

package sqlbuilder

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// JoinOption is the option in JOIN.
type JoinOption string

// Join options.
const (
	LeftJoin       JoinOption = "LEFT"
	AllLeftJoin    JoinOption = "ALL LEFT"
	LeftOuterJoin  JoinOption = "LEFT OUTER"
	RightJoin      JoinOption = "RIGHT"
	AllRightJoin   JoinOption = "ALL RIGHT"
	RightOuterJoin JoinOption = "RIGHT OUTER"
	FullJoin       JoinOption = "FULL"
	FullOuterJoin  JoinOption = "FULL OUTER"
	AllInnerJoin   JoinOption = "ALL INNER"
	ArrayJoin      JoinOption = "ARRAY"
)

// NewSelectBuilder creates a new SELECT builder.
func NewSelectBuilder() *SelectBuilder {
	return DefaultFlavor.NewSelectBuilder()
}

func newSelectBuilder() *SelectBuilder {
	args := &Args{}
	return &SelectBuilder{
		Cond: Cond{
			Args: args,
		},
		limit:  -1,
		offset: -1,
		args:   args,
	}
}

// SelectBuilder is a builder to build SELECT.
type SelectBuilder struct {
	Cond

	distinct          bool
	tables            []string
	withCols          []string
	selectCols        []string
	joinOptions       []JoinOption
	joinTables        []string
	joinExprsOperator []string
	joinExprs         [][]string
	whereExprs        []string
	havingExprs       []string
	groupByCols       []string
	orderByCols       []string
	order             string
	limit             int
	offset            int
	limitVar          string
	offsetVar         string

	args *Args
}

// Distinct marks this SELECT as DISTINCT.
func (sb *SelectBuilder) Distinct() *SelectBuilder {
	sb.distinct = true
	return sb
}

// Select sets columns in WITH.
func (sb *SelectBuilder) With(col ...string) *SelectBuilder {
	sb.withCols = col
	return sb
}

// Select sets columns in SELECT.
func (sb *SelectBuilder) Select(col ...string) *SelectBuilder {
	sb.selectCols = col
	return sb
}

// From sets table names in SELECT.
func (sb *SelectBuilder) From(table ...string) *SelectBuilder {
	sb.tables = table
	return sb
}

// Join sets expressions of JOIN in SELECT.
//
// It builds a JOIN expression like
//     JOIN table ON onExpr[0] AND onExpr[1] ...
func (sb *SelectBuilder) Join(table string, onExpr ...string) *SelectBuilder {
	return sb.joinWithOption("", table, "ON", onExpr...)
}

// JoinUsing sets expressions of JOIN in SELECT with USING operator (for Clickhouse)
//
// It builds a JOIN expression like
//     JOIN table USING col1, col2 ...
func (sb *SelectBuilder) JoinUsing(table string, onExpr ...string) *SelectBuilder {
	return sb.joinWithOption("", table, "USING", onExpr...)
}

// JoinWithOption sets expressions of JOIN with an option.
//
// It builds a JOIN expression like
//     option JOIN table ON onExpr[0] AND onExpr[1] ...
//
// Here is a list of supported options.
//     - LeftJoin: LEFT JOIN
//     - LeftOuterJoin: LEFT OUTER JOIN
//     - RightJoin: RIGHT JOIN
//     - RightOuterJoin: RIGHT OUTER JOIN
func (sb *SelectBuilder) JoinWithOption(option JoinOption, table string, onExpr ...string) *SelectBuilder {
	return sb.joinWithOption(option, table, "ON", onExpr...)
}

// JoinWithOptionUsing sets expressions of JOIN with an option and with USING operator (for Clickhouse)
//
// It builds a JOIN expression like
//     option JOIN table USING col1, col2 ...
//
// Here is a list of supported options.
//     - LeftJoin: LEFT JOIN
//     - LeftOuterJoin: LEFT OUTER JOIN
//     - RightJoin: RIGHT JOIN
//     - RightOuterJoin: RIGHT OUTER JOIN
func (sb *SelectBuilder) JoinWithOptionUsing(option JoinOption, table string, onExpr ...string) *SelectBuilder {
	return sb.joinWithOption(option, table, "USING", onExpr...)
}

// joinWithOption sets expressions of JOIN with an option and conditions operator
//
// It builds a JOIN expression like
//     option JOIN table ON onExpr[0] AND onExpr[1] ...
//     or
//     option JOIN table USING col1, col2 ...
//
// Here is a list of supported options.
//     - LeftJoin: LEFT JOIN
//     - LeftOuterJoin: LEFT OUTER JOIN
//     - RightJoin: RIGHT JOIN
//     - RightOuterJoin: RIGHT OUTER JOIN
func (sb *SelectBuilder) joinWithOption(option JoinOption, table string, operator string, onExpr ...string) *SelectBuilder {
	sb.joinOptions = append(sb.joinOptions, option)
	sb.joinTables = append(sb.joinTables, table)
	sb.joinExprsOperator = append(sb.joinExprsOperator, operator)
	sb.joinExprs = append(sb.joinExprs, onExpr)
	return sb
}

// Where sets expressions of WHERE in SELECT.
func (sb *SelectBuilder) Where(andExpr ...string) *SelectBuilder {
	sb.whereExprs = append(sb.whereExprs, andExpr...)
	return sb
}

// Having sets expressions of HAVING in SELECT.
func (sb *SelectBuilder) Having(andExpr ...string) *SelectBuilder {
	sb.havingExprs = append(sb.havingExprs, andExpr...)
	return sb
}

// GroupBy sets columns of GROUP BY in SELECT.
func (sb *SelectBuilder) GroupBy(col ...string) *SelectBuilder {
	sb.groupByCols = col
	return sb
}

// OrderBy sets columns of ORDER BY in SELECT.
func (sb *SelectBuilder) OrderBy(col ...string) *SelectBuilder {
	sb.orderByCols = col
	return sb
}

// Asc sets order of ORDER BY to ASC.
func (sb *SelectBuilder) Asc() *SelectBuilder {
	sb.order = "ASC"
	return sb
}

// Desc sets order of ORDER BY to DESC.
func (sb *SelectBuilder) Desc() *SelectBuilder {
	sb.order = "DESC"
	return sb
}

// Limit sets the LIMIT in SELECT.
func (sb *SelectBuilder) Limit(limit int) *SelectBuilder {
	sb.limit = limit
	return sb
}

// Offset sets the LIMIT offset in SELECT.
func (sb *SelectBuilder) Offset(offset int) *SelectBuilder {
	sb.offset = offset
	return sb
}

// Limit sets the LIMIT in SELECT.
func (sb *SelectBuilder) LimitVar(limit string) *SelectBuilder {
	sb.limitVar = limit
	return sb
}

// Offset sets the LIMIT offset in SELECT.
func (sb *SelectBuilder) OffsetVar(offset string) *SelectBuilder {
	sb.offsetVar = offset
	return sb
}

// As returns an AS expression.
func (sb *SelectBuilder) As(name, alias string) string {
	return fmt.Sprintf("%s AS %s", name, alias)
}

// BuilderAs returns an AS expression wrapping a complex SQL.
// According to SQL syntax, SQL built by builder is surrounded by parens.
func (sb *SelectBuilder) BuilderAs(builder Builder, alias string) string {
	return fmt.Sprintf("(%s) AS %s", sb.Var(builder), alias)
}

// String returns the compiled SELECT string.
func (sb *SelectBuilder) String() string {
	s, _ := sb.Build()
	return s
}

// Build returns compiled SELECT string and args.
// They can be used in `DB#Query` of package `database/sql` directly.
func (sb *SelectBuilder) Build() (sql string, args []interface{}) {
	return sb.BuildWithFlavor(sb.args.Flavor)
}

// BuildWithFlavor returns compiled SELECT string and args with flavor and initial args.
// They can be used in `DB#Query` of package `database/sql` directly.
func (sb *SelectBuilder) BuildWithFlavor(flavor Flavor, initialArg ...interface{}) (sql string, args []interface{}) {
	buf := &bytes.Buffer{}

	if len(sb.withCols) > 0 {
		buf.WriteString("WITH (")
		buf.WriteString(strings.Join(sb.withCols, ", "))
		buf.WriteString(") ")
	}

	buf.WriteString("SELECT ")

	if sb.distinct {
		buf.WriteString("DISTINCT ")
	}

	buf.WriteString(strings.Join(sb.selectCols, ", "))

	if len(sb.tables) > 0 {
		buf.WriteString(" FROM ")
		buf.WriteString(strings.Join(sb.tables, ", "))
	}

	for i := range sb.joinTables {
		if option := sb.joinOptions[i]; option != "" {
			buf.WriteRune(' ')
			buf.WriteString(string(option))
		}

		buf.WriteString(" JOIN ")
		buf.WriteString(sb.joinTables[i])

		if exprs := sb.joinExprs[i]; len(exprs) > 0 {
			operator := sb.joinExprsOperator[i]

			sep := " AND "
			if operator == "USING" {
				sep = ", "
			}

			buf.WriteRune(' ')
			buf.WriteString(operator)
			buf.WriteRune(' ')
			buf.WriteString(strings.Join(sb.joinExprs[i], sep))
		}
	}

	if len(sb.whereExprs) > 0 {
		buf.WriteString(" WHERE ")
		buf.WriteString(strings.Join(sb.whereExprs, " AND "))
	}

	if len(sb.groupByCols) > 0 {
		buf.WriteString(" GROUP BY ")
		buf.WriteString(strings.Join(sb.groupByCols, ", "))

		if len(sb.havingExprs) > 0 {
			buf.WriteString(" HAVING ")
			buf.WriteString(strings.Join(sb.havingExprs, " AND "))
		}
	}

	if len(sb.orderByCols) > 0 {
		buf.WriteString(" ORDER BY ")
		buf.WriteString(strings.Join(sb.orderByCols, ", "))

		if sb.order != "" {
			buf.WriteRune(' ')
			buf.WriteString(sb.order)
		}
	}

	if sb.limitVar != "" {
		buf.WriteString(" LIMIT ")
		buf.WriteString(sb.limitVar)
		if sb.offsetVar != "" {
			buf.WriteString(" OFFSET ")
			buf.WriteString(sb.offsetVar)
		}
	} else {

		if sb.limit >= 0 {
			buf.WriteString(" LIMIT ")
			buf.WriteString(strconv.Itoa(sb.limit))
		}
		if MySQL == flavor && sb.limit >= 0 || PostgreSQL == flavor {
			if sb.offset >= 0 {
				buf.WriteString(" OFFSET ")
				buf.WriteString(strconv.Itoa(sb.offset))
			}
		}
	}

	return sb.Args.CompileWithFlavor(buf.String(), flavor, initialArg...)
}

// SetFlavor sets the flavor of compiled sql.
func (sb *SelectBuilder) SetFlavor(flavor Flavor) (old Flavor) {
	old = sb.args.Flavor
	sb.args.Flavor = flavor
	return
}
