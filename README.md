# Show Sql

# Description
showsql is a utility that analyzes Go code, extracts SQL queries built with github.com/Masterminds/squirrel, and prints them to stdout.

# Installation
```
go install github.com/mrsubudei/showsql@latest
```

# Usage
The utility must be run from the project's root directory.
```
showsql [OPTION]...
```

After launch, a navigation interface opens:

1. Use the ↑ / ↓ keys to move through directories.
2. Press Enter to select the file containing the queries.
3. Select the method name.

Alternatively, run it with the -p={path to file} and -m={method name} flags:
```
showsql -p=internal/repo/users/store.go -m=GetByID [OPTION]...
```

# How it works
The utility copies the files containing the queries into a temporary directory, `cmd/showsql_tmp`, generates a `main.go` entry point, runs it, and then deletes the temporary directory. To keep the temporary directory, run with the `-k` flag.
_______________________________________________________________________________________________
Only methods are supported; plain functions are not processed.
_______________________________________________________________________________________________
The fields of the repository struct are not populated, so if the code calls a method on a struct field (for example, s.someField.SomeMethod) before the database call, the utility will panic due to a nil dereference.
_______________________________________________________________________________________________

By default, the utility looks for the following database calls: `Get`, `Getx`, `Select`, `Selectx`, `Exec`, `Execx`, `Query`, `Queryx`, `QueryRow`, `QueryxRow`, and inserts early `return` statements at those points, since no database connection is initialized. If your code uses other method names, you can list them in the `showsql.yaml` config file. By default, the file is expected at `scripts/showsql.yaml`. You can specify a different path with the `-f={path to file}` flag.

Example config file:
```
exec:
- ExecContext
- QueryContext
- QueryRowContext
get:
- GetContext
- SelectContext
execx:
- ExecxContext
- QueryxContext
- QueryxRowContext
getx:
- GetxContext
- SelectxContext
```

The `exec` group is for methods with the arguments `(ctx context.Context, query string, args ...interface{})`
The `get` group is for methods with the arguments `(ctx context.Context, dest interface{}, query string, args ...interface{})`
The `execx` group is for methods with the arguments `(ctx context.Context, sq Sqlizer)`
The `getx` group is for methods with the arguments `(ctx context.Context, dest interface{}, sqlizer Sqlizer)`
_______________________________________________________________________________________________
If a method contains two or more database calls, only the first query will be printed.
_______________________________________________________________________________________________

Before the method containing the query is called, all input arguments are filled with values. Composite types, including structs, are also filled recursively down to primitive types. Strings are filled with `str1`, `str2`, `str3`, and so on; numbers with prime numbers starting from 7, 11, 13, and so on; booleans are always `true`; time.Time values start at `2025-01-01` and increase by 1 day each. To leave default (zero) values instead, use these flags: `-i` sets all numbers to 0, `-s` sets all strings to empty, `-b` sets all booleans to `false`, `-t` sets all time.Time values to `0001-01-01`.

By default, slices, arrays, and maps are filled with only one element. To increase the number of elements, use the `-n={number}` flag (maximum 10).
_______________________________________________________________________________________________

If an interface other than `context.Context` is passed as a method argument, that argument is not filled and `nil` is passed instead.

# Options
-v, version
-h, help
-p=string, path to the file
-m=string, method name
-k, do not delete the temporary directory
-i, fill all numeric arguments with the default value
-s, fill all string arguments with the default value
-b, fill all boolean arguments with the default value
-t, fill all time.Time arguments with the default value
-n=int, number of elements in filled slices, arrays, and maps (maximum 10)
-f=string, path to the config file listing the database call methods
