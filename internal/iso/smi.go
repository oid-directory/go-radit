package iso

import (
	"encoding/xml"
	"fmt"

	"github.com/oid-directory/go-radir"
	"github.com/oid-directory/go-radit/internal/common"
)

/*
SMINumbers implements the top-level structure of the IANA SMI Numbers registry.
*/
type smiRegistry struct {
	XMLName     xml.Name   `xml:"registry"`
	XMLNS       string     `xml:"xmlns,attr"`
	Title       string     `xml:"title"`
	ID          string     `xml:"id,attr"`
	Updated     string     `xml:"updated"`
	Note        []inner    `xml:"note"`
	People      []person   `xml:"people>person"`
	Registries  registries `xml:"registry"`
	*common.DIT `xml:"-"`
	people      map[string]*radir.Registrant
}

/*
registry implements a recursive OID registry structure in which zero or more
OIDs may be registered. Instances of this type are populated through SMI
Numbers registry processing.
*/
type registry struct {
	Description string     `xml:"description"`
	ID          string     `xml:"id,attr"`
	Title       string     `xml:"title"`
	Updated     string     `xml:"updated"`
	Rule        inner      `xml:"registration_rule"`
	Expert      inner      `xml:"expert"`
	Note        []inner    `xml:"note"`
	XRef        []xref     `xml:"xref"`
	Registries  registries `xml:"registry"`
	Records     records    `xml:"record"`
	Footnote    []footnote `xml:"footnote"`
	smireg      *smiRegistry
	experts     []*radir.Registrant
}

/*
inner is a common XML unmarshaling type which is meant to contain data
related to "registration_rule", "expert" and "note" SMI Numbers XML elements.
*/
type inner struct {
	Text     string	`xml:",innerxml"`
	Title    string `xml:"title,attr"`
	XRef     []xref `xml:"xref"`
        FullText string `xml:"-"`
}

/*
Registries implements the slice form of SMI Numbers *[Registry] instances.
*/
type registries []*registry

/*
footnote implements the "footnote" SMI Numbers XML element.
*/
type footnote struct {
	Anchor string `xml:"anchor,attr,omitempty"`
	Text   string `xml:",innerxml"`
}

/*
person implements the "person" SMI Numbers XML element.
*/
type person struct {
	XMLName xml.Name `xml:"person"`
	ID      string   `xml:"id,attr"`
	Name    string   `xml:"name"`
	URI     string   `xml:"uri"`
	Updated string   `xml:"updated"`
	RegID   string   `xml:"-"`
}

/*
record implements a non-recursive OID list found within a *[registry]
instance.
*/
type record struct {
	XMLName     xml.Name `xml:"record"`
	Date        string   `xml:"date,attr,omitempty"`
	Value       string   `xml:"value"`
	Name        string   `xml:"name"`
	Description string   `xml:"description"`
	Recommended string   `xml:"recommended"`
	XRef        []xref   `xml:"xref"`
	*common.DIT `xml:"-"`
}

/*
records implements the slice form of SMI Numbers [record] instances.
*/
type records []record

/*
xref implements the "xref" SMI Numbers XML element, which is used to
house references to documents, sites and other resources relevant to
the bearer.
*/
type xref struct {
	XMLName xml.Name `xml:"xref"`
	Type    string   `xml:"type,attr"`
	Data    string   `xml:"data,attr,omitempty"`
	Content string   `xml:",chardata"`
}

/*
Obsolete returns a Boolean value indicative of perceived obsolescence on
part of the bearer. Obsolescence is determined by the presence of an
"xref" data value of one (1), or an explicit "obsolete" mention by way
of an "xref.text" content value.
*/
func (r record) obsolete() bool {
	for _, xr := range r.XRef {
		switch xr.Type {
		case `note`:
			return xr.Data == `1`
		case `text`:
			return lc(xr.Content) == `obsolete`
		}
	}

	return false
}

