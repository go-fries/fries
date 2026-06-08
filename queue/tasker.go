package queue

// HandleTasker registers tasker as the typed handler for its TaskType.
//
// It is equivalent to calling HandleFor(tasker.TaskType(), tasker). A nil tasker
// is ignored, matching HandleFor's nil-handler behavior.
func HandleTasker[T any](tasker Tasker[T]) WorkerOption {
	if tasker == nil {
		return Handle("", nil)
	}
	return HandleFor(tasker.TaskType(), tasker)
}
