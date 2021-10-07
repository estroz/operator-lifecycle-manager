package constraints

import (
	"bytes"
	"encoding/json"
	"fmt"
)

const OLMConstraintType = "olm.constraint"

// Constraint holds parsed, potentially nested dependency constraints.
type Constraint struct {
	Message string `json:"message"`

	Package *PackageConstraint `json:"package,omitempty"`
	GVK     *GVKConstraint     `json:"gvk,omitempty"`

	All  *CompoundConstraint `json:"all,omitempty"`
	Any  *CompoundConstraint `json:"any,omitempty"`
	None *CompoundConstraint `json:"none,omitempty"`
}

// CompoundConstraint holds a list of potentially nested constraints
// over which a boolean operation is applied.
type CompoundConstraint struct {
	Constraints []Constraint `json:"constraints"`
}

// GVKConstraint defines a GVK constraint.
type GVKConstraint struct {
	Group   string `json:"group"`
	Kind    string `json:"kind"`
	Version string `json:"version"`
}

// PackageConstraint defines a package constraint.
type PackageConstraint struct {
	// Name of the package.
	Name string `json:"name"`
	// VersionRange required for the package.
	VersionRange string `json:"versionRange"`
}

// maxConstraintSize defines the maximum raw size in bytes of an olm.constraint.
// 64Kb seems reasonable, since this number allows for long description strings
// and either few deep nestings or shallow nestings and long constraints lists,
// but not both.
const maxConstraintSize = 2 << 16

// ErrMaxConstraintSizeExceeded is returned when a constraint's size > maxConstraintSize.
var ErrMaxConstraintSizeExceeded = fmt.Errorf("olm.constraint value is greater than max constraint size %d", maxConstraintSize)

// Parse parses an olm.constraint property's value recursively into a Constraint.
// Unknown value schemas result in an error. A too-large value results in an error.
func Parse(v json.RawMessage) (c Constraint, err error) {
	// There is no way to explicitly limit nesting depth.
	// From https://github.com/golang/go/issues/31789#issuecomment-538134396,
	// the recommended approach is to error out if raw input size
	// is greater than some threshold.
	if len(v) > maxConstraintSize {
		return c, ErrMaxConstraintSizeExceeded
	}

	d := json.NewDecoder(bytes.NewBuffer(v))
	d.DisallowUnknownFields()
	err = d.Decode(&c)

	return
}
