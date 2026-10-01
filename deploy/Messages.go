package deploy

// TickMsg drives the streamed, line-by-line reveal of a pre-computed
// rollout transcript (see Plan). It carries no payload; the UI layer tracks
// how many lines have been revealed so far.
type TickMsg struct{}
