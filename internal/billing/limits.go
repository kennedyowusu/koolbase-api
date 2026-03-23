package billing

const EnforceLimits = false

type PlanLimit struct {
	MaxEnvironments int
	MaxFlags        int
	MaxConfigs      int
	MaxMembers      int
	MaxFunctions    int
	MaxSecrets      int
}

var Plans = map[string]PlanLimit{
	"free": {
		MaxEnvironments: 1,
		MaxFlags:        5,
		MaxConfigs:      10,
		MaxMembers:      1,
		MaxFunctions:    10,
		MaxSecrets:      5,
	},
	"pro": {
		MaxEnvironments: 3,
		MaxFlags:        -1,
		MaxConfigs:      -1,
		MaxMembers:      5,
		MaxFunctions:    -1,
		MaxSecrets:      -1,
	},
	"enterprise": {
		MaxEnvironments: -1,
		MaxFlags:        -1,
		MaxConfigs:      -1,
		MaxMembers:      -1,
		MaxFunctions:    -1,
		MaxSecrets:      -1,
	},
}

func GetLimit(plan string, resource string) int {
	p, ok := Plans[plan]
	if !ok {
		p = Plans["free"]
	}
	switch resource {
	case "environments":
		return p.MaxEnvironments
	case "flags":
		return p.MaxFlags
	case "configs":
		return p.MaxConfigs
	case "members":
		return p.MaxMembers
	case "functions":
		return p.MaxFunctions
	case "secrets":
		return p.MaxSecrets
	}
	return -1
}

func IsAtLimit(plan string, resource string, currentCount int) bool {
	if !EnforceLimits {
		return false
	}
	limit := GetLimit(plan, resource)
	if limit == -1 {
		return false
	}
	return currentCount >= limit
}

type Usage struct {
	Resource string `json:"resource"`
	Current  int    `json:"current"`
	Limit    int    `json:"limit"`
	Plan     string `json:"plan"`
}