func (r record) processValue() (dot, number, rangeTerminus string, err error) {
	if r.IsZero() {
		err = mkerr("Empty record")
		return
	}

	switch {
	case hasSfx(r.Value, ` and up`):
		// Seems to be an INFINITE range
		number = r.Value[:len(r.Value)-7]
		rangeTerminus = "-1"
	case idxr(r.Value, '-') != -1:
		// Seems to be a FINITE range
		sp := split(r.Value, `-`)
		number = sp[0]
		rangeTerminus = sp[1]
	case common.IsNumber(r.Value):
		number = r.Value
		// ok as-is
	case radir.IsNumericOID(r.Value):
		// in some cases, particularly in the ldap-parameters.xml, a
		// full OID is present where we normally find number form
		// values. Thus, we'll chop off the leaf and discard the rest.
		sp := split(r.Value, `.`)
		number = sp[len(sp)-1]
		dot = r.Value
	default:
		err = mkerr("Bad value; not a range, number form or dotNotation: " + r.Value)
	}

	return
}

func (r *record) processIdentifier(dot string) (identifier string) {
	if !common.IsNumber(r.Name) {
		if identifier = r.Name; identifier == "" {
			var found bool
			identifier, found = patchMissingName(dot, r.Value)
			if !found && len(r.Description) > 0 {
				identifier = r.Description
			}
		}

		if identifier = legalizeIdentifier(identifier); !radir.IsIdentifier(identifier) {
			if alt := legalizeIdentifier(r.Description); radir.IsIdentifier(alt) {
				identifier = alt
			} else {
				identifier = ``
			}
		}
	} else {
		r.Name = ""
	}

	return
}

/*
NumRegistries returns the integer number of [registry] instances found
within the receiver instance.
*/
func (r registries) NumRegistries() int {
	return len(r)
}

/*
NumRegistries returns the integer number of [registry] instances found
within the receiver instance.
*/
func (r smiRegistry) NumRegistries() int {
	return len(r.Registries)
}

/*
NumRecords returns the integer number of [record] instances found within
the receiver instance.
*/
func (r registry) NumRecords() int {
	return len(r.Records)
}

func errNotEoF(err error) (notEof bool) {
	if err != nil {
		notEof = err != eof
	}

	return
}

/*
UnmarshalXML returns an error following an attempt to decode the receiver
instance. This method is implemented to help in the decoding process of
<note></note> contents which contain nested XML elements -- such as xref
and expert -- and to clean up the raw text by removing those elements
that have already been captured.
*/
func (n *inner) unmarshalXML(d *xml.Decoder, start xml.StartElement) (err error) {
	type noteAlias inner
	nn := noteAlias(*n)
	if err = d.DecodeElement(&nn, &start); errNotEoF(err) {
		return
	}

	// Extract the full text content with XML tags and xrefs
	b := newBuilder()
	decoder := xml.NewDecoder(newReader(nn.Text))
	for {
		var token xml.Token
		if token, err = decoder.Token(); err != nil {
			break
		}

		switch t := token.(type) {
		case xml.CharData:
			b.Write(t)
		case xml.StartElement:
			if t.Name.Local == "xref" {
				for _, attr := range t.Attr {
					if attr.Name.Local == "data" {
						b.WriteString(attr.Value)
					}
				}
				// Skip the content inside the xref element
				decoder.Skip()
			}
		case xml.EndElement:
			if t.Name.Local == "xref" {
				// Do nothing, just skip the end element
			}
		}
	}

	if !errNotEoF(err) {
		err = nil
	}

	n.FullText = b.String()
	n.XRef = nn.XRef

	return
}

func descriptionAndOID(descr string) (desc, dot string) {
	if descr != "" {
		desc = descr
		for _, ch := range []string{
			`[`, `]`,
			`(`, `)`,
		} {
			desc = trimS(rplc(desc, ch, ``))
		}

		sp := split(desc, ` `)
		for i := 0; i < len(sp); i++ {
			sp[i] = trimR(sp[i], `.`)
			if radir.IsNumericOID(sp[i]) {
				dot = sp[i]
			} else if ds := split(sp[i], `.`); len(ds) > 0 {
				desc = sp[i]
			}

			//if len(dot) > 0 {
			//	break
			//}
		}
		dot = trimR(dot,`.`)
	}

	return
}

