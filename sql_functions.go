package pgexpect

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
)

// MockFunction replace a function and expect calls to it
func MockFunction(function Function, expectedCalls []Call, t *testing.T, db *sqlx.DB, testFunc func(t *testing.T, db *sqlx.DB)) {
	body := createCallTrackingTable(function.Name, function.Args, t, db)
	function.Body = body + function.Body // This is not especially nice, but this at least don't overwrite the body someone might pass.

	createSQLFunctions(function, t, db)
	testFunc(t, db)
	calls := getCalls(function.Name, t, db)
	validateCalls(calls, expectedCalls, function.Args, t)
}

// StubFunction replace a function with a fake version of it
func StubFunction(function Function, t *testing.T, db *sqlx.DB, testFunc func()) {
	createSQLFunctions(function, t, db)
	testFunc()
}

func createCallTrackingTable(functionName string, args []Argument, t *testing.T, db *sqlx.DB) (body string) {
	var tableColumns []string
	var argNames []string
	for _, arg := range args {
		argNames = append(argNames, arg.Name)
		tableColumns = append(tableColumns, fmt.Sprintf("%s %s", arg.Name, arg.Type))
	}

	tableColumnsReady := strings.Join(tableColumns, ",")
	argNamesReady := strings.Join(argNames, ",")

	q := fmt.Sprintf(`
		DROP TABLE IF EXISTS %s_calls;
		CREATE TABLE %s_calls (%s);
	`, functionName, functionName, tableColumnsReady)

	_, err := db.Exec(q)
	if err != nil {
		t.Fatalf(`[pgexpect] Error while creating the call tracking SQL function %s_calls: %s`, functionName, err)
	}

	body = fmt.Sprintf(`INSERT INTO %s_calls (%s) VALUES(%s);`, functionName, argNamesReady, argNamesReady)
	return
}

func createSQLFunctions(function Function, t *testing.T, db *sqlx.DB) {
	var argsWithTypes []string
	for _, arg := range function.Args {
		argsWithTypes = append(argsWithTypes, fmt.Sprintf("%s %s", arg.Name, arg.Type))
	}
	argsReadyWithTypes := strings.Join(argsWithTypes, ",")

	if function.RaiseErrorCode != "" {
		raise := fmt.Sprintf(`RAISE 'error from pgexpect' USING ERRCODE = '%s';`, function.RaiseErrorCode)
		function.Body = function.Body + raise
	}

	q := fmt.Sprintf(`
		CREATE OR REPLACE FUNCTION %s(%s) RETURNS %s
			AS $$
			BEGIN
				%s
			END
		$$ LANGUAGE plpgsql;
	`, function.Name, argsReadyWithTypes, function.ReturnType, function.Body)

	_, err := db.Exec(q)
	if err != nil {
		t.Fatalf("[pgexpect] Error while creating the fake SQL function %s: %s", function.Name, err)
	}
}

func getCalls(functionName string, t *testing.T, db *sqlx.DB) []map[string]interface{} {
	calls := []map[string]interface{}{}
	q := fmt.Sprintf(`SELECT * FROM %s_calls`, functionName)
	rows, err := db.Queryx(q)
	if err != nil {
		t.Fatalf("[pgexpect] Error getting calls: %s", err)
	}
	for rows.Next() {
		results := make(map[string]interface{})
		err = rows.MapScan(results)
		if err != nil {
			t.Fatalf("[pgexpect] Problem with scanning of results to call tracking: %s", err)
		} else {
			calls = append(calls, results)
		}
	}

	if len(calls) != 1 {
		t.Fatalf("[pgexpect] We expected one call but got %v", len(calls))
	}

	return calls
}

func validateCalls(calls []map[string]interface{}, expectedCalls []Call, args []Argument, t *testing.T) {
	// Compare the results. sqlx does not play super nice with types with `MapScan`,
	// we gotta do something not super nice. There might be a better way, but I
	// don't think it will ever be very nice. There is def room for improvement.
	for callIndex, call := range calls {
		for argIndex, arg := range args {
			switch expected := expectedCalls[callIndex].Values[argIndex].(type) {
			case time.Time:
				got := call[arg.Name].(time.Time)
				if expected != got {
					t.Errorf("[pgexpect] Wrong value for %s. Expected %s but got %s", arg.Name, expected, got)
				}
			case bool:
				got := call[arg.Name].(bool)
				if expected != got {
					t.Errorf("[pgexpect] Wrong value for %s. Expected %s but got %s", arg.Name, expected, got)
				}
			case string:
				got := getString(call[arg.Name], arg.Name, t)
				if expected != got {
					t.Errorf("[pgexpect] Wrong value for %s. Expected %s but got %s", arg.Name, expected, got)
				}
			case uint64:
				got := uint64(call[arg.Name].(int64))
				if expected != got {
					t.Errorf("[pgexpect] Wrong value for %s. Expected %v but got %v", arg.Name, expected, got)
				}
			case int64:
				got := call[arg.Name].(int64)
				if expected != got {
					t.Errorf("[pgexpect] Wrong value for %s. Expected %v but got %v", arg.Name, expected, got)
				}
			case int32:
				got := call[arg.Name].(int32)
				if expected != got {
					t.Errorf("[pgexpect] Wrong value for %s. Expected %v but got %v", arg.Name, expected, got)
				}
			case int:
				got := int(call[arg.Name].(int64))
				if expected != got {
					t.Errorf("[pgexpect] Wrong value for %s. Expected %v but got %v", arg.Name, expected, got)
				}
			case []uint8:
				got := call[arg.Name].([]uint8)
				if !reflect.DeepEqual(expected, got) {
					t.Errorf("[pgexpect] Wrong value for %s. Expected %v but got %v", arg.Name, string(expected), string(got))
				}
			default:
				t.Fatalf(`[pgexpect] Unknown type for column %s, expected value is %v`, arg.Name, expected)
			}
		}
	}
}

// getString is used to turn []byte/[]uint8 into strings. This happens when using custom types.
func getString(src interface{}, argName string, t *testing.T) string {
	switch v := src.(type) {
	case string:
		return v
	case []uint8:
		return string(v)
	default:
		t.Fatalf(`[pgexpect] Unknown type "%s" for column %s to turn into a string`, v, argName)
		return ""
	}
}
