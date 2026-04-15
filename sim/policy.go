package sim

// PolicyRegistry maps policy names to their implementation.
var PolicyRegistry = map[string]PolicyFunc{
	"greedy-fill":    GreedyFillPolicy,
	"greedy-drain":   GreedyDrainPolicy,
	"two-threshold":  TwoThresholdPolicy,
}

// AllPolicyNames returns the policy names in a stable display order.
func AllPolicyNames() []string {
	return []string{"greedy-fill", "greedy-drain", "two-threshold"}
}
