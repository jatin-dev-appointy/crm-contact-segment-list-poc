package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/lib/pq"
	"log"
	"strings"

	_ "github.com/lib/pq"
)

type QueryValue interface{}

type QueryRule struct {
	Field      string       `json:"field,omitempty"`
	Operator   string       `json:"operator,omitempty"`
	Value      QueryValue   `json:"value,omitempty"`
	Combinator string       `json:"combinator,omitempty"`
	Rules      []*QueryRule `json:"rules,omitempty"`
}

type DbRule struct {
	Id   string          `json:"id"`
	Name string          `json:"name"`
	Rule json.RawMessage `json:"rule"`
}

func main() {
	fmt.Println()
	fmt.Println("POC on Contact Query Rules")
	fmt.Println("")
	fmt.Println("Enter Values")
	var parent, dbConnStr string
	fmt.Print("CompanyID (grp_/cmp_): ")
	fmt.Scanln(&parent)
	fmt.Print("Database Connection String: ")
	fmt.Scanln(&dbConnStr)

	if parent == "" {
		log.Fatalf("CompanyID is required")
	}
	if dbConnStr == "" {
		log.Fatalf("Database Connection String is required")
	}

	db, err := sql.Open("postgres", dbConnStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer func() {
		if err = db.Close(); err != nil {
			log.Fatalf("Failed to close database connection: %v", err)
		}
	}()

	ruleRows, err := db.Query(`select * from demo.segment_rule`)
	if err != nil {
		log.Fatalf("Failed to execute query: %v", err)
	}
	defer func() {
		if err = ruleRows.Close(); err != nil {
			log.Fatalf("Failed to close rows: %v", err)
		}
	}()

	var dbRules []*DbRule
	for ruleRows.Next() {
		var id, name string
		var rule []byte
		if err = ruleRows.Scan(&id, &name, &rule); err != nil {
			log.Fatalf("Failed to scan row: %v", err)
		}
		dbRules = append(dbRules, &DbRule{
			Id:   id,
			Name: name,
			Rule: rule,
		})
	}

	fmt.Println("Starting Now...")
	for _, dbRule := range dbRules {
		fmt.Println()
		fmt.Println("-> For Rule: ", dbRule.Name)
		var rule QueryRule
		err = json.Unmarshal(dbRule.Rule, &rule)

		args := []interface{}{parent, false}
		var query = `SELECT id, first_name, last_name, email, phone_number FROM saastack_customer_v1.customer WHERE parent = $1 AND is_deleted = $2 `

		ruleCondition, err := buildWhereClauseNested(&rule, &args)
		if err != nil {
			log.Fatalf("Failed to build where clause: %v", err)
		}
		query += " AND " + ruleCondition

		fmt.Println("Query: ", query)
		fmt.Println("Args: ", args)

		rows, err := db.QueryContext(context.Background(), query, args...)
		if err != nil {
			log.Fatalf("Failed to execute query: %v", err)
		}
		defer func() {
			if err = rows.Close(); err != nil {
				log.Fatalf("Failed to close rows: %v", err)
			}
		}()

		fmt.Println("Results:")
		for rows.Next() {
			var id, firstName, lastName, email, phone string
			if err = rows.Scan(&id, &firstName, &lastName, &email, &phone); err != nil {
				log.Fatalf("Failed to scan row: %v", err)
			}
			fmt.Printf("- (%s, %s %s, %s, %s) \n", id, firstName, lastName, email, phone)
		}
	}

	fmt.Println("Done...")
}

func buildWhereClauseNested(queryRule *QueryRule, args *[]interface{}) (string, error) {

	conditions := make([]string, 0, len(queryRule.Rules)+1)

	baseQuery, err := buildWhereClauseBase(queryRule, args)
	if err != nil {
		return "", err
	}
	if baseQuery != "" {
		conditions = append(conditions, baseQuery)
	}

	for _, rule := range queryRule.Rules {
		var nestedQuery string
		nestedQuery, err = buildWhereClauseNested(rule, args)
		if err != nil {
			return "", err
		}
		if nestedQuery != "" {
			conditions = append(conditions, nestedQuery)
		}
	}

	return fmt.Sprintf("( %s )", strings.Join(conditions, fmt.Sprintf(" %s ", queryRule.Combinator))), nil
}

func buildWhereClauseBase(queryRule *QueryRule, args *[]interface{}) (string, error) {
	if queryRule.Field == "" && queryRule.Operator == "" && queryRule.Value == nil {
		return "", nil
	}

	if queryRule.Field == "" || queryRule.Operator == "" || queryRule.Value == nil {
		return "", fmt.Errorf("missing field, operator or value to build base condition")
	}

	baseQuery := ""
	switch queryRule.Value.(type) {
	case string:
		if strings.ToLower(queryRule.Operator) == "in" {
			return "", fmt.Errorf("invalid operator %s for string value", queryRule.Operator)
		}

		*args = append(*args, queryRule.Value.(string))
		baseQuery = fmt.Sprintf("%s %s $%d", queryRule.Field, queryRule.Operator, len(*args))

	case []string:
		if strings.ToLower(queryRule.Operator) != "in" {
			return "", fmt.Errorf("invalid operator %s for []string value", queryRule.Operator)
		}
		queryRule.Operator = "="

		*args = append(*args, pq.Array(queryRule.Value.([]string)))
		baseQuery = fmt.Sprintf("%s %s ANY($%d)", queryRule.Field, queryRule.Operator, len(*args))

	case []interface{}:
		if strings.ToLower(queryRule.Operator) != "in" {
			return "", fmt.Errorf("invalid operator %s for []interface value", queryRule.Operator)
		}
		queryRule.Operator = "="

		strSlice := make([]string, 0, len(queryRule.Value.([]interface{})))
		for _, v := range queryRule.Value.([]interface{}) {
			strSlice = append(strSlice, v.(string))
		}

		*args = append(*args, pq.Array(strSlice))
		baseQuery = fmt.Sprintf("%s %s ANY($%d)", queryRule.Field, queryRule.Operator, len(*args))

	default:
		return "", fmt.Errorf("unsupported value type: %T, allowed types are string, []string and []interface{}", queryRule.Value)
	}

	return baseQuery, nil
}
