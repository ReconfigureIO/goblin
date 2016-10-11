package quick_test

import ("goblin"
	"testing"
        "testing/quick"
	"fmt"
       )

func TestRoundTripUInt(t *testing.T) {
	f := func(int uint64) bool {
		needed := fmt.Sprintf("%d", int)
		gotten := goblin.TestExpr(needed)
		return needed == gotten["value"]
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}
