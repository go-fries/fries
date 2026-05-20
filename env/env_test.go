package env

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnv(t *testing.T) {
	assert.Equal(t, Env("dev"), Dev)
	assert.Equal(t, Env("prod"), Prod)
	assert.Equal(t, Env("debug"), Debug)
	assert.Equal(t, Env("stage"), Stage)
}

func TestString(t *testing.T) {
	assert.Equal(t, "dev", Dev.String())
	assert.Equal(t, "prod", Prod.String())
	assert.Equal(t, "debug", Debug.String())
	assert.Equal(t, "stage", Stage.String())
}

func TestIs(t *testing.T) {
	SetEnv(Dev)
	assert.True(t, Is(Dev))
	assert.False(t, Is(Prod))
	assert.False(t, Is(Debug))
	assert.False(t, Is(Stage))
	assert.True(t, Is(Dev, Prod))
	assert.True(t, IsDev())
	assert.False(t, IsProd())
	assert.False(t, IsDebug())
	assert.False(t, IsStage())
	assert.True(t, IsUseString("dev"))
	assert.False(t, IsUseString("prod"))
	assert.True(t, IsUseString("dev", "prod"))
	assert.True(t, Is("dev"))
	assert.False(t, Is("prod"))
	assert.Equal(t, Dev, GetEnv())

	SetEnv(Prod)
	assert.True(t, Is(Prod))
	assert.False(t, Is(Dev))
	assert.Equal(t, Prod, GetEnv())
}

func TestCustom(t *testing.T) {
	SetEnv("online")

	assert.True(t, Is("online"))
	assert.False(t, Is("dev"))
}

func TestConcurrentSafe(t *testing.T) {
	t.Cleanup(func() {
		SetEnv(Prod)
	})

	envs := []Env{Dev, Prod, Debug, Stage, "online"}

	var wg sync.WaitGroup
	for i := range 64 {
		wg.Add(1)
		go func(offset int) {
			defer wg.Done()

			for j := range 1000 {
				SetEnv(envs[(offset+j)%len(envs)])
			}
		}(i)
	}

	for range 64 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for range 1000 {
				_ = GetEnv()
				_ = Is(Dev, Prod, Debug, Stage, "online")
				_ = IsDev()
				_ = IsProd()
				_ = IsDebug()
				_ = IsStage()
				_ = IsUseString("dev", "prod", "debug", "stage", "online")
			}
		}()
	}

	wg.Wait()
}
