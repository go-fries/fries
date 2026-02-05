package env

import "slices"

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

var currentEnv = Prod

func SetEnv(env Env) {
	currentEnv = env
}

func GetEnv() Env {
	return currentEnv
}

func Is(envs ...Env) bool {
	return currentEnv.Is(envs...)
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
	for _, env := range envs {
		if currentEnv == Env(env) {
			return true
		}
	}

	return false
}