func (r records) unmarshal(smi *smiRegistry, parent *radir.Registration) {
	dedi := smi.DIT.Profile().Dedicated()
	comb := smi.DIT.Profile().Combined()

	for i := 0; i < len(r); i++ {
		rec := r[i]

		_, number, rangeTerm, err := rec.processValue()
		if err != nil || rec.obsolete() {
			continue
		}

		identifier := rec.processIdentifier(parent.X680().DotNotation())
		if len(identifier) == 0 {
			identifier = number
		}

		if child := parent.Children().Get(number); child.IsZero() {
			child = parent.NewChild(number, identifier)
			if rangeTerm != "" {
				child.Supplement().SetRange(rangeTerm)
			}

			if ctns(child.X680().Identifier(), `.`) {
				child.X680().SetIdentifier(identifier)
			}

        		// Process the XRef into one of a few possible
        		// forms, such as uri, rfc, person, et al.
        		for _, xr := range rec.XRef {
        		        xr.process(child,smi)
                        	if xr.Type == "person" {
                        	        if athy, found := smi.people[xr.Data]; found {
                        	                cath := athy.CurrentAuthority()
                        	                if dedi {
                        	                        child.X660().SetCurrentAuthorities(athy.DN())
                        	                } else if comb {
							coauth := child.X660().CombinedCurrentAuthority()
                        	                        coauth.SetEmail(cath.Email())
                        	                        coauth.SetCN(cath.CN())
                        	                        coauth.SetO(cath.O())
                        	                        child.SetDescription(athy.Description())
                        	                }
                        	        }
                        	}
        		}
		}
	}

	// Sort children
	parent.Children().SortByNumberForm()

	return
}

func (r *registry) unmarshalRecords() (err error) {
	if r.IsZero() {
		return
	}

	var srcinfo string = r.Description
	if r.Title != "" && srcinfo == "" {
		// If title is used and NOT description
		// then swap the values
		srcinfo = r.Title
	}

	desc, oid := descriptionAndOID(srcinfo)
	if oid == "" {
		// Some registries do not contain any
		// OIDs because they're stored in a
		// separately downloaded file. In this
		// case, just exit without errors.
		return
	}

	var (
		sp     []string = split(oid, `.`)
		path   []string
		ident  string
		parent *radir.Registration = r.smireg.DIT.ISO().Allocate(oid)
	)

        // Process experts into registrant data
        //r.processExperts(parent)
	for _, expert :=range r.experts {
		parent.X660().SetCurrentAuthorities(expert.DN())
	}

        // Process the XRef into one of a few possible
        // forms, such as uri, rfc, person, et al.
        for _, xr := range r.XRef {
                xr.process(parent,r.smireg)
        }

        for _, n := range r.Note {
                clean := trimS(n.Text)
                clean = trimL(clean, `\n`)
                clean = trimR(clean, `\n`)
                clean = trimR(clean, `\n`)

		start := xml.StartElement{Name:xml.Name{Space:"", Local:"note"}, Attr:[]xml.Attr{}}
		dec := xml.NewDecoder(newReader(n.Text))
		if err = n.unmarshalXML(dec,start); errNotEoF(err) {
			return
		}

		_ = fmt.Sprintf("---")

                if len(n.FullText) > 0 && clean != "" && !hasPfx(clean,`<`) {
                        // Don't write pure xml content as reg info.
                        inf := common.CondenseWHSP(common.RemoveNL(n.FullText))
                        parent.Supplement().SetInfo(inf)
                }

		// Output the XRef if needed
		for _, xr := range n.XRef {
			if xr.Type == "uri" && n.Title != "" {
				xr.Content = n.Title
			} else if xr.Type == "registry" {
				xr.Type = "uri"
                		xr.Data = common.IANAAssignmentsPrefix + xr.Data
			}
			xr.process(parent, r.smireg)
		}
	}

	dot := parent.X680().DotNotation()
	if dot != oid {
		err = mkerr("Allocation error: " +dot+" != "+oid)
		return
	}

	if dp := split(desc, `.`); len(dp) == len(sp) {
		path = dp
		ident = dp[len(dp)-1]
	} else if len(path) > 0 {
		if path[len(path)-1] != parent.X680().Identifier() {
			ident = path[len(path)-1]
			path[len(path)-1] = parent.X680().Identifier()
		}
	}

	if parent.X680().Identifier() == "" {
		parent.X680().SetIdentifier(ident)
	}

	parent.X680().SetASN1Notation(buildASN1Not(sp, path))
	r.Records.unmarshal(r.smireg,parent)

	return
}

