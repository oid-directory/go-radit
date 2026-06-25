package radit

import (
	"bytes"
	"errors"

	"github.com/oid-directory/go-radir"
	"github.com/oid-directory/go-radit/internal/common"
	"github.com/oid-directory/go-radit/internal/iso"
	"github.com/oid-directory/go-radit/internal/itu"
	"github.com/oid-directory/go-radit/internal/jii"
)

type RADIT struct {
	dit *common.DIT
}

func New(cfg *radir.DITProfile) (r *RADIT) {
	if !cfg.IsZero() {
		r = &RADIT{dit: common.NewDIT(cfg)}
	}

	return
}

func (r *RADIT) IsZero() bool {
	return r == nil
}

/*
PrimeITUT loads the receiver with preliminary *[radit.Registration]
instances which belong to the "itu-t" root.
*/
func (r *RADIT) PrimeITUT() {
	if !r.IsZero() {
		r.dit.Prime(0, itu.Tree...)
	}
}

/*
PrimeISO loads the receiver with preliminary *[radit.Registration]
instances which belong to the "iso" root.
*/
func (r *RADIT) PrimeISO() {
	if !r.IsZero() {
		r.dit.Prime(1, iso.Tree...)
	}
}

/*
PrimeJointISOITUT loads the receiver with preliminary *[radit.Registration]
instances which belong to the "joint-iso-itu-t" root.
*/
func (r *RADIT) PrimeJointISOITUT() {
	if !r.IsZero() {
		r.dit.Prime(2, jii.Tree...)
	}
}

/*
ImportList specifies the key name and file path for each of the desired
registries to be imported, or loaded, into the receiver instance.

Valid key names are as follows, and must be case-folded as shown.

  - "smifile" specifies the full path and filename of IANA's SMI registry XML file
  - "ldapfile" specifies the full path and filename of IANA's LDAP registry XML file
  - "penfile" specifies the full path and filename of IANA's PEN numbers TXT file
*/
type ImportList map[string]string

/*
Import returns an error following an attempt to load the contents specified
within the input [ImportList] instance into the receiver instance.
*/
func (r *RADIT) Import(imp ImportList) (err error) {
	if imp == nil || len(imp) == 0 {
		err = errors.New("ImportList instance is nil, aborting import")
		return
	}

	type index struct {
		file string
		funk func(*common.DIT, string) error
	}

	indices := []index{
		{`smifile`, iso.LoadSMIRegistry},
		{`ldapfile`, iso.LoadSMIRegistry},
		{`penfile`, iso.LoadPENRegistry},
	}

	for i := 0; i < len(indices) && err == nil; i++ {
		if file, specified := imp[indices[i].file]; specified {
			err = indices[i].funk(r.dit, file)
		}
	}

	return
}

/*
Write returns an instance of *[bytes.Buffer] containing LDIF content present
within the receive instance.

Use of this method is pretty costly, but is intended for a "once-in-a-lifetime
context" to seed a new directory tree with entries. Keep in mind that OIDs rarely
change.

See _examples/main.go for a demonstration of this method.
*/
func (r *RADIT) Write(sortByNumberForm, spatialXY, subentries bool) (buf *bytes.Buffer) {

	if sortByNumberForm {
		// sort the ENTIRE root by number form magnitude
		r.dit.ITUT().SortByNumberForm(sortByNumberForm)
		r.dit.ISO().SortByNumberForm(sortByNumberForm)
		r.dit.JointISOITUT().SortByNumberForm(sortByNumberForm)
	}

	if spatialXY {
		// Order ALL registrations according
		// to number form along X and Y axes.
		r.dit.ITUT().SetXAxes(spatialXY)
		r.dit.ITUT().SetYAxes(spatialXY)
		r.dit.ISO().SetXAxes(spatialXY)
		r.dit.ISO().SetYAxes(spatialXY)
		r.dit.JointISOITUT().SetXAxes(spatialXY)
		r.dit.JointISOITUT().SetYAxes(spatialXY)
	}

	// Finally, dump the content to the byte buffer
	buf = new(bytes.Buffer)
	buf.WriteString(r.dit.ITUT().LDIF(2, subentries))
	buf.WriteString(r.dit.ISO().LDIF(2, subentries))
	buf.WriteString(r.dit.JointISOITUT().LDIF(2, subentries))

	if r.dit.Profile().Dedicated() {
		// DEDICATED registrants are in use; include in buffer.
		buf.WriteString(r.dit.Registrants().LDIF())
	}

	return
}
