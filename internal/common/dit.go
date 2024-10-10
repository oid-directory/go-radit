package common

import (
	"encoding/csv"
	"errors"

	"github.com/oid-directory/go-radir"
)

/*
OIDTree implements an abstraction of an OID tree.

Instances of this type are automatically initialized within instances of
*[DIT], and may be populated using the [OIDTree.Prime] method.

Root branches may be accessed using the [DIT.ITUT], [DIT.ISO], [DIT.JointISOITUT]
and [DIT.Root] methods.
*/
type OIDTree [3]*radir.Registration

/*
DIT implements an mere abstraction of a directory information tree with
respect to the spirit of the OID Directory I-D series. It is not a true
functional X.500/LDAP DIT, nor does it have any RFC4511 functionality.
It is stored entirely in memory.

Generally, the purpose of instances of this type is for temporary assembly
of DIT content derived from the various sources supported by this package,
but could conceivably be used as a replacement for a directory information
tree when a real one is not available.
*/
type DIT struct {
	tree    OIDTree
	aths    *radir.Registrants
	bsel    [2]int // base selector: [2]int{REG_BASE,ATH_BASE}
	profile *radir.DITProfile
}

/*
NewDIT returns a freshly initialized instance of *[DIT].
*/
func NewDIT(profile *radir.DITProfile) *DIT {
	aths := make(radir.Registrants, 0)
	return &DIT{
		aths: &aths,
		profile: profile,
	}
}

/*
IsZero returns a Boolean value indicative of a nil receiver state.
*/
func (r *DIT) IsZero() bool {
	return r == nil
}

/*
Profile returns the underlying instance of *[radit.DITProfile].
*/
func (r *DIT) Profile() (profile *radir.DITProfile) {
	if !r.IsZero() {
		profile = r.profile
	}

	return
}

/*
Registrants returns the underlying instance of *[radir.Registrants], or
a nil instance if the receiver is not yet initialized.
*/
func (r *DIT) Registrants() (aths *radir.Registrants) {
	if !r.IsZero() {
		aths = r.aths
	}

	return
}

/*
Tree returns the underlying instance of [OIDTree].
*/
func (r *DIT) Tree() (tree OIDTree) {
	if !r.IsZero() {
		tree = r.tree
	}

	return
}

/*
LDIF returns a single string value containing all LDIF entries present
within the receiver instance.

Note that the output should be verified through use of the [go-ldap/ldif.Parse]
function.

[go-ldap/ldif.Parse]: https://pkg.go.dev/github.com/go-ldap/ldif#Parse
*/
func (r *DIT) LDIF() (l string) {
	for i := 0; i < 3; i++ {
		if r.tree[i] != nil {
			l += r.tree[i].LDIF(2)
		}
	}

	return
}

/*
ITUT returns the ITU-T *[Registration] instance. If the instance is nil,
it is initialized and stored prior to return.
*/
func (r *DIT) ITUT() (root *radir.Registration) {
	if root = r.tree[0]; root.IsZero() {
		r.tree[0] = r.profile.NewRegistration(true)
		r.tree[0].X680().SetN(`0`)
		r.tree[0].X680().SetIRI(`/ITU-T`)
		r.tree[0].X680().SetASN1Notation(`{itu-t(0)}`)
		r.tree[0].X660().SetUnicodeValue(`ITU-T`)
		r.tree[0].X680().SetNameAndNumberForm(`itu-t(0)`)
		r.tree[0].SetDN(`n=0,` + r.profile.RegistrationBase())
		root = r.tree[0]
	}

	return
}

/*
ISO returns the ISO *[Registration] instance. If the instance is nil, it
is initialized and stored prior to return.
*/
func (r *DIT) ISO() (root *radir.Registration) {
	if root = r.tree[1]; root.IsZero() {
		r.tree[1] = r.profile.NewRegistration(true)
		r.tree[1].X680().SetN(`1`)
		r.tree[1].X680().SetIRI(`/ISO`)
		r.tree[1].X680().SetASN1Notation(`{iso(1)}`)
		r.tree[1].X660().SetUnicodeValue(`ISO`)
		r.tree[1].X680().SetNameAndNumberForm(`iso(1)`)
		r.tree[1].SetDN(`n=1,` + r.profile.RegistrationBase())
		root = r.tree[1]
	}

	return
}

