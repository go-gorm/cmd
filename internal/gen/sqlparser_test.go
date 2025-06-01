package gen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

var methodExpectedLines = map[string][]string{
	"GetByID": {
		"var sb strings.Builder",
		"params := make([]any, 0, 2)",
		`sb.WriteString("SELECT * FROM ? WHERE id=? AND name = \\\"@name\\\"")`,
		"params = append(params, clause.CurrentTable, id)",
	},
	"FilterWithColumn": {
		"var sb strings.Builder",
		"params := make([]any, 0, 4)",
		`sb.WriteString("SELECT * FROM ? WHERE ?=?")`,
		`params = append(params, clause.CurrentTable, gorm.Expr("?", column), value)`,
	},
	"QueryWith": {
		"var sb strings.Builder",
		"params := make([]any, 0, 2)",
		`sb.WriteString("SELECT * FROM users")`,
		"if user.ID > 0 {",
		`sb.WriteString("WHERE id=?")`,
		"params = append(params, user.ID)",
		"} else if user.Name != \"\" {",
		`sb.WriteString("WHERE username=?")`,
		"params = append(params, user.Name)",
		"}",
	},
	"UpdateInfo": {
		"var sb strings.Builder",
		"params := make([]any, 0, 4)",
		`sb.WriteString("UPDATE ?")`,
		"params = append(params, clause.CurrentTable)",
		"{",
		"var tmp strings.Builder",
		"if user.Name != \"\" {",
		`tmp.WriteString("username=?,")`,
		"params = append(params, user.Name)",
		"}",
		"if user.Age > 0 {",
		`tmp.WriteString("age=?,")`,
		"params = append(params, user.Age)",
		"}",
		"if user.Age >= 18 {",
		`tmp.WriteString("is_adult=1")`,
		"} else {",
		`tmp.WriteString("is_adult=0")`,
		"}",
		"c := strings.TrimSpace(tmp.String())",
		"if c != \"\" {",
		"if strings.HasSuffix(c, \",\") {",
		"c = strings.TrimSpace(strings.TrimRight(c, \",\"))",
		"}",
		`sb.WriteString("SET ")`,
		"sb.WriteString(c)",
		"}",
		"}",
		`sb.WriteString("WHERE id=?")`,
		"params = append(params, id)",
	},
	"Filter": {
		"var sb strings.Builder",
		"params := make([]any, 0, 13)",
		`sb.WriteString("SELECT * FROM ?")`,
		"params = append(params, clause.CurrentTable)",
		"{",
		"var tmp strings.Builder",
		"for _, user := range users {",
		"if user.Name != \"\" && user.Age > 0 {",
		`tmp.WriteString("(username = ? AND age=? AND role LIKE concat(\\\"%\\\",?,\\\"%\\\")) OR")`,
		"params = append(params, user.Name, user.Age, user.Role)",
		"}",
		"}",
		"c := strings.TrimSpace(tmp.String())",
		"if c != \"\" {",
		`sb.WriteString("WHERE ")`,
		"if len(c) >= 3 && strings.EqualFold(c[len(c)-3:], \"AND\") {",
		"c = strings.TrimSpace(c[:len(c)-3])",
		"} else if len(c) >= 2 && strings.EqualFold(c[len(c)-2:], \"OR\") {",
		"c = strings.TrimSpace(c[:len(c)-2])",
		"}",
		`sb.WriteString("WHERE ")`,
		"sb.WriteString(c)",
		"}",
		"}",
	},
	"FilterByNameAndAge": {
		"var sb strings.Builder",
		"params := make([]any, 0, 2)",
		`sb.WriteString("name=? AND age=?")`,
		"params = append(params, name, age)",
		"e.Where(sb.String(), params...)",
	},
	"FilterWithTime": {
		"var sb strings.Builder",
		"params := make([]any, 0, 3)",
		`sb.WriteString("SELECT * FROM ?")`,
		"params = append(params, clause.CurrentTable)",
		"{",
		"var tmp strings.Builder",
		"if !start.IsZero() {",
		`tmp.WriteString("created_time > ?")`,
		"params = append(params, start)",
		"}",
		"if !end.IsZero() {",
		`tmp.WriteString("AND created_time < ?")`,
		"params = append(params, end)",
		"}",
		"c := strings.TrimSpace(tmp.String())",
		"if c != \"\" {",
		`sb.WriteString("WHERE ")`,
		"if len(c) >= 3 && strings.EqualFold(c[len(c)-3:], \"AND\") {",
		"c = strings.TrimSpace(c[:len(c)-3])",
		"} else if len(c) >= 2 && strings.EqualFold(c[len(c)-2:], \"OR\") {",
		"c = strings.TrimSpace(c[:len(c)-2])",
		"}",
		`sb.WriteString("WHERE ")`,
		"sb.WriteString(c)",
		"}",
		"}",
	},
}

// TestRenderSQLTemplate
func TestRenderSQLTemplate(t *testing.T) {
	const queryFilePath = "../../examples/query.go"

	fset := token.NewFileSet()
	parsedFile, err := parser.ParseFile(fset, queryFilePath, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse %s: %v", queryFilePath, err)
	}

	var queryInterface *ast.InterfaceType
	for _, decl := range parsedFile.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if ts.Name.Name == "Query" {
				if iface, ok := ts.Type.(*ast.InterfaceType); ok {
					queryInterface = iface
					break
				}
			}
		}
	}

	if queryInterface == nil {
		t.Fatalf("did not find Query[T any] interface in %s", queryFilePath)
	}

	for _, method := range queryInterface.Methods.List {
		name := method.Names[0].Name

		doc := extractSQL(method.Doc.Text(), name)
		if doc.Raw == "" && doc.Where == "" && doc.Select == "" {
			t.Fatalf("[SKIP] method %s has no doc", name)
			continue
		}

		if doc.Raw == "" {
			continue
		}

		got, err := RenderSQLTemplate(doc.Raw)
		t.Run(name, func(t *testing.T) {
			if err != nil {
				t.Fatalf("RenderSQLTemplate error for method %s: %v\nDoc:\n%s", name, err, doc)
			}

			gotLines := splitNonEmptyLines(got)
			wantLines := methodExpectedLines[name]

			if len(gotLines) != len(wantLines) {
				t.Errorf("line count mismatch for %s:\n gotLines=%d\n wantLines=%d\n---got---\n%v\n---want---\n%v\n",
					name, len(gotLines), len(wantLines),
					strings.Join(gotLines, "\n"), strings.Join(wantLines, "\n"))
				return
			}

			for i := range wantLines {
				gotLine := strings.TrimSpace(gotLines[i])
				wantLine := strings.TrimSpace(wantLines[i])
				if gotLine != wantLine {
					t.Errorf("method %s line %d mismatch:\n got:  %q\n want: %q", name, i+1, gotLine, wantLine)
				}
			}
		})
	}
}

func splitNonEmptyLines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		l = strings.TrimSpace(l)
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}
