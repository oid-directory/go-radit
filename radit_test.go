package radit

import (
	"os"
	"path/filepath"
	"testing"

	_ "embed"

	"github.com/oid-directory/go-radir"
)

//go:embed testdata/iana.xml
var testSMIXML []byte

//go:embed testdata/ldap.xml
var testLDAPXML []byte

//go:embed testdata/pen.txt
var testPENTXT []byte

func TestDITLoad(t *testing.T) {
	tmpDir := t.TempDir()

	// tmp file paths
	smiFile := filepath.Join(tmpDir, "iana.xml")
	ldapFile := filepath.Join(tmpDir, "ldap.xml")
	penFile := filepath.Join(tmpDir, "pen.txt")

	// Write embedded data to the tmp files
	if err := os.WriteFile(smiFile, testSMIXML, 0600); err != nil {
		t.Fatalf("%s failed: unable to write tmp file (%s): %v", t.Name(), smiFile, err)
	}
	if err := os.WriteFile(ldapFile, testLDAPXML, 0600); err != nil {
		t.Fatalf("%s failed: unable to write tmp file (%s): %v", t.Name(), ldapFile, err)
	}
	if err := os.WriteFile(penFile, testPENTXT, 0600); err != nil {
		t.Fatalf("%s failed: unable to write tmp file (%s): %v", t.Name(), penFile, err)
	}

	// Confirm tmp file exists
	for _, file := range []string{
		smiFile, ldapFile, penFile,
	} {
		if _, err := os.Stat(file); err != nil {
			t.Fatalf("%s failed: temp file (%s) missing after write: %v", t.Name(), file, err)
		}
	}

	// Perform data load
	imps := ImportList{
		`smifile`:  smiFile,
		`ldapfile`: ldapFile,
		`penfile`:  penFile,
	}

	cfg := radir.NewFactoryDefaultDUAConfig()
	dit := New(cfg.Profile())

	dit.PrimeITUT()
	dit.PrimeISO()
	dit.PrimeJointISOITUT()

	if err := dit.Import(imps); err != nil {
		t.Fatalf("%s failed: unable to import one or more files: %v", t.Name(), err)
	}

	content := dit.Write(true, true, true)

	want := 747777
	if got := content.Len(); want != got {
		t.Fatalf("%s failed: unexpected byte len; want %d, got %d", t.Name(), want, got)
	}

	t.Logf("%s\n", content.String())

	// Execute and confirm tmp file deletion
	for _, file := range []string{
		smiFile, ldapFile, penFile,
	} {
		if err := os.Remove(file); err != nil {
			t.Fatalf("%s failed: error encountered while deleting tmp file (%s): %v", t.Name(), file, err)
		}

		if _, err := os.Stat(file); !os.IsNotExist(err) {
			t.Fatalf("%s failed: tmp file (%s) still exists after delete (err:%v)", t.Name(), file, err)
		}
	}
}

func TestDITLoad_codecov(_ *testing.T) {
	cfg := radir.NewFactoryDefaultDUAConfig()
	dit := New(cfg.Profile())
	_ = dit.Import(nil)
	_ = dit.Import(ImportList{})
}
