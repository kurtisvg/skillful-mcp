package tools

import (
	"context"
	"strings"
	"testing"

	monty "github.com/ewhauser/gomonty"
)

func TestExecuteCodeDescriptionRefersToUseSkill(t *testing.T) {
	if !strings.Contains(executeCodeDescription, "use_skill") {
		t.Error("description should refer to use_skill for tool discovery")
	}
	if !strings.Contains(executeCodeDescription, "tool_name") {
		t.Error("description should show tools are called by name")
	}
	if !strings.Contains(executeCodeDescription, "resources") {
		t.Error("description should mention resources")
	}
}

func TestExecuteCodeBasicMath(t *testing.T) {
	runner, err := monty.New("40 + 2", monty.CompileOptions{ScriptName: "script.py"})
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	value, err := runner.Run(context.Background(), monty.RunOptions{})
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	if value.String() != "42" {
		t.Errorf("result = %q, want '42'", value.String())
	}
}

func TestExecuteCodeStringExpression(t *testing.T) {
	runner, err := monty.New("'hello' + ' ' + 'world'", monty.CompileOptions{ScriptName: "script.py"})
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	value, err := runner.Run(context.Background(), monty.RunOptions{})
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	if value.String() != "hello world" {
		t.Errorf("result = %q, want 'hello world'", value.String())
	}
}

func TestExecuteCodeSyntaxError(t *testing.T) {
	_, err := monty.New("def (invalid syntax", monty.CompileOptions{ScriptName: "script.py"})
	if err == nil {
		t.Fatal("expected compile error for invalid syntax")
	}
}

// --- extractParamSchema tests ---

func TestExtractParamSchemaRequiredAndOptional(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"limit": map[string]any{"type": "integer"},
			"sql":   map[string]any{"type": "string"},
		},
		"required": []any{"sql"},
	}
	params := extractParamSchema(schema)
	if len(params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(params))
	}
	// Required first, then optional sorted.
	if params[0].Name != "sql" || params[0].Types[0] != "string" {
		t.Errorf("params[0] = %+v, want sql/string", params[0])
	}
	if params[1].Name != "limit" || params[1].Types[0] != "integer" {
		t.Errorf("params[1] = %+v, want limit/integer", params[1])
	}
}

func TestExtractParamSchemaNoRequired(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"gamma": map[string]any{"type": "string"},
			"alpha": map[string]any{"type": "string"},
			"beta":  map[string]any{"type": "number"},
		},
	}
	params := extractParamSchema(schema)
	if len(params) != 3 {
		t.Fatalf("expected 3 params, got %d", len(params))
	}
	// All sorted lexicographically.
	expected := []string{"alpha", "beta", "gamma"}
	for i, name := range expected {
		if params[i].Name != name {
			t.Errorf("params[%d].Name = %q, want %q", i, params[i].Name, name)
		}
	}
}

func TestExtractParamSchemaRequiredWithNonRequiredSorted(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"z": map[string]any{"type": "string"},
			"c": map[string]any{"type": "string"},
			"a": map[string]any{"type": "string"},
		},
		"required": []any{"z"},
	}
	params := extractParamSchema(schema)
	if len(params) != 3 {
		t.Fatalf("expected 3 params, got %d", len(params))
	}
	// z first (required), then a, c (sorted).
	expected := []string{"z", "a", "c"}
	for i, name := range expected {
		if params[i].Name != name {
			t.Errorf("params[%d].Name = %q, want %q", i, params[i].Name, name)
		}
	}
}

func TestExtractParamSchemaNilSchema(t *testing.T) {
	params := extractParamSchema(nil)
	if params != nil {
		t.Errorf("expected nil, got %v", params)
	}
}

func TestExtractParamSchemaEmptyProperties(t *testing.T) {
	schema := map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
	params := extractParamSchema(schema)
	if len(params) != 0 {
		t.Errorf("expected empty, got %v", params)
	}
}

func TestExtractParamSchemaArrayType(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"value": map[string]any{"type": []any{"string", "null"}},
		},
	}
	params := extractParamSchema(schema)
	if len(params) != 1 {
		t.Fatalf("expected 1 param, got %d", len(params))
	}
	if len(params[0].Types) != 2 || params[0].Types[0] != "string" || params[0].Types[1] != "null" {
		t.Errorf("types = %v, want [string null]", params[0].Types)
	}
}

func TestExtractParamSchemaNoTypeField(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"value": map[string]any{"description": "no type here"},
		},
	}
	params := extractParamSchema(schema)
	if len(params) != 1 {
		t.Fatalf("expected 1 param, got %d", len(params))
	}
	if params[0].Types != nil {
		t.Errorf("types = %v, want nil", params[0].Types)
	}
}

