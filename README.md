# pgexpect

Test mocks & stubs made easy for Postgres functions.

You can verify that a function was called with the expected parameters. You can also replace the body of a function or make it raise a SQL exception.

## How to mock?

Example:

```go
db := GetTestDB()
defer db.Close()

function := pgexpect.Function{
  Name: "some_function_name",
  Args: []pgexpect.Argument{
    {Name: "param1", Type: "integer"},
    {Name: "param2", Type: "text"},
  },
  ReturnType: "SETOF stuff",
  Body: `INSERT INTO ...` // You can add some SQL that the mock should run.
}

expectedCalls := []pgexpect.Call{
  {Values: []interface{}{uint64(123), "Gemnasium is awesome! ;)"}},
}

pgexpect.MockFunction(function, expectedCalls, t, db, func(t *testing.T, db *sqlx.DB) {
  service := SomeDBService{DB: db}
  err := service.CreateStuff(123, "Gemnasium is awesome! ;)")
  if err != nil {
    t.Fatal(err)
  }
})
```

## How to stub?

Example:

```go
db := GetTestDB()
defer db.Close()

function := pgexpect.Function{
  Name: "your_super_smart_function",
  Args: []pgexpect.Argument{
    {Name: "name", Type: "text"},
  },
  ReturnType:     "VOID",
  RaiseErrorCode: "GM007", // in case you need to simulate an error in your SQL function
}

pgexpect.StubFunction(function, t, db, func() {
  service := UserDBService{DB: db}
  _, err := service.Create("john@example.com")
  if err == nil || err != ErrUserEmailAlreadyExists {
    t.Errorf("An ErrUserEmailAlreadyExists error was expected but we got %s", err)
  }
})
```