package iso

/*
pen.go handles the processing and storage of IANA's Private Enterprise
Numbers (PEN) Registry.
*/

import (
	"fmt"

	"github.com/oid-directory/go-radir"
	"github.com/oid-directory/go-radit/internal/common"
)

const (
	enterpriseRDNSeq = `n=1,n=4,n=1,n=6,n=3,n=1`
	entASNPfx        = `{iso(1) identified-organization(3) dod(6) internet(1) private(4) enterprise(1)`
	entDotPfx        = `1.3.6.1.4.1`
	entIRIPfx        = `/ISO/Identified-Organization/6/1/4/1/`
)

/*
penRegistry facilitates storage and interaction with any number of [PEN]
instances previously parsed from IANA's PEN Registry.
*/
type penRegistry struct {
	Numbers   []pen
	*common.DIT
}

/*
pen, or Private Enterprise Number, implements any single registered
enterprise number.
*/
type pen struct {
	Decimal int
	Name    string
	Contact string
	Email   string
}

/*
IsZero returns a Boolean value indicative of a nil receiver state.
*/
func (r *penRegistry) IsZero() bool {
	return r == nil
}

/*
Unmarshal returns instances of [radir.Registrations], [radir.Registrants]
and an error.

The [radir.Registrants] instance will be zero unless the underlying
*[radir.DITProfile] instance operates under the terms of the "Dedicated
Registrants Policy".
*/
func (r *penRegistry) unmarshal() (err error) {
	prof := r.DIT.Profile()

	parent := r.DIT.ISO().Walk(entDotPfx)
	if parent.IsZero() {
		err = mkerr("Missing 1.3.6.1.4.1 parent; DIT must be primed before use")
		return
	}

	dnFunc := radir.DotNotToDN3D
	if prof.Model() == radir.TwoDimensional {
		dnFunc = radir.DotNotToDN2D
	}

	if parent.X680().ASN1Notation() == "" {
		parent.X680().SetASN1Notation(`{`+entASNPfx+`}`)
	}

	for _, ent := range r.Numbers {
		if &ent == nil {
			continue
		}

		child := parent.NewChild(itoa(ent.Decimal), ``)
		child.SetDN(child.X680().DotNotation(), dnFunc)
		if err = ent.handleRegistrant(child, r.DIT); err != nil {
			break
		}

		if child.X680().N() == `56521` {
			// Load Jesse Coretta's registrations	
			r.loadJesseCoretta()
		}

		r.Numbers = r.Numbers[1:]
	}

	return
}

func (r *penRegistry) loadJesseCoretta() {
	for _, j := range JesseOID {
		r.DIT.ISO().Allocate(j)
	}
}

func (r pen) handleRegistrant(child *radir.Registration, dit *common.DIT) (err error) {

	if dit.Profile().Dedicated() {
	        // Process DEDICATED registrants
	        athy := dit.Profile().NewRegistrant()
	        athy.SetDN(radir.RegistrantDNGenerator)
	        child.X660().SetCurrentAuthorities(athy.DN())
	        dit.Registrants().Push(athy)
	
	        for _, strukt := range []struct {
	                Field string
	                Func  func(...any) error
	        }{
	                {r.Name, athy.CurrentAuthority().SetO},
	                {r.Name, athy.SetDescription},
	                {r.Name, child.SetDescription},
	                {r.Contact, athy.CurrentAuthority().SetCN},
	                {r.Email, athy.CurrentAuthority().SetEmail},
	        } {
	                if strukt.Field != `---none---` && strukt.Field != "" {
	                        if err = strukt.Func(strukt.Field); err != nil {
	                                break
	                        }
	                }
	        }
	} else if dit.Profile().Combined() {
        	// Process COMBINED registrants
        	for _, strukt := range []struct {
        	        Field string
        	        Func  func(...any) error
        	}{
        	        {r.Name, child.X660().CombinedCurrentAuthority().SetO},
        	        {r.Name, child.SetDescription},
        	        {r.Contact, child.X660().CombinedCurrentAuthority().SetCN},
        	        {r.Email, child.X660().CombinedCurrentAuthority().SetEmail},
        	} {
        	        if strukt.Field != `---none---` && strukt.Field != "" {
        	                if err = strukt.Func(strukt.Field); err != nil {
        	                        break
        	                }
        	        }
        	}
	}

	return
}

/*
LoadPENRegistry returns an error following an attempt to parse the input
filename, which is expected to refer to an UNMODIFIED copy of IANA's
[PEN Numbers Text Registry].

Be advised: the text registry is a LARGE file; do not click on the link
needlessly.

[PEN Numbers Text Registry]: https://www.iana.org/assignments/enterprise-numbers.txt
*/
func LoadPENRegistry(r *common.DIT, filename string) error {
	if r.IsZero() {
		return nilInstanceErr
	}

	f, err := open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := newScan(f)

	// TODO :: instead of skipping these lines
	// we should use them as seeding for new
	// Registration instances.
	skipLines := 16
	for i := 0; i < skipLines; i++ {
		scanner.Scan()
	}

	var (
		ents *penRegistry = &penRegistry{
			Numbers: make([]pen, 0),
			DIT:	 r,
		}
		ent pen
	)

	for scanner.Scan() {
		if line := scanner.Text(); trimS(line) != "" {
			switch {
			case ent.Decimal == -1:
				fmt.Sscanf(line, "%d", &ent.Decimal)
				continue
			case ent.Name == "":
				ent.Name = trimS(line)
				continue
			case ent.Contact == "":
				ent.Contact = trimS(line)
				continue
			case ent.Email == "":
				ent.Email = trimS(line)
			}
		}

		if ent.Decimal != -1 {
			ents.Numbers = append(ents.Numbers, ent)
			ent = pen{Decimal: -1}
		}
	}

	if err = scanner.Err(); err == nil {
		err = ents.unmarshal()
	}

	return err
}
