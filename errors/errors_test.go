package errors

import (
	stderrors "errors"
	"testing"
)

func TestRegistry(t *testing.T) {
	r := NewRegistry()

	defA := r.Register(5001, "A")
	if defA == nil {
		t.Fatal("Register should return definition")
	}
	if defA.Code != 5001 || defA.Name != "A" {
		t.Fatalf("unexpected definition: %+v", defA)
	}

	defN1 := r.New("AUTO_ONE")
	if defN1 == nil {
		t.Fatal("New should return definition")
	}
	if defN1.Code != 4000 || defN1.Name != "AUTO_ONE" {
		t.Fatalf("unexpected auto definition: %+v", defN1)
	}

	defN2 := r.New("AUTO_TWO")
	if defN2.Code != 4001 || defN2.Name != "AUTO_TWO" {
		t.Fatalf("unexpected second auto definition: %+v", defN2)
	}

	defN1Again := r.New("AUTO_ONE")
	if defN1Again != defN1 {
		t.Fatal("New should return existing definition for same name")
	}

	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("Register should panic on duplicate name")
			}
		}()
		r.Register(5002, "A")
	}()
}

func TestRegistryDuplicateCode(t *testing.T) {
	r := NewRegistry()
	r.Register(6001, "X")

	defer func() {
		if recover() == nil {
			t.Fatal("Register should panic on duplicate code")
		}
	}()

	r.Register(6001, "Y")
}

func TestRegister(t *testing.T) {
	old := global
	global = NewRegistry()
	t.Cleanup(func() { global = old })

	def := Register(7001, "ORDER_NOT_FOUND")
	if def == nil {
		t.Fatal("Register should return definition")
	}
	if def.Code != 7001 || def.Name != "ORDER_NOT_FOUND" {
		t.Fatalf("unexpected definition: %+v", def)
	}
}

func TestNewAPI(t *testing.T) {
	old := global
	global = NewRegistry()
	t.Cleanup(func() { global = old })

	def1 := NewDefinition("PAYMENT_FAILED")
	if def1 == nil {
		t.Fatal("NewDefinition should return definition")
	}
	if def1.Code != 4000 || def1.Name != "PAYMENT_FAILED" {
		t.Fatalf("unexpected definition: %+v", def1)
	}

	def1Again := NewDefinition("PAYMENT_FAILED")
	if def1Again != def1 {
		t.Fatal("NewDefinition should return existing definition for same name")
	}

	def2 := NewDefinition("PAYMENT_TIMEOUT")
	if def2.Code != 4001 {
		t.Fatalf("expected auto code 4001, got %d", def2.Code)
	}

	errA := def1.New("pay failed")
	if got, want := errA.Error(), "[PAYMENT_FAILED] pay failed"; got != want {
		t.Fatalf("New() error = %q, want %q", got, want)
	}
	if !stderrors.Is(errA, def1) {
		t.Fatal("errors.Is should match definition")
	}

	errB := def1.Newf("pay failed: %d", 42)
	if got, want := errB.Error(), "[PAYMENT_FAILED] pay failed: 42"; got != want {
		t.Fatalf("Newf() error = %q, want %q", got, want)
	}

	cause := stderrors.New("db down")
	errC := def1.Wrap(cause, "write failed")
	if got, want := errC.Error(), "[PAYMENT_FAILED] write failed"; got != want {
		t.Fatalf("Wrap() error = %q, want %q", got, want)
	}
	if !stderrors.Is(errC, cause) {
		t.Fatal("Wrap should keep cause chain")
	}

	errD := def1.Wrapf(cause, "write failed: %s", "tx")
	if got, want := errD.Error(), "[PAYMENT_FAILED] write failed: tx"; got != want {
		t.Fatalf("Wrapf() error = %q, want %q", got, want)
	}

	defHTTP := def1.WithHTTP(422)
	if defHTTP == nil {
		t.Fatal("WithHTTP should return definition")
	}
	if defHTTP == def1 {
		t.Fatal("WithHTTP should return copied definition")
	}
	if defHTTP.HTTP != 422 {
		t.Fatalf("WithHTTP() HTTP = %d, want 422", defHTTP.HTTP)
	}
	if def1.HTTP != 0 {
		t.Fatalf("WithHTTP() should not mutate original definition, got %d", def1.HTTP)
	}
}

func TestIsHierarchy(t *testing.T) {
	base := &Definition{Code: 3000, Name: "BASE"}
	parent := &Definition{Code: 3001, Name: "PARENT", Class: base}
	child := &Definition{Code: 3002, Name: "CHILD", Class: parent}

	err := &codedError{def: child}

	if !stderrors.Is(err, child) {
		t.Fatal("errors.Is should match own definition")
	}
	if !stderrors.Is(err, parent) {
		t.Fatal("errors.Is should match parent definition")
	}
	if !stderrors.Is(err, base) {
		t.Fatal("errors.Is should match ancestor definition")
	}
	if stderrors.Is(err, &Definition{Code: 3999, Name: "OTHER"}) {
		t.Fatal("errors.Is should not match unrelated definition")
	}

	if !stderrors.Is(err, &codedError{def: child}) {
		t.Fatal("errors.Is should match codedError target with same code")
	}
	if stderrors.Is(err, &codedError{def: parent}) {
		t.Fatal("errors.Is should not match codedError target with different code")
	}
}

