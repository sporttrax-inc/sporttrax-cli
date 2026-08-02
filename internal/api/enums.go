package api

// Closed value sets mirroring the server's enums. Both surfaces read them
// from here — commands for flag help, completions, and pre-request
// validation; MCP tools for their input schemas — so a value can never be
// accepted on one surface and rejected on the other.
//
// Keep each list exactly as wide as the server's enum. Narrower locks
// users out of real data (road meets were listable but unfilterable for
// exactly this reason); wider sends values the API silently matches
// nothing for.
var (
	Sports    = []string{"track", "xc", "road"}
	Genders   = []string{"male", "female", "mixed"}
	Levels    = []string{"professional", "college", "high_school", "middle_school", "elementary", "unattached", "hs_unified", "club"}
	Rounds    = []string{"finals", "semi_finals", "quarter_finals", "prelims"}
	MarkTypes = []string{"time", "distance", "score"}

	// ResultSorts are the sort keys /results accepts; a leading "-"
	// reverses. Other collections sort by id only, which is the default.
	ResultSorts = []string{"at", "-at", "place", "-place", "id", "-id"}
)
