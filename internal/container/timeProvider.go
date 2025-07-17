package container

import "time"

type TimeProvider interface {
	Sleep(duration time.Duration)
	Now() time.Time
}

type RealTimeProvider struct{}

func (r *RealTimeProvider) Sleep(duration time.Duration) {
	time.Sleep(duration)
}

func (r *RealTimeProvider) Now() time.Time {
	return time.Now()
}
