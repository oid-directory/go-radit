package main

import (
	"fmt"

	"github.com/oid-directory/go-radir"
	"github.com/oid-directory/go-radit"
)

func main() {
	// Specify the registries we want to load.
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

	// Write data to content (*bytes.Buffer)
	content := dit.Write(true, true, true)

	// Note that this is a BIG LDIF file ... avoid
	// writing to STDOUT without a file redirect,
	// e.g.: go run main.go > my.ldif
	fmt.Printf("%s", content.String())
}
