package lsp

import (
	"database/sql"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"tops-lsp/lsp/data"

	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/protobuf/encoding/prototext"
)

func TestBuildFileIndex(t *testing.T) {
	ctx := &MockLspContext{workspaceDir: GetGoModDir()}
	db := NewDataBase()

	// Dynamically generate the file path
	sourceFilePath := filepath.Join(GetGoModDir(), "test-files", "test.tops")
	cache_dir := filepath.Dir(sourceFilePath)
	config := GetCompileConfig(ctx, sourceFilePath)
	idxFile := GetIndexFileName(config)
	idxFile = filepath.Join(cache_dir, idxFile)

	g_pluginPath = filepath.Join(GetGoModDir(), "libtops-lsp.so")

	builder := newAstBuilder()

	builder.worker = AsyncRun(func(_ *AsyncWorker) {
		ast := builder.buildAst(ctx, config, idxFile)
		if ast == nil {
			return
		}
		db.AddAstCache(config.File, ast)
	})

	builder.worker.Wait()

	t.Log("Indexing completed")
	// Verify that the source file is indexed
	if db.GetAstCache(sourceFilePath) == nil {
		t.Errorf("Failed to index source file: %s", sourceFilePath)
	}

	ast := db.GetAstCache(sourceFilePath).GetAst()

	txtFilePath := filepath.Join(cache_dir, "ast_cache.txt")

	// Save the AST as a readable proto text format
	txtFile, err := os.Create(txtFilePath)
	if err != nil {
		t.Errorf("Failed to create text file: %v", err)
		return
	}
	defer txtFile.Close()
	var bytes []byte
	bytes, err = prototext.MarshalOptions{Indent: "  "}.Marshal(ast)
	if err != nil {
		t.Errorf("Failed to marshal AST to text format: %v", err)
	}

	t.Log("proto Marshal Done")
	_, err = txtFile.Write(bytes)

	t.Log("Write text file done")
	// Save AST to SQLite database
	dbFilePath := filepath.Join(cache_dir, "ast_cache.db")
	os.Remove(dbFilePath) // Remove the file if it exists
	t.Log("Create db file done, saveing to db")
	err = SaveAstToSQLite(dbFilePath, sourceFilePath, ast)
	if err != nil {
		t.Errorf("Failed to save AST to SQLite DB: %v", err)
	}
}

func locationToString(location *data.Location, tu *data.TranslationUnit) string {
	if location == nil {
		return ""
	}
	filename := tu.StringTable[location.FileName.Index]
	return filename + ":" + strconv.Itoa(int(location.Line)) + ":" + strconv.Itoa(int(location.Column)) + " len " + strconv.Itoa(int(location.Length))
}

