package tmp

import (
	"fmt"
	"os"
	"testing"
)

func TestEnvVar(t *testing.T) {
	fmt.Printf("ENTITLE_WORKFLOW_ID=%q\n", os.Getenv("ENTITLE_WORKFLOW_ID"))
	if os.Getenv("ENTITLE_WORKFLOW_ID") == "" {
		t.Fatal("ENTITLE_WORKFLOW_ID not set in environment")
	}
}
