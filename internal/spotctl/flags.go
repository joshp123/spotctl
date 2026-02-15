package spotctl

// Small helpers because Go's stdlib flag package stops parsing at the first
// non-flag argument. For agent / automation usage we want "--json" etc to work
// even if it appears at the end.

func popBoolFlag(args []string, name string) (bool, []string) {
	out := make([]string, 0, len(args))
	set := false
	for _, a := range args {
		if a == name {
			set = true
			continue
		}
		out = append(out, a)
	}
	return set, out
}