func (r xref) processDataUsers(reg *radir.Registration, smi *smiRegistry) {
        if r.Data == "" {
                return
        }

        switch r.Type {
        case `rfc`,`draft`:
                reg.Supplement().SetURI(common.RFCURIPrefix +
			r.Data + " " + uc(r.Data))
        case `rfc-errata`:
                reg.Supplement().SetURI(common.RFCErrataPrefix +
			r.Data + " Errata ID " + r.Data)
        case `uri`:
		if r.Content != "" {
			// There may be a link label ...
			r.Data += " " + r.Content
                	reg.Supplement().SetURI(r.Data)
		}
        case `note`:
                // TODO :: see if there are other values of significance.
                if r.Data == `1` {
                        // This seems to indicate obsolescence and correlates
                        // to <footnote anchor="1">...</footnote>
                        reg.Supplement().SetStatus(`OBSOLETE`)
                } else {
                        reg.Supplement().SetInfo(r.Data)
                }
        	//case `person`:
                //switch {
                //case smi.DIT.Profile().Dedicated():
                //        var (
                //                ok   bool
                //                ath  *radir.Registrant
                //                pers person
                //        )

                //        if ath = smi.DIT.Registrants().Get(r.Data); !ath.IsZero() {
                //                // We already manufactured a Registrant
                //                // instance, so just grab its DN
                //                reg.X660().SetCurrentAuthorities(ath.DN())
                //        } else if pers, ok = smi.people[r.Data]; ok {
                //                // We know this is a legitimate Registrant,
                //                // but we have not encountered it yet. Let's
                //                // create the radir.Registrant instance now.
                //                ath = smi.DIT.Profile().NewRegistrant() // init
		//		ath.SetDN(radir.RegistrantDNGenerator)
                //                dn := ath.DN()
                //                ath.SetDN(dn)
                //                reg.X660().SetCurrentAuthorities(dn)
                //                pers.setAttributes(reg,ath)

                //                // No need to hold onto the
                //                // people[r.Data] k/v ...
                //                delete(smi.people, r.Data)

                //                // ... because we store the
                //                // final form here:
                //                smi.DIT.Registrants().Push(ath)
                //        }

                //case reg.Combined():
                //        if pers, ok := smi.people[r.Data]; ok {
                //                pers.setAttributes(reg, nil)
                //        }
                //}
        }
}

func (r xref) processContentUsers(reg *radir.Registration) {
        if r.Content==`Not Defined ?` || r.Content == "" {
                return
        }

        switch r.Type {
        case `registry`:
                reg.Supplement().SetInfo(common.RemoveNL(r.Content))
        case `text`:
                value := common.CondenseWHSP(common.RemoveNL(r.Content))
                if eq(value, `obsolete`) {
                        reg.Supplement().SetStatus(`OBSOLETE`)
                } else {
                        reg.Supplement().SetInfo(value)
                }
        }
}

func (r xref) process(reg *radir.Registration, smi *smiRegistry) {
        switch r.Type {
        case `registry`,`text`:
                r.processContentUsers(reg)
        case `rfc`,`rfc-errata`,`draft`,`note`,`uri`,`person`:
                r.processDataUsers(reg, smi)
        }
}

