package iso

import (
	"github.com/oid-directory/go-radir"
)

func NewRoot(profile *radir.DITProfile) *radir.Registration {
	return profile.NewRegistration()
}