// --- validateMontyValue tests ---

func TestValidateMontyValueStringMatch(t *testing.T) {
	err := validateMontyValue(monty.String("hello"), paramInfo{Name: "x", Types: []string{"string"}})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateMontyValueIntegerMatch(t *testing.T) {
	err := validateMontyValue(monty.Int(42), paramInfo{Name: "x", Types: []string{"integer"}})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateMontyValueNumberAcceptsInt(t *testing.T) {
	err := validateMontyValue(monty.Int(42), paramInfo{Name: "x", Types: []string{"number"}})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateMontyValueNumberAcceptsFloat(t *testing.T) {
	err := validateMontyValue(monty.Float(3.14), paramInfo{Name: "x", Types: []string{"number"}})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateMontyValueBooleanMatch(t *testing.T) {
	err := validateMontyValue(monty.Bool(true), paramInfo{Name: "x", Types: []string{"boolean"}})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateMontyValueArrayMatch(t *testing.T) {
	err := validateMontyValue(monty.List(monty.Int(1)), paramInfo{Name: "x", Types: []string{"array"}})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateMontyValueObjectMatch(t *testing.T) {
	dict := monty.DictValue(monty.Dict{{Key: monty.String("k"), Value: monty.String("v")}})
	err := validateMontyValue(dict, paramInfo{Name: "x", Types: []string{"object"}})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateMontyValueTypeMismatch(t *testing.T) {
	err := validateMontyValue(monty.Int(42), paramInfo{Name: "sql", Types: []string{"string"}})
	if err == nil {
		t.Fatal("expected error for type mismatch")
	}
	if !strings.Contains(err.Error(), "sql") {
		t.Errorf("error should mention parameter name, got: %v", err)
	}
}

func TestValidateMontyValueNullableString(t *testing.T) {
	// None should pass for ["string", "null"].
	err := validateMontyValue(monty.None(), paramInfo{Name: "x", Types: []string{"string", "null"}})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// String should also pass.
	err = validateMontyValue(monty.String("hi"), paramInfo{Name: "x", Types: []string{"string", "null"}})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateMontyValueNoTypes(t *testing.T) {
	// Nil types should skip validation.
	err := validateMontyValue(monty.Int(42), paramInfo{Name: "x", Types: nil})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- Integration: type validation via Monty runner ---

func TestExecuteCodeTypeValidation(t *testing.T) {
	// Register a function with a string parameter schema, then call it with an int.
	params := []paramInfo{{Name: "name", Types: []string{"string"}}}
	paramByName := map[string]paramInfo{"name": params[0]}

	fn := func(fnCtx context.Context, call monty.Call) (monty.Result, error) {
		args := make(map[string]any)
		for i, val := range call.Args {
			if i < len(params) {
				if err := validateMontyValue(val, params[i]); err != nil {
					msg := err.Error()
					return monty.Raise(monty.Exception{Type: "TypeError", Arg: &msg}), nil
				}
				args[params[i].Name] = montyValueToAny(val)
			}
		}
		for _, pair := range call.Kwargs {
			key, ok := pair.Key.Raw().(string)
			if !ok {
				continue
			}
			if pi, ok := paramByName[key]; ok {
				if err := validateMontyValue(pair.Value, pi); err != nil {
					msg := err.Error()
					return monty.Raise(monty.Exception{Type: "TypeError", Arg: &msg}), nil
				}
			}
			args[key] = montyValueToAny(pair.Value)
		}
		return monty.Return(monty.String("ok")), nil
	}

	// Valid call: string argument.
	t.Run("valid_string_arg", func(t *testing.T) {
		runner, err := monty.New(`greet("world")`, monty.CompileOptions{ScriptName: "test.py"})
		if err != nil {
			t.Fatal(err)
		}
		value, err := runner.Run(context.Background(), monty.RunOptions{
			Functions: map[string]monty.ExternalFunction{"greet": fn},
		})
		if err != nil {
			t.Fatalf("runtime error: %v", err)
		}
		if value.String() != "ok" {
			t.Errorf("result = %q, want 'ok'", value.String())
		}
	})

	// Invalid call: int argument where string expected.
	t.Run("invalid_int_for_string", func(t *testing.T) {
		runner, err := monty.New(`greet(42)`, monty.CompileOptions{ScriptName: "test.py"})
		if err != nil {
			t.Fatal(err)
		}
		_, err = runner.Run(context.Background(), monty.RunOptions{
			Functions: map[string]monty.ExternalFunction{"greet": fn},
		})
		if err == nil {
			t.Fatal("expected runtime error for type mismatch")
		}
		if !strings.Contains(err.Error(), "TypeError") {
			t.Errorf("expected TypeError, got: %v", err)
		}
	})
}