//func (r person) setAttributes(reg *radir.Registration, ath *radir.Registrant) {
//        var base *radir.CurrentAuthority
//        if reg.Dedicated() {
//                base = ath.CurrentAuthority()
//        } else if reg.Combined() {
//                base = reg.X660().CombinedCurrentAuthority()
//        } else {
//                return // no registrants policy?
//        }
//
//        if len(r.URI) > 0 {
//                if hasPfx(r.URI,`mailto:`) {
//                        // URI is an email address. We'll strip-off
//                        // the mailto: and replace amp with com-at.
//                        uri := r.URI[7:]
//                        uri = rplc(uri,`&`,`@`)
//                        uri = rplc(uri,`%25`,`%`)
//                        base.SetEmail(uri)
//                } else {
//                        // Sometimes a URI is just a URI.
//                        base.SetURI(r.URI)
//                }
//        }
//
//        if len(r.Name) > 0 {
//                // TODO :: this may need to be expanded if there are
//                // other official body "names" besides IANA (not
//                // individual people) found in the SMI registries.
//                if r.Name == `IANA` {
//                        base.SetO(r.Name)
//                } else {
//                        // Assume its a person's name.
//                        base.SetCN(r.Name)
//                }
//        }
//}

/*
IsZero returns a Boolean value indicative of a nil receiver state.
*/
func (r *record) IsZero() bool { return r == nil }

/*
IsZero returns a Boolean value indicative of a nil receiver state.
*/
func (r *registry) IsZero() bool { return r == nil }

func (r *registry) unmarshal() (err error) {
	if !r.IsZero() {
		if err = r.unmarshalRecords(); err != nil {
			return
		}

		for _, subregi := range r.Registries {
			subregi.smireg = r.smireg
			if err = subregi.unmarshal(); err != nil {
				break
			}
		}
	}

	return
}

/*
Unmarshal returns an error following an attempt to write the contents of
the receiver into instances of *[radir.Registration] and (if appropriate)
*[radir.Registrant].
*/
func (r *smiRegistry) unmarshal() (err error) {
	if !r.IsZero() {
		r.gatherRegistrants()

		// Process all registries, descending into
		// Records and (sub) Registries as needed.
		for _, regi := range r.Registries {
			regi.smireg = r
			if k, found := missingRegistryURNs[regi.ID]; found {
				regi.Description = missingRegistryURNs[k]
			}

			if err = regi.unmarshal(); err != nil {
				break
			}
		}
	}

	return
}

func (r *registry) gatherExperts() {
	sp := split(r.Expert.Text,`,`)

        for _, sl := range sp {
                var desc string
                var orig string = trimS(sl)
                sl = trimS(common.CondenseWHSP(sl))
                if idx := sidx(sl,` (`); idx != -1 {
                        desc = `IANA, ` + trimR(sl[idx+2:],`)`)
                        sl = sl[:idx]
                } else {
                        desc = `IANA`
                }

                if sl != "" {
			if _, found := r.smireg.people[orig]; !found {
                		athy := r.smireg.DIT.Profile().NewRegistrant()
                		athy.SetDN(radir.RegistrantDNGenerator)
                		athy.CurrentAuthority().SetCN(sl)
                		athy.CurrentAuthority().SetO(desc)
                		r.smireg.people[orig] = athy
				r.smireg.DIT.Registrants().Push(athy)
				r.experts = append(r.experts, athy)
			}
		}
        }

	for i := 0; i < len(r.Registries); i++ {
		r.Registries[i].smireg = r.smireg
		r.Registries[i].gatherExperts()
	}
}

func (r *smiRegistry) gatherRegistrants() {
        // Process and load all known <person>
        // elements into temporary storage ...
        for _, person := range r.People {
		if _, found := r.people[person.ID]; !found {
			regi := r.DIT.Profile().NewRegistrant()

                	regi.SetDN(radir.RegistrantDNGenerator)
                	regi.CurrentAuthority().SetCN(person.Name)
                	regi.SetDescription(person.Name)

		        if uri := person.URI; len(uri) > 0 {
		                if hasPfx(uri,`mailto:`) {
		                        // URI is an email address. We'll strip-off
		                        // the mailto: and replace amp with com-at.
		                        uri = uri[7:]
		                        uri = rplc(uri,`&`,`@`)
		                        uri = rplc(uri,`%25`,`%`)
		                        regi.CurrentAuthority().SetEmail(uri)
		                } else {
		                        // Sometimes a URI is just a URI.
		                        regi.CurrentAuthority().SetURI(uri)
		                }
		        }

		        if len(person.Name) > 0 {
		                // TODO :: this may need to be expanded if there are
		                // other official body "names" besides IANA (not
		                // individual people) found in the SMI registries.
		                if person.Name == `IANA` {
		                        regi.CurrentAuthority().SetO(person.Name)
		                } else {
		                        // Assume its a person's name.
		                        regi.CurrentAuthority().SetCN(person.Name)
		                }
		        }

                	r.people[person.ID] = regi
			r.DIT.Registrants().Push(regi)
		}
        }

	for _, regi := range r.Registries {
		regi.smireg = r
		regi.gatherExperts()
	}
}

