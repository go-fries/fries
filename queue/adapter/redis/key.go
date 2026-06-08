package redis

func (q *Queue) streamKey(name string) string {
	return q.prefix + ":" + name + ":stream"
}

func (q *Queue) delayedKey(name string) string {
	return q.prefix + ":" + name + ":delayed"
}

func (q *Queue) deadLetterKey(name string) string {
	return q.prefix + ":" + name + ":dead"
}
