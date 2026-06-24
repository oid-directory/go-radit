package main

import (
	"fmt"

	"github.com/oid-directory/go-radir"
	"github.com/oid-directory/go-radit"
)

func main() {
	// Specify the registries we want to load. For reference,
	// I downloaded them using the wget commands below.
	//
	// Naturally, update the directory paths below to reference
	// something that actually exists on your system :)
	//
	// $ cd ~/dev/tmp
	// $ wget -O smi.xml https://www.iana.org/assignments/smi-numbers/smi-numbers.xml
	// $ wget -O ldap.xml https://www.iana.org/assignments/ldap-parameters/ldap-parameters.xml
	// $ wget -O pen.txt https://www.iana.org/assignments/enterprise-numbers.txt
	//
	// Finally, execute this main.go file:
	// $ go run main.go > ra.ldif
	//
	// On my crappy little laptop, it took about a minute to generate ra.ldif,
	// which contains 134736 entries as of the time of this writing. Fortunately,
	// this is a "once-in-a-lifetime" action needed only for "priming" a new OID
	// directory implementation, and is not something done regularly.
	imps := radit.ImportList{
		`smifile`:  `/home/jc/dev/tmp/iana.xml`,
		`ldapfile`: `/home/jc/dev/tmp/ldap.xml`,
		`penfile`:  `/home/jc/dev/tmp/pen.txt`,
	}

	// Here, a factory-default RA DIT DUAConfig is
	// ideal for a demonstration.
	cfg := radir.NewFactoryDefaultDUAConfig()

	// Initialize a new instance of *radit.DIT.
	dit := radit.New(cfg.Profile())

	// Prime the "skeleton" of the OID Tree with
	// common registrations.
	dit.PrimeITUT()
	dit.PrimeISO()
	dit.PrimeJointISOITUT()

	// Import specified data (see manifest above)
	if err := dit.Import(imps); err != nil {
		fmt.Println(err)
		return
	}

	// Write data to content (*bytes.Buffer). See the go docs for
	// the meaning of these input bools.
	content := dit.Write(true, true, true)

	// Note that this is about to spit out a BIG LDIF file ... so
	// avoid writing to STDOUT without a file redirect, e.g.:
	// $ go run main.go > ra.ldif
	fmt.Printf("%s", content.String())
}
