package account

type hashEntry struct {
	userHash string
	timeSec  int64
}

type hashEntryHeap []*hashEntry

func (heap hashEntryHeap) Len() int {
	return len(heap)
}

func (heap hashEntryHeap) Less(i, j int) bool {
	return heap[i].timeSec < heap[j].timeSec
}

func (heap hashEntryHeap) Swap(i, j int) {
	heap[i], heap[j] = heap[j], heap[i]
}

func (heap *hashEntryHeap) Push(x interface{}) {
	*heap = append(*heap, x.(*hashEntry))
}

func (heap *hashEntryHeap) Pop() interface{} {
	old := *heap
	n := len(old)
	x := old[n-1]
	*heap = old[:n-1]
	return x
}

func newHashEntryHeap(capacity int) *hashEntryHeap {
	var heap hashEntryHeap = make([]*hashEntry, 0, capacity)
	return &heap
}
