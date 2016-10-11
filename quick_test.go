package quick_test

import ("goblin"
	"testing"
        "testing/quick"
	"fmt"
	"math"
	"strconv"
)

func TestRoundTripFloat(t *testing.T) {
	f := func(flt float64) bool {
		flt = math.Abs(flt)
		needed := fmt.Sprintf("%f", flt)
		gotten := goblin.TestExpr(needed)
		result, _ := strconv.ParseFloat(gotten["value"].(string), 64)
		return flt == result
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestProvidedFloat(t *testing.T) {
	gotten := goblin.TestExpr("3.14")
	if gotten["value"].(string) != "3.14" {
		t.Error("Floats not parsing correctly")
	}
}


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
