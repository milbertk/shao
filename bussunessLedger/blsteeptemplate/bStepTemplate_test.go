package blsteptemplate

import (
	"errors"
	"testing"

	"github.com/milbertk/shao/dataBasesLedger/steptemplate"
)

func step(name string, done bool) steptemplate.StepTemplateStepS {
	return steptemplate.StepTemplateStepS{Name: name, IsDoneStep: done, RequiresPrevious: true}
}

func TestCleanSteps(t *testing.T) {
	got := cleanSteps([]steptemplate.StepTemplateStepS{
		step("Confirm purchase", false), step("  ", false), step("Address", false), step("Date", true), step("", false),
	})
	if len(got) != 3 {
		t.Fatalf("want 3 steps, got %d", len(got))
	}
	for i, st := range got {
		if st.StepOrder != i+1 {
			t.Errorf("step %d order %d", i, st.StepOrder)
		}
	}
	if got[0].RequiresPrevious {
		t.Error("first step must not require a previous one")
	}
	if !got[2].IsDoneStep || got[0].IsDoneStep || got[1].IsDoneStep {
		t.Error("only the last flagged step must be done")
	}
	none := cleanSteps([]steptemplate.StepTemplateStepS{step("A", false), step("B", false)})
	if !none[1].IsDoneStep || none[0].IsDoneStep {
		t.Error("with no flag the last step is done")
	}
}

func TestPlanStepMerge(t *testing.T) {
	existing := []steptemplate.StepTemplateStepS{{StepTemplateStepID: "s1", Name: "A"}, {StepTemplateStepID: "s2", Name: "B"}}
	grow := planStepMerge(existing, cleanSteps([]steptemplate.StepTemplateStepS{step("A", false), step("B2", false), step("C", false), step("D", false)}))
	if len(grow.Updates) != 2 || grow.Updates[0].StepTemplateStepID != "s1" || grow.Updates[1].StepTemplateStepID != "s2" {
		t.Fatalf("updates must keep ids: %+v", grow.Updates)
	}
	if len(grow.Inserts) != 2 || grow.Inserts[0].StepTemplateStepID != "" || len(grow.Removes) != 0 {
		t.Fatalf("unexpected plan %+v", grow)
	}
	shrink := planStepMerge(existing, cleanSteps([]steptemplate.StepTemplateStepS{step("A", false)}))
	if len(shrink.Updates) != 1 || len(shrink.Inserts) != 0 || len(shrink.Removes) != 1 || shrink.Removes[0].StepTemplateStepID != "s2" {
		t.Fatalf("unexpected plan %+v", shrink)
	}
}

func TestStepInUseError(t *testing.T) {
	err := error(&StepInUseError{Name: "Shipped"})
	if !errors.Is(err, ErrStepInUse) {
		t.Fatal("must wrap ErrStepInUse")
	}
}