func TestCode(t *testing.T) {
	def := &Definition{Code: 4101, Name: "CODE_X"}
	err := &codedError{def: def}

	if got := Code(err); got != 4101 {
		t.Fatalf("Code(err) = %d, want 4101", got)
	}
	if got := Code(stderrors.New("x")); got != 0 {
		t.Fatalf("Code(non-coded) = %d, want 0", got)
	}
}

func TestName(t *testing.T) {
	def := &Definition{Code: 4201, Name: "NAME_X"}
	err := &codedError{def: def}

	if got := Name(err); got != "NAME_X" {
		t.Fatalf("Name(err) = %q, want %q", got, "NAME_X")
	}
	if got := Name(stderrors.New("x")); got != "" {
		t.Fatalf("Name(non-coded) = %q, want empty", got)
	}
}

func TestHTTPCode(t *testing.T) {
	parent := &Definition{Code: 4300, Name: "PARENT", HTTP: 409}
	childNoHTTP := &Definition{Code: 4301, Name: "CHILD", Class: parent}
	childWithHTTP := &Definition{Code: 4302, Name: "CHILD_HTTP", Class: parent, HTTP: 422}

	if got := HTTPCode(&codedError{def: childWithHTTP}); got != 422 {
		t.Fatalf("HTTPCode(childWithHTTP) = %d, want 422", got)
	}
	if got := HTTPCode(&codedError{def: childNoHTTP}); got != 409 {
		t.Fatalf("HTTPCode(childNoHTTP) = %d, want 409", got)
	}
	if got := HTTPCode(stderrors.New("x")); got != 500 {
		t.Fatalf("HTTPCode(non-coded) = %d, want 500", got)
	}
}

func TestCodedError(t *testing.T) {
	def := &Definition{Code: 1001, Name: "ITEM_NOT_FOUND"}
	cause := stderrors.New("db: no rows")

	err := &codedError{
		def:     def,
		message: "item missing",
		cause:   cause,
	}

	if got, want := err.Error(), "[ITEM_NOT_FOUND] item missing"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}

	err.message = ""
	if got, want := err.Error(), "[ITEM_NOT_FOUND]"; got != want {
		t.Fatalf("Error() without message = %q, want %q", got, want)
	}

	if got := err.Unwrap(); got != cause {
		t.Fatalf("Unwrap() = %v, want %v", got, cause)
	}

	if !stderrors.Is(err, cause) {
		t.Fatal("errors.Is should match wrapped cause")
	}
}

func TestCodedErrorIs(t *testing.T) {
	base := &Definition{Code: 2000, Name: "BASE"}
	parent := &Definition{Code: 2001, Name: "PARENT", Class: base}
	child := &Definition{Code: 2002, Name: "CHILD", Class: parent}

	err := &codedError{def: child}

	if !err.Is(&codedError{def: child}) {
		t.Fatal("Is should match same codedError code")
	}
	if err.Is(&codedError{def: parent}) {
		t.Fatal("Is should not match different codedError code")
	}

	if !err.Is(child) {
		t.Fatal("Is should match own definition")
	}
	if !err.Is(parent) {
		t.Fatal("Is should match parent definition")
	}
	if !err.Is(base) {
		t.Fatal("Is should match ancestor definition")
	}
	if err.Is(&Definition{Code: 9999, Name: "OTHER"}) {
		t.Fatal("Is should not match unrelated definition")
	}

	if !stderrors.Is(err, &codedError{def: child}) {
		t.Fatal("errors.Is should match codedError target")
	}
	if !stderrors.Is(err, parent) {
		t.Fatal("errors.Is should match parent definition target")
	}
	if stderrors.Is(err, &Definition{Code: 9999, Name: "OTHER"}) {
		t.Fatal("errors.Is should not match unrelated definition target")
	}
}

func TestSentinel(t *testing.T) {
	cases := []struct {
		name string
		def  *Definition
		code int
		http int
	}{
		{name: "NOT_FOUND", def: NotFound, code: 2001, http: 404},
		{name: "INVALID_PARAM", def: InvalidParam, code: 2002, http: 400},
		{name: "UNAUTHORIZED", def: Unauthorized, code: 2003, http: 401},
		{name: "FORBIDDEN", def: Forbidden, code: 2004, http: 403},
		{name: "INTERNAL_ERROR", def: Internal, code: 2005, http: 500},
		{name: "TIMEOUT", def: Timeout, code: 2006, http: 504},
		{name: "BAD_GATEWAY", def: BadGateway, code: 2007, http: 502},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.def == nil {
				t.Fatal("sentinel definition should not be nil")
			}
			if tc.def.Code != tc.code {
				t.Fatalf("Code = %d, want %d", tc.def.Code, tc.code)
			}
			if tc.def.Name != tc.name {
				t.Fatalf("Name = %q, want %q", tc.def.Name, tc.name)
			}
			if tc.def.HTTP != tc.http {
				t.Fatalf("HTTP = %d, want %d", tc.def.HTTP, tc.http)
			}

			err := tc.def.New("boom")
			if got, want := err.Error(), "["+tc.name+"] boom"; got != want {
				t.Fatalf("error string = %q, want %q", got, want)
			}
			if got := Code(err); got != tc.code {
				t.Fatalf("Code(err) = %d, want %d", got, tc.code)
			}
			if got := HTTPCode(err); got != tc.http {
				t.Fatalf("HTTPCode(err) = %d, want %d", got, tc.http)
			}
		})
	}
}
