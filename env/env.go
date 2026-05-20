package env

import (
	"slices"
	"sync/atomic"
)

type Env string

var (
	Dev   Env = "dev"
	Prod  Env = "prod"
	Debug Env = "debug"
	Stage Env = "stage"
)

func (e Env) String() string {
	return string(e)
}

func (e Env) Is(envs ...Env) bool {
	return slices.Contains(envs, e)
}

var (
	defaultCurrentEnv = Prod
	currentEnv        atomic.Pointer[Env]
)

func init() {
	currentEnv.Store(&defaultCurrentEnv)
}

func SetEnv(env Env) {
	currentEnv.Store(&env)
}

func GetEnv() Env {
	if env := currentEnv.Load(); env != nil {
		return *env
	}

	return Prod
}

func Is(envs ...Env) bool {
	return GetEnv().Is(envs...)
}

func IsDev() bool {
	return Is(Dev)
}

func IsProd() bool {
	return Is(Prod)
}

func IsDebug() bool {
	return Is(Debug)
}

func IsStage() bool {
	return Is(Stage)
}

func IsUseString(envs ...string) bool {
	envSnapshot := GetEnv()
	for _, env := range envs {
		if envSnapshot == Env(env) {
			return true
		}
	}

	return false
}