/*
JointISOITUT returns the Joint-ISO-ITU-T *[Registration] instance. If
the instance is nil, it is initialized and stored prior to return.
*/
func (r *DIT) JointISOITUT() (root *radir.Registration) {
	if root = r.tree[2]; root.IsZero() {
		r.tree[2] = r.profile.NewRegistration(true)
		r.tree[2].X680().SetN(`2`)
		r.tree[2].X680().SetIRI(`/Joint-ISO-ITU-T`)
		r.tree[2].X680().SetASN1Notation(`{joint-iso-itu-t(2)}`)
		r.tree[2].X660().SetUnicodeValue(`Joint-ISO-ITU-T`)
		r.tree[2].X680().SetNameAndNumberForm(`joint-iso-itu-t(2)`)
		r.tree[2].SetDN(`n=2,` + r.profile.RegistrationBase())
		root = r.tree[2]
	}

	return
}

/*
Root returns the root *[Registration] instance associated with the input
integer value, which must be 0, 1 or 2. A nil instance is returned if the
input value is anything other than those values.  This method is merely
a convenient programmatic alternative to explicit calls of the named root
context (e.g.: input of 1 equals [DIT.ISO] call).
*/
func (r *DIT) Root(n int) (root *radir.Registration) {
	if 0 <= n && n <= 2 {
		funk := []func() *radir.Registration{
			r.ITUT,
			r.ISO,
			r.JointISOITUT,
		}

		if r.tree[n].IsZero() {
			r.tree[n] = funk[n]()
		}
		root = r.tree[n]
	}

	return
}

/*
Prime the root number form (n) within receiver instance using a series
of string instances. n MUST be 0, 1 or 2.

The correct syntax for the input string values is the ITU-T Rec. X.680
ObjectIdentifierValue form (or "ASN.1 Notation").  For example, for the
OID "1.3.6.1.4", the proper form would appear as:

  {iso(1) identified-organization(3) dod(6) internet(1) private(4)}
*/
func (r *DIT) Prime(n int, nodes ...string) {
	if 0 <= n && n <= 2 {
		root := r.Root(n)
		for _, node := range nodes {
			root.Allocate(node)
		}
	}
}

/*
LoadCSV returns an error following an attempt to process the input *[csv.Reader]
instance using the input closure instance. The result is


 a general-use method for loading Comma-Separated Value data
*/
func (r *DIT) LoadCSV(reader *csv.Reader, closure func() any) (err error) {
	if reader == nil {
		err = errors.New("CSV reader is nil")
		return
	} else if closure == nil {
		err = errors.New("closure is nil")
		return
	}

	out := closure()
	switch out.(type) {
	case *radir.Registrations:
		//for i := 0; i < tv.Len(); i++ {
		//	reg := tv.Index(i)
		//	n, _ := reg.Root()
		//	r.Root(n).Put
		//}
		if !r.IsZero() {
			//r.Root(n).Allocate
		}
	case *radir.Registrants:
		if !r.IsZero() {
		}
	default:
		err = errors.New("Return value is neither *radir.Registration nor *radir.Registrant")
	}

	return
}

/*
Print will write the structure of the receiver instance, including all of
its descendants, to STDOUT.
*/
/*
func (r Node) Print(level int) {
	if &r == nil {
		return
	}

	if !isNumber(r.Name) {
		fmt.Printf("%s%s (%s)\n", repeat("  ", level), r.N, r.Name)
	} else {
		fmt.Printf("%s%s\n", repeat("  ", level), r.N)
	}

	for _, child := range r.Children {
		child.print(level + 1)
	}
}
*/