// SaveAstToSQLite saves the AST to a SQLite database file.
func SaveAstToSQLite(dbFilePath, sourceFilePath string, ast *data.TranslationUnit) error {
	// Open or create the SQLite database
	db, err := sql.Open("sqlite3", dbFilePath)
	if err != nil {
		return err
	}
	defer db.Close()
	// Create tables if they don't exist
	createTablesQuery := `
	CREATE TABLE IF NOT EXISTS TranslationUnit (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		source_file TEXT,
		compile_args TEXT
	);
	CREATE TABLE IF NOT EXISTS decl_refs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		referenced_name TEXT,
		ref_type TEXT,
		obj_id INTEGER,
		location TEXT
	);
	CREATE TABLE IF NOT EXISTS variable_table (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		var_name TEXT,
		attr TEXT,
		type TEXT,
		location TEXT
	);
	CREATE TABLE IF NOT EXISTS function_table (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		func_name TEXT,
		return_type TEXT,
		location TEXT,
		vars TEXT,
		params TEXT
	);
	-- 创建关联表
	CREATE TABLE function_vars (
 	 	func_id INTEGER REFERENCES function_table(id),  -- 外键指向functions表
 		var_id INTEGER REFERENCES variable_table(id),      -- 外键指向vars表
  		PRIMARY KEY (func_id, var_id)             -- 联合主键
	);
	CREATE TABLE IF NOT EXISTS included_headers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		header_file TEXT
	);`
	_, err = db.Exec(createTablesQuery)
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	insertAstQuery := `
	INSERT INTO TranslationUnit (source_file, compile_args)
	VALUES (?, ?);`
	_, err = tx.Exec(insertAstQuery, sourceFilePath, ast.CompileArgs)
	if err != nil {
		return err
	}

	// Save decl_refs to decl_refs table
	for _, declRef := range ast.DeclRefs {
		locationData := locationToString(declRef.Location, ast)
		var obj_id uint32 = 0
		if declRef.GetRefType() == data.DeclRef_VARIABLE || declRef.GetRefType() == data.DeclRef_PARAMETER {
			obj_id = declRef.GetVariable()
		} else if declRef.GetRefType() == data.DeclRef_FUNCTION {
			obj_id = declRef.GetFunction()
		}
		insertDeclRefQuery := `
		INSERT INTO decl_refs (referenced_name, ref_type, obj_id, location)
		VALUES (?, ?, ?, ?);`
		_, err = tx.Exec(insertDeclRefQuery, declRef.ReferencedName, declRef.RefType.String(), obj_id, string(locationData))
		if err != nil {
			return err
		}
	}

	// Save variable_table to variable_table table
	for _, variableEntry := range ast.VariableTable {
		locationData := locationToString(variableEntry.Location, ast)
		insertVariableQuery := `
		INSERT INTO variable_table (var_name, attr, type, location)
		VALUES (?, ?, ?, ?);`
		_, err = tx.Exec(insertVariableQuery, variableEntry.Name, variableEntry.VarType.String(), variableEntry.Type, string(locationData))
		if err != nil {
			return err
		}
	}

	// Save function_table to function_table table
	for func_id, functionEntry := range ast.FunctionTable {
		locationData := locationToString(functionEntry.Location, ast)
		var vars, params string

		var localVars []string
		for _, localVar := range functionEntry.GetLocalVars() {
			localVars = append(localVars, strconv.Itoa(int(localVar)))
		}
		var localParams []string
		for _, localParam := range functionEntry.GetParameters() {
			localParams = append(localParams, strconv.Itoa(int(localParam)))
		}
		vars = strings.Join(localVars, " ")
		params = strings.Join(localParams, " ")
		insertFunctionQuery := `
		INSERT INTO function_table (func_name, return_type, location, vars, params)
		VALUES (?, ?, ?, ?, ?);`
		_, err = tx.Exec(insertFunctionQuery, functionEntry.Name, functionEntry.ReturnType, string(locationData), vars, params)
		if err != nil {
			return err
		}
		// Save the relationship between functions and variables
		for _, localVar := range functionEntry.GetLocalVars() {
			insertFunctionVarsQuery := `
			INSERT INTO function_vars (func_id, var_id)
			VALUES (?, ?);`
			_, err = tx.Exec(insertFunctionVarsQuery, func_id, localVar)
			if err != nil {
				return err
			}
		}
		// Save the relationship between functions and parameters
		for _, localParam := range functionEntry.GetParameters() {
			insertFunctionVarsQuery := `
			INSERT INTO function_vars (func_id, var_id)
			VALUES (?, ?);`
			_, err = tx.Exec(insertFunctionVarsQuery, func_id, localParam)
			if err != nil {
				return err
			}
		}
	}

	// Save included_headers to included_headers table
	for _, header := range ast.IncludedHeaders {
		headerFile := ast.StringTable[header.FileName.Index]
		insertHeaderQuery := `
		INSERT INTO included_headers (header_file)
		VALUES (?);`
		_, err = tx.Exec(insertHeaderQuery, sourceFilePath, headerFile)
		if err != nil {
			return err
		}
	}
	tx.Commit()

	return nil
}
