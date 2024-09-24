package avalanchecore

type RollingAvg struct {
	values []int64
	sum    int64
	size   int
	index  int
	count  int
}

func NewRollingAvg(n int) *RollingAvg {
	return &RollingAvg{
		values: make([]int64, n),
		size:   n,
		index:  0,
		count:  0,
	}
}

func (r *RollingAvg) Push(v int64) {
	// Remove bottom of list from the sum
	r.sum -= r.values[r.index]

	// Put current value on the top of the list
	r.values[r.index] = v
	r.sum += v

	// Advance the index
	r.index = (r.index + 1) % r.size

	// Advance the count (for when there's <n push events recorded)
	if r.count < r.size {
		r.count++
	}
}

func (r *RollingAvg) Avg() int64 {
	if r.count == 0 {
		return 0
	}

	return r.sum / int64(r.count)
}
