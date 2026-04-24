package sect

import "testing"

func TestNewInitialSectStateInitializesRootAndBoundary(t *testing.T) {
	state := NewInitialSectState(SectID("sect-preview"), UserID("player-preview"), "青崖宗")

	if state.Meta.SectID != SectID("sect-preview") {
		t.Fatalf("unexpected sect id: %q", state.Meta.SectID)
	}
	if state.Runtime.AuthorityBoundary.SchemaVersion != SectStateSchemaVersion {
		t.Fatalf("unexpected schema version: %d", state.Runtime.AuthorityBoundary.SchemaVersion)
	}
	if state.Runtime.AuthorityBoundary.SimulationVersion != SectStateSimulationVersion {
		t.Fatalf("unexpected simulation version: %d", state.Runtime.AuthorityBoundary.SimulationVersion)
	}
	if state.Runtime.AuthorityBoundary.ConfigVersion != SectStateConfigVersion {
		t.Fatalf("unexpected config version: %d", state.Runtime.AuthorityBoundary.ConfigVersion)
	}
	if state.Disciples == nil || state.Buildings == nil || state.Tasks == nil {
		t.Fatalf("authoritative root maps must be initialized")
	}
	if state.Institutions.ByID == nil || state.Storylets == nil {
		t.Fatalf("institution and storylet state must stay inside the root big-state")
	}
	if state.Events.ActiveEvents == nil || state.Events.ResolvedEvents == nil {
		t.Fatalf("sect event seed and omen state must initialize inside the root big-state")
	}
	if state.Resources.Stock == nil || state.Resources.Nodes == nil {
		t.Fatalf("resource state must initialize both stock and node maps")
	}
	if state.Resources.Stock[ResourceKindSpiritStone] <= 0 {
		t.Fatalf("expected initial spirit stone reserve, got %+v", state.Resources.Stock)
	}
	if state.Admissions.Candidates == nil {
		t.Fatalf("admission candidate pool must initialize inside the root big-state")
	}
}

func TestNewInitialSectStateAdmitsStarterDisciple(t *testing.T) {
	state := NewInitialSectState(SectID("sect-preview"), UserID("player-preview"), "青崖宗")

	if len(state.Disciples) != 1 {
		t.Fatalf("expected exactly one starter disciple, got %+v", state.Disciples)
	}
	starter, ok := state.Disciples[starterDiscipleID]
	if !ok {
		t.Fatalf("expected starter disciple %q, got %+v", starterDiscipleID, state.Disciples)
	}
	if starter.Name == "" {
		t.Fatalf("expected starter disciple name to be populated")
	}
	if starter.HP != starter.MaxHP || starter.HP <= 0 {
		t.Fatalf("expected starter disciple hp/max hp to be initialized, got %+v", starter)
	}
	if starter.AssignmentKind != DiscipleAssignmentIdle {
		t.Fatalf("expected authority-owned idle assignment, got %+v", starter)
	}
	if starter.Identity != IdentityRankOuter {
		t.Fatalf("expected starter disciple identity in authority state, got %+v", starter)
	}
	if starter.Aptitude.SpiritRoot <= 0 || starter.Aptitude.Comprehension <= 0 || starter.Aptitude.Physique <= 0 {
		t.Fatalf("expected starter disciple aptitude in authority state, got %+v", starter.Aptitude)
	}
	if starter.WorkTarget.Description == "" {
		t.Fatalf("expected authority-owned work target description, got %+v", starter.WorkTarget)
	}
	if starter.Needs.DailySpiritGrain <= 0 || starter.Needs.DailyRestTicks <= 0 {
		t.Fatalf("expected authority-owned disciple needs, got %+v", starter.Needs)
	}
	if !starter.Support.FoodSatisfied || !starter.Support.HousingSatisfied {
		t.Fatalf("expected starter disciple to have basic support state, got %+v", starter.Support)
	}
	account, ok := state.Contribution.Accounts[starterDiscipleID]
	if !ok || account.DiscipleID != starterDiscipleID {
		t.Fatalf("expected starter disciple contribution account in SectState, got %+v", state.Contribution.Accounts)
	}
}
