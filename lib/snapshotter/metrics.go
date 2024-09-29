package snapshotter

type snapshotMetrics struct {
	totalSelected int
	updated       int
	unchanged     int
	errored       int
}

func (m *snapshotMetrics) Add(other *snapshotMetrics) {
	m.totalSelected += other.totalSelected
	m.updated += other.updated
	m.unchanged += other.unchanged
}
