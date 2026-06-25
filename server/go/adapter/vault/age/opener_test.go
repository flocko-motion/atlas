package age_test

import (
	"context"
	"testing"

	"rankedb/adapter/config"
	cmem "rankedb/adapter/config/mem"
	vaultage "rankedb/adapter/vault/age"
)

// The age opener declares it needs secret-zero, and opens only with the
// right master — exactly what the assembler keys its seal decision on.
func TestOpenerNeedsSecret(t *testing.T) {
	ctx := context.Background()
	ct, _ := vaultage.Encrypt("master", map[string]string{"k": "v"})
	cell := config.NewCell(cmem.New(), "vault.value.age")
	if err := cell.Set(ctx, ct); err != nil {
		t.Fatal(err)
	}

	op := vaultage.Opener(cell)
	if !op.NeedsSecret() {
		t.Fatal("age opener should report NeedsSecret() == true")
	}
	v, err := op.Open(ctx, []byte("master"))
	if err != nil {
		t.Fatalf("Open with master: %v", err)
	}
	if sec, err := v.Secret("k"); err != nil || string(sec) != "v" {
		t.Fatalf("Secret = (%q, %v), want v", sec, err)
	}
	if _, err := op.Open(ctx, []byte("wrong")); err == nil {
		t.Fatal("Open with wrong master should fail")
	}
}