/*
LoadSMIRegistry returns an error following an attempt to parse the input
filename, which is expected to refer to an UNMODIFIED copy of IANA's
[SMI-Numbers XML registry].

[SMI-Numbers XML registry]: https://www.iana.org/assignments/smi-numbers/smi-numbers.xml
*/
func LoadSMIRegistry(r *common.DIT, filename string) (err error) {
	var (
		content []byte
		smi     smiRegistry
	)

	smi.people = make(map[string]*radir.Registrant, 0)

	if content, err = common.ReadBytes(filename); err == nil {
		if err = xml.Unmarshal(content, &smi); !errNotEoF(err) {
			smi.DIT = r
			err = smi.unmarshal()
		}
	}

	return
}

/*
legalizeIdentifier will attempt to take a record.Name value, such
as IEEE802.4, which is ILLEGAL as an X.680 identifier (name form),
and replace or augment the value for compliance.
*/
func legalizeIdentifier(in string) (out string) {
	if radir.IsIdentifier(in) || len(in) == 0 {
		out = in
		return
	}

	// Make sure this isn't a "re-allocated" OID
	if ctns(lc(in), `retained by`) {
		if cut, found := cutPfx(lc(in), `retained by`); found {
			if in = trimS(cut); radir.IsIdentifier(in) {
				out = in
				return
			}
		}
	}

	if lup, found := i2i[in]; found {
		// i2i map lookup succeeded
		out = lup
		return
	}

	// There is a chance the value is something like
	// "name (Junk)", where name by itself represents
	// the valid identifier. We need only split it and
	// try each value
	if ins := split(trimS(in), ` `); len(ins) > 1 {
		for _, try := range ins {
			if tried := legalizeIdentifier(try); radir.IsIdentifier(tried) {
				out = tried
				return
			}
		}
	} else if tweaked := rplc(lc(ins[0]), `_`, `-`); radir.IsIdentifier(tweaked) {
		// Some names are valid simply by folding
		// case to lower and swapping underscores
		// with dashes.
		out = tweaked
	} else if tweaked = rplc(lc(ins[0]), `.`, ``); radir.IsIdentifier(tweaked) {
		// If simply removing dots and folding the
		// case to lower revealed a valid identifier
		// then use it and bail out. Note that if the
		// value was a dot OID, it would not have made
		// a valid identifier.
		out = tweaked
	}
	return
}

func patchMissingName(oid, leaf string) (name string, found bool) {
	if !common.IsNumber(leaf) {
		return
	}

	if entry, ok := missingRecordNames[oid]; ok {
		name, found = entry[leaf]
	}

	return
}

func buildASN1Not(sp, path []string) (anot string) {
	if len(sp) == len(path) {
		// Here we build-up the ASN1Notation instance
		// using the SMI URN description sequence and
		// the numeric OID -- but only if they are of
		// equal lengths.
		if len(path) > 1 {
			if path[1] == `org` {
				// replace URN "org" with ASN.1
				// "identified-organization".
				path[1] = `identified-organization`
			}
		}

		var slice []string
		for i := 0; i < len(path); i++ {
			if !common.IsNumber(path[i]) {
				// Add name and number form to sequence
				slice = append(slice, path[i]+`(`+sp[i]+`)`)
			} else {
				// Add number form only to sequence
				slice = append(slice, sp[i])
			}
		}

		anot = `{` + join(slice, ` `) + `}`
	}

	return
}

