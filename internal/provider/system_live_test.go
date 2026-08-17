package provider

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Live CRUD coverage for the system family: users, groups, tunables, the
// certificate stack (CA, certificate, CRL, revoked certificate), authentication
// servers, REST API access list entries and the three system singletons
// (hostname, DNS, timezone). Everything here runs against a real pfSense box and
// is gated by testAccPreCheck, so `go test ./...` without TF_ACC stays hermetic.
//
// SAFETY. This batch reaches further into the box than the firewall, interface
// and routing batches, and the tests run over the very REST API they configure.
// The rules every test below obeys:
//
//   - Users and groups: only brand new `tftest_`-prefixed objects are created and
//     destroyed. `admin`, `root` and the built-in groups are never read-modify-
//     written, so the credentials these tests authenticate with cannot be lost.
//   - Certificates: the CAs and certificates are generated in-process for the run
//     and described as `tftest_`. The box's own webConfigurator certificate (the
//     "GUI default" entry) is never imported, modified or deleted.
//   - REST API access list: the test only ever adds an `allow` entry for a
//     documentation-range network (192.0.2.0/24) with a weight above the two
//     default allow-all entries, which it leaves untouched and asserts are still
//     present. No `deny` entry is ever created, so the management client at
//     10.99.0.2 cannot be locked out of the API mid-test.
//   - Tunables: only `net.inet.ip.redirect`, which controls whether the kernel
//     emits ICMP redirects for routed traffic and cannot affect the management
//     path. The last value the test applies is the box's running value, and the
//     tunable itself is deleted on destroy.
//   - Authentication servers: a throwaway RADIUS server pointing at a
//     documentation-range address with a dummy secret. It is never made the
//     active authentication backend, so local `admin` authentication is
//     unaffected.
//   - Singletons: hostname, DNS and timezone have no create/destroy semantics, so
//     each test captures the box's current value first, applies a reversible
//     change, restores the captured value in a later step, and finally asserts
//     over the API that the box is back where it started. The changes are chosen
//     so management by IP cannot break, and each guard is idempotent: a run that
//     died mid-test is detected and rolled back before the next one starts.
//
// `pfsense_system_package` is the one resource here without live CRUD coverage;
// see TestAccSystemPackageResourceLive for the reason.

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

// testAccLiveObjectExists reports whether an object with keyField == keyValue is
// present on the live box, resolving it by natural key exactly the way the
// resources do.
func testAccLiveObjectExists(pluralPath, keyField, keyValue string) (bool, error) {
	c, err := testAccClient()
	if err != nil {
		return false, fmt.Errorf("building verification client: %w", err)
	}
	_, _, found, err := findByKey(context.Background(), c, pluralPath, keyField, keyValue)
	if err != nil {
		return false, fmt.Errorf("looking up %s with %s %q: %w", pluralPath, keyField, keyValue, err)
	}
	return found, nil
}

// testAccPreCheckLiveObjectAbsent refuses to start when a throwaway object is
// already on the box. The identities are fixed, so a leftover from an aborted run
// would otherwise be adopted and then destroyed by the test — better to stop and
// let a human clean up.
func testAccPreCheckLiveObjectAbsent(t *testing.T, kind, pluralPath, keyField, keyValue string) {
	t.Helper()

	found, err := testAccLiveObjectExists(pluralPath, keyField, keyValue)
	if err != nil {
		t.Fatalf("checking %s %q does not already exist: %v", kind, keyValue, err)
	}
	if found {
		t.Fatalf("%s %q already exists on the box; remove it before running the live test", kind, keyValue)
	}
}

// testAccCheckLiveObjectAbsent verifies an object is gone from the live box.
func testAccCheckLiveObjectAbsent(kind, pluralPath, keyField, keyValue string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		found, err := testAccLiveObjectExists(pluralPath, keyField, keyValue)
		if err != nil {
			return fmt.Errorf("checking %s %q was destroyed: %w", kind, keyValue, err)
		}
		if found {
			return fmt.Errorf("%s %q still exists after destroy", kind, keyValue)
		}
		return nil
	}
}

// testAccCheckLiveObjectsAbsent composes absence checks for a resource and every
// throwaway object its configuration depends on.
func testAccCheckLiveObjectsAbsent(checks ...resource.TestCheckFunc) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, check := range checks {
			if err := check(s); err != nil {
				return err
			}
		}
		return nil
	}
}

// testAccLiveGetObject reads a singleton endpoint directly, bypassing Terraform
// state. The singleton tests use it to capture the box's original settings and to
// prove they were put back.
func testAccLiveGetObject(path string) (map[string]any, error) {
	c, err := testAccClient()
	if err != nil {
		return nil, fmt.Errorf("building verification client: %w", err)
	}
	raw, err := c.Get(context.Background(), path, nil)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return decodeObject(raw)
}

// testAccLiveUpdateObject writes a singleton endpoint directly. Only the
// idempotent singleton guards use it, to roll back a change an aborted run left
// behind before the test starts.
func testAccLiveUpdateObject(path string, payload map[string]any) error {
	c, err := testAccClient()
	if err != nil {
		return fmt.Errorf("building verification client: %w", err)
	}
	if _, err := c.Update(context.Background(), path, applyNow(payload)); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

// testAccHCLStringList renders a []string as an HCL list literal.
func testAccHCLStringList(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, v := range values {
		quoted = append(quoted, fmt.Sprintf("%q", v))
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

// testAccImportStateIDFromAttr builds an import ID out of an attribute the API
// assigned at create time (a certificate refid, say), which the test cannot know
// up front.
func testAccImportStateIDFromAttr(resourceName, attribute string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("%s not found in state", resourceName)
		}
		value, ok := rs.Primary.Attributes[attribute]
		if !ok || value == "" {
			return "", fmt.Errorf("%s: attribute %s is not set", resourceName, attribute)
		}
		return value, nil
	}
}

// ---------------------------------------------------------------------------
// Certificate material
//
// The CA, certificate and CRL tests need real x509 material. It is generated in
// process rather than committed so no private key ever lands in the repository,
// and generated exactly once per test binary so every step of a test renders
// byte-identical configuration (a regenerated key would look like a diff).
// ---------------------------------------------------------------------------

type testAccCertBundle struct {
	// CACert/CAKey sign LeafCert; SelfSignedCert is its own issuer and is used by
	// the certificate test, which runs without a CA on the box.
	CACert         string
	CAKey          string
	LeafCert       string
	LeafKey        string
	SelfSignedCert string
	SelfSignedKey  string
}

var (
	testAccCertOnce   sync.Once
	testAccCertCached testAccCertBundle
	testAccCertErr    error
)

func testAccCertificates(t *testing.T) testAccCertBundle {
	t.Helper()

	testAccCertOnce.Do(func() {
		testAccCertCached, testAccCertErr = testAccGenerateCertBundle()
	})
	if testAccCertErr != nil {
		t.Fatalf("generating test certificate material: %v", testAccCertErr)
	}
	return testAccCertCached
}

// testAccGenerateCertBundle builds a throwaway CA, a leaf signed by it and a
// standalone self-signed certificate.
//
// Serial numbers are deliberately small: pfSense regenerates an internal CRL
// whenever a certificate is revoked on it, and its CRL library rejects serials
// that do not fit in an ASN.1 INT (the 20-byte random serials openssl hands out
// by default make revocation fail with a 500).
func testAccGenerateCertBundle() (testAccCertBundle, error) {
	var out testAccCertBundle

	notBefore := time.Now().Add(-time.Hour)
	notAfter := notBefore.AddDate(10, 0, 0)

	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return out, fmt.Errorf("generating CA key: %w", err)
	}
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "tftest_live_ca"},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caCert, err := testAccEncodeCert(caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return out, fmt.Errorf("generating CA certificate: %w", err)
	}
	caCertParsed, err := testAccParseCert(caCert)
	if err != nil {
		return out, err
	}

	leafKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return out, fmt.Errorf("generating leaf key: %w", err)
	}
	leafTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(4242),
		Subject:      pkix.Name{CommonName: "tftest_live_leaf"},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	leafCert, err := testAccEncodeCert(leafTemplate, caCertParsed, &leafKey.PublicKey, caKey)
	if err != nil {
		return out, fmt.Errorf("generating leaf certificate: %w", err)
	}

	selfKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return out, fmt.Errorf("generating self-signed key: %w", err)
	}
	selfTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(4243),
		Subject:      pkix.Name{CommonName: "tftest_live_cert"},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	selfCert, err := testAccEncodeCert(selfTemplate, selfTemplate, &selfKey.PublicKey, selfKey)
	if err != nil {
		return out, fmt.Errorf("generating self-signed certificate: %w", err)
	}

	caKeyPEM, err := testAccEncodeKey(caKey)
	if err != nil {
		return out, err
	}
	leafKeyPEM, err := testAccEncodeKey(leafKey)
	if err != nil {
		return out, err
	}
	selfKeyPEM, err := testAccEncodeKey(selfKey)
	if err != nil {
		return out, err
	}

	out = testAccCertBundle{
		CACert:         caCert,
		CAKey:          caKeyPEM,
		LeafCert:       leafCert,
		LeafKey:        leafKeyPEM,
		SelfSignedCert: selfCert,
		SelfSignedKey:  selfKeyPEM,
	}
	return out, nil
}

func testAccEncodeCert(template, parent *x509.Certificate, pub *rsa.PublicKey, signer *rsa.PrivateKey) (string, error) {
	der, err := x509.CreateCertificate(rand.Reader, template, parent, pub, signer)
	if err != nil {
		return "", err
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})), nil
}

func testAccParseCert(certPEM string) (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return nil, fmt.Errorf("decoding generated certificate: no PEM block")
	}
	return x509.ParseCertificate(block.Bytes)
}

func testAccEncodeKey(key *rsa.PrivateKey) (string, error) {
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return "", fmt.Errorf("encoding private key: %w", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})), nil
}

// ---------------------------------------------------------------------------
// pfsense_system_user
// ---------------------------------------------------------------------------

// The test only ever creates and destroys its own user. `admin` — the account
// these tests authenticate with — is never touched.
const (
	testAccLiveUserName     = "tftest_live_user"
	testAccLiveUserDescr    = "tftest_live_user"
	testAccLiveUserPassword = "tftest-Live-Passw0rd"
)

// testAccSystemUserLiveConfig renders the user for one step. `expires` and
// `authorizedkeys` are pinned to the empty string the API reports for an account
// that has neither, so the plan settles.
func testAccSystemUserLiveConfig(descr string, disabled bool) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "pfsense_system_user" "live" {
  name           = %q
  password       = %q
  descr          = %q
  disabled       = %t
  expires        = ""
  authorizedkeys = ""
}
`, testAccLiveUserName, testAccLiveUserPassword, descr, disabled)
}

func TestAccSystemUserResourceLive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckLiveObjectAbsent(t, "user", systemUserPlural, "name", testAccLiveUserName)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckLiveObjectAbsent("user", systemUserPlural, "name", testAccLiveUserName),
		Steps: []resource.TestStep{
			{
				Config: testAccSystemUserLiveConfig(testAccLiveUserDescr, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_system_user.live", "id", testAccLiveUserName),
					resource.TestCheckResourceAttr("pfsense_system_user.live", "name", testAccLiveUserName),
					resource.TestCheckResourceAttr("pfsense_system_user.live", "descr", testAccLiveUserDescr),
					resource.TestCheckResourceAttr("pfsense_system_user.live", "disabled", "false"),
					// The API assigns the uid; the resource must report it.
					resource.TestCheckResourceAttrSet("pfsense_system_user.live", "uid"),
				),
			},
			{
				// `name` (the natural key) is unchanged, so this is an in-place
				// update; the description and the disabled flag move.
				Config: testAccSystemUserLiveConfig(testAccLiveUserDescr+" (updated)", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_system_user.live", "id", testAccLiveUserName),
					resource.TestCheckResourceAttr("pfsense_system_user.live", "descr", testAccLiveUserDescr+" (updated)"),
					resource.TestCheckResourceAttr("pfsense_system_user.live", "disabled", "true"),
					resource.TestCheckResourceAttrSet("pfsense_system_user.live", "uid"),
				),
			},
			{
				ResourceName:      "pfsense_system_user.live",
				ImportState:       true,
				ImportStateId:     testAccLiveUserName,
				ImportStateVerify: true,
				// The API hashes the password and never echoes it back, so an
				// imported user has no password to compare against.
				ImportStateVerifyIgnore: []string{"password"},
			},
		},
	})
}

// ---------------------------------------------------------------------------
// pfsense_system_group
// ---------------------------------------------------------------------------

// The built-in `all` and `admins` groups (which carry admin's privileges) are
// never touched; the test creates its own local group. A local-scope group name
// may not exceed 16 characters, so the throwaway name is abbreviated.
const (
	testAccLiveGroupName  = "tftest_grp"
	testAccLiveGroupDescr = "tftest_live_group"
)

// testAccSystemGroupLiveConfig renders the group for one step. `scope` is pinned
// to the API default `local` because the API reports it on every read.
func testAccSystemGroupLiveConfig(description string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "pfsense_system_group" "live" {
  name        = %q
  scope       = "local"
  description = %q
}
`, testAccLiveGroupName, description)
}

func TestAccSystemGroupResourceLive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckLiveObjectAbsent(t, "group", systemGroupPlural, "name", testAccLiveGroupName)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckLiveObjectAbsent("group", systemGroupPlural, "name", testAccLiveGroupName),
		Steps: []resource.TestStep{
			{
				Config: testAccSystemGroupLiveConfig(testAccLiveGroupDescr),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_system_group.live", "id", testAccLiveGroupName),
					resource.TestCheckResourceAttr("pfsense_system_group.live", "name", testAccLiveGroupName),
					resource.TestCheckResourceAttr("pfsense_system_group.live", "scope", "local"),
					resource.TestCheckResourceAttr("pfsense_system_group.live", "description", testAccLiveGroupDescr),
					// The API assigns the gid; the resource must report it.
					resource.TestCheckResourceAttrSet("pfsense_system_group.live", "gid"),
				),
			},
			{
				// `name` (the natural key) is unchanged; only the description moves.
				Config: testAccSystemGroupLiveConfig(testAccLiveGroupDescr + " (updated)"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_system_group.live", "id", testAccLiveGroupName),
					resource.TestCheckResourceAttr("pfsense_system_group.live", "description", testAccLiveGroupDescr+" (updated)"),
					resource.TestCheckResourceAttrSet("pfsense_system_group.live", "gid"),
				),
			},
			{
				ResourceName:      "pfsense_system_group.live",
				ImportState:       true,
				ImportStateId:     testAccLiveGroupName,
				ImportStateVerify: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// pfsense_system_tunable
// ---------------------------------------------------------------------------

// `net.inet.ip.redirect` decides whether the kernel emits ICMP redirects for
// traffic it routes; it cannot affect a management session to the box itself, and
// nothing on the reference box routes through it. Interface, ARP and routing
// tunables are deliberately avoided. The second (and last) value the test applies
// is `1`, the box's running value, so the sysctl is where it started even though
// the tunable itself is deleted on destroy.
const (
	testAccLiveTunableName  = "net.inet.ip.redirect"
	testAccLiveTunableDescr = "tftest_live_tunable"
)

func testAccSystemTunableLiveConfig(value, descr string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "pfsense_system_tunable" "live" {
  tunable = %q
  value   = %q
  descr   = %q
}
`, testAccLiveTunableName, value, descr)
}

func TestAccSystemTunableResourceLive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckLiveObjectAbsent(t, "tunable", systemTunablePlural, "tunable", testAccLiveTunableName)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckLiveObjectAbsent("tunable", systemTunablePlural, "tunable", testAccLiveTunableName),
		Steps: []resource.TestStep{
			{
				Config: testAccSystemTunableLiveConfig("0", testAccLiveTunableDescr),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_system_tunable.live", "id", testAccLiveTunableName),
					resource.TestCheckResourceAttr("pfsense_system_tunable.live", "tunable", testAccLiveTunableName),
					resource.TestCheckResourceAttr("pfsense_system_tunable.live", "value", "0"),
					resource.TestCheckResourceAttr("pfsense_system_tunable.live", "descr", testAccLiveTunableDescr),
					testAccCheckLiveTunableValue(testAccLiveTunableName, "0"),
				),
			},
			{
				// `tunable` (the natural key) is unchanged; the value and the
				// description move. `1` is the box's own value for this sysctl, so
				// this step doubles as the restore.
				Config: testAccSystemTunableLiveConfig("1", testAccLiveTunableDescr+" (updated)"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_system_tunable.live", "id", testAccLiveTunableName),
					resource.TestCheckResourceAttr("pfsense_system_tunable.live", "value", "1"),
					resource.TestCheckResourceAttr("pfsense_system_tunable.live", "descr", testAccLiveTunableDescr+" (updated)"),
					testAccCheckLiveTunableValue(testAccLiveTunableName, "1"),
				),
			},
			{
				ResourceName:      "pfsense_system_tunable.live",
				ImportState:       true,
				ImportStateId:     testAccLiveTunableName,
				ImportStateVerify: true,
			},
		},
	})
}

// testAccCheckLiveTunableValue asserts the value the box stored for a tunable,
// read back over the API rather than out of Terraform state.
func testAccCheckLiveTunableValue(tunable, want string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		c, err := testAccClient()
		if err != nil {
			return fmt.Errorf("building verification client: %w", err)
		}
		_, obj, found, err := findByKey(context.Background(), c, systemTunablePlural, "tunable", tunable)
		if err != nil {
			return fmt.Errorf("reading tunable %q: %w", tunable, err)
		}
		if !found {
			return fmt.Errorf("tunable %q is not configured on the box", tunable)
		}
		if got := objectKey(obj, "value"); got != want {
			return fmt.Errorf("tunable %q: box reports value %q, want %q", tunable, got, want)
		}
		return nil
	}
}

// ---------------------------------------------------------------------------
// pfsense_system_ca
// ---------------------------------------------------------------------------

// CAs are keyed by the persistent `refid` the API assigns, so the `tftest_`
// marker lives in `descr` and the import ID is read out of state. The box's own
// webConfigurator certificate has no CA behind it and is never touched here.
const testAccLiveCADescr = "tftest_live_ca"

// testAccSystemCABlock renders one throwaway CA. Every field the API echoes back
// is pinned: they are all Optional (never Computed), so an omitted one would read
// back as the API default and leave a diff that never settles.
func testAccSystemCABlock(label, descr, crt, prv string, serial int) string {
	return fmt.Sprintf(`
resource "pfsense_system_ca" %q {
  descr        = %q
  trust        = false
  randomserial = false
  serial       = %d
  crt          = %q
  prv          = %q
}
`, label, descr, serial, crt, prv)
}

func testAccSystemCALiveConfig(t *testing.T, descr string, serial int) string {
	certs := testAccCertificates(t)
	return testAccProviderConfig() + testAccSystemCABlock("live", descr, certs.CACert, certs.CAKey, serial)
}

func TestAccSystemCAResourceLive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckLiveObjectAbsent(t, "certificate authority", systemCAPlural, "descr", testAccLiveCADescr)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckLiveObjectAbsent("certificate authority", systemCAPlural, "descr", testAccLiveCADescr),
		Steps: []resource.TestStep{
			{
				Config: testAccSystemCALiveConfig(t, testAccLiveCADescr, 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_system_ca.live", "descr", testAccLiveCADescr),
					resource.TestCheckResourceAttr("pfsense_system_ca.live", "trust", "false"),
					resource.TestCheckResourceAttr("pfsense_system_ca.live", "randomserial", "false"),
					resource.TestCheckResourceAttr("pfsense_system_ca.live", "serial", "1"),
					// The refid is assigned by the API and doubles as the ID.
					resource.TestCheckResourceAttrSet("pfsense_system_ca.live", "refid"),
					resource.TestCheckResourceAttrPair("pfsense_system_ca.live", "id", "pfsense_system_ca.live", "refid"),
				),
			},
			{
				// The refid (the identifier) is untouched, so this is an in-place
				// update; the description and the next serial move.
				Config: testAccSystemCALiveConfig(t, testAccLiveCADescr+" (updated)", 5),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_system_ca.live", "descr", testAccLiveCADescr+" (updated)"),
					resource.TestCheckResourceAttr("pfsense_system_ca.live", "serial", "5"),
					resource.TestCheckResourceAttrPair("pfsense_system_ca.live", "id", "pfsense_system_ca.live", "refid"),
				),
			},
			{
				// CAs import by refid, not by description.
				ResourceName:      "pfsense_system_ca.live",
				ImportState:       true,
				ImportStateIdFunc: testAccImportStateIDFromAttr("pfsense_system_ca.live", "refid"),
				ImportStateVerify: true,
				// The API stores the private key but never returns it.
				ImportStateVerifyIgnore: []string{"prv"},
			},
		},
	})
}

// ---------------------------------------------------------------------------
// pfsense_system_certificate
// ---------------------------------------------------------------------------

// The certificate is self-signed and imported on its own, so `caref` stays null:
// nothing here reads, replaces or deletes the box's "GUI default" certificate,
// which serves its webConfigurator TLS.
const testAccLiveCertificateDescr = "tftest_live_cert"

func testAccSystemCertificateLiveConfig(t *testing.T, descr string) string {
	certs := testAccCertificates(t)
	return testAccProviderConfig() + fmt.Sprintf(`
resource "pfsense_system_certificate" "live" {
  descr = %q
  type  = "server"
  crt   = %q
  prv   = %q
}
`, descr, certs.SelfSignedCert, certs.SelfSignedKey)
}

func TestAccSystemCertificateResourceLive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckLiveObjectAbsent(t, "certificate", systemCertificatePlural, "descr", testAccLiveCertificateDescr)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckLiveObjectAbsent("certificate", systemCertificatePlural, "descr", testAccLiveCertificateDescr),
		Steps: []resource.TestStep{
			{
				Config: testAccSystemCertificateLiveConfig(t, testAccLiveCertificateDescr),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_system_certificate.live", "descr", testAccLiveCertificateDescr),
					resource.TestCheckResourceAttr("pfsense_system_certificate.live", "type", "server"),
					resource.TestCheckResourceAttrSet("pfsense_system_certificate.live", "refid"),
					resource.TestCheckResourceAttrPair("pfsense_system_certificate.live", "id", "pfsense_system_certificate.live", "refid"),
				),
			},
			{
				// The refid is untouched, so this is an in-place update of the
				// description. By now the state has been refreshed once, so the
				// validity dates the API computes are populated too.
				Config: testAccSystemCertificateLiveConfig(t, testAccLiveCertificateDescr+" (updated)"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_system_certificate.live", "descr", testAccLiveCertificateDescr+" (updated)"),
					resource.TestCheckResourceAttrPair("pfsense_system_certificate.live", "id", "pfsense_system_certificate.live", "refid"),
					resource.TestCheckResourceAttrSet("pfsense_system_certificate.live", "valid_from"),
					resource.TestCheckResourceAttrSet("pfsense_system_certificate.live", "valid_until"),
				),
			},
			{
				ResourceName:      "pfsense_system_certificate.live",
				ImportState:       true,
				ImportStateIdFunc: testAccImportStateIDFromAttr("pfsense_system_certificate.live", "refid"),
				ImportStateVerify: true,
				// The API stores the private key but never returns it.
				ImportStateVerifyIgnore: []string{"prv"},
			},
		},
	})
}

// ---------------------------------------------------------------------------
// pfsense_system_crl
// ---------------------------------------------------------------------------

// An internal CRL has to be signed by a CA that holds a private key, so the
// config brings its own throwaway CA rather than pointing at anything already on
// the box. Terraform tears the CRL down before the CA it depends on.
//
// `serial` is deliberately left out of the configuration: the box regenerates an
// internal CRL on every write and increments the serial as it does, so a pinned
// value would leave a diff that never settles.
const (
	testAccLiveCRLDescr   = "tftest_live_crl"
	testAccLiveCRLCADescr = "tftest_live_crl_ca"
)

func testAccSystemCRLLiveConfig(t *testing.T, lifetime int) string {
	certs := testAccCertificates(t)
	return testAccProviderConfig() +
		testAccSystemCABlock("crl_ca", testAccLiveCRLCADescr, certs.CACert, certs.CAKey, 1) +
		fmt.Sprintf(`
resource "pfsense_system_crl" "live" {
  caref    = pfsense_system_ca.crl_ca.refid
  descr    = %q
  method   = "internal"
  lifetime = %d
}
`, testAccLiveCRLDescr, lifetime)
}

func TestAccSystemCRLResourceLive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckLiveObjectAbsent(t, "CRL", systemCRLPlural, "descr", testAccLiveCRLDescr)
			testAccPreCheckLiveObjectAbsent(t, "certificate authority", systemCAPlural, "descr", testAccLiveCRLCADescr)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: testAccCheckLiveObjectsAbsent(
			testAccCheckLiveObjectAbsent("CRL", systemCRLPlural, "descr", testAccLiveCRLDescr),
			testAccCheckLiveObjectAbsent("certificate authority", systemCAPlural, "descr", testAccLiveCRLCADescr),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccSystemCRLLiveConfig(t, 730),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_system_crl.live", "id", testAccLiveCRLDescr),
					resource.TestCheckResourceAttr("pfsense_system_crl.live", "descr", testAccLiveCRLDescr),
					resource.TestCheckResourceAttr("pfsense_system_crl.live", "method", "internal"),
					resource.TestCheckResourceAttr("pfsense_system_crl.live", "lifetime", "730"),
					// The serial is seeded and then maintained by the box.
					resource.TestCheckResourceAttrSet("pfsense_system_crl.live", "serial"),
					resource.TestCheckResourceAttrSet("pfsense_system_crl.live", "refid"),
					resource.TestCheckResourceAttrPair("pfsense_system_crl.live", "caref", "pfsense_system_ca.crl_ca", "refid"),
					// An internal CRL is generated and signed by the box, which
					// returns the x509 data it produced.
					resource.TestCheckResourceAttrSet("pfsense_system_crl.live", "text"),
				),
			},
			{
				// `descr` (the natural key) is unchanged, so this is an in-place
				// update; only the lifetime moves.
				Config: testAccSystemCRLLiveConfig(t, 365),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_system_crl.live", "id", testAccLiveCRLDescr),
					resource.TestCheckResourceAttr("pfsense_system_crl.live", "lifetime", "365"),
					resource.TestCheckResourceAttrSet("pfsense_system_crl.live", "text"),
				),
			},
			{
				ResourceName:      "pfsense_system_crl.live",
				ImportState:       true,
				ImportStateId:     testAccLiveCRLDescr,
				ImportStateVerify: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// pfsense_system_crl_revoked_certificate
// ---------------------------------------------------------------------------

// The child resource needs a full chain: a CA with a private key, a certificate
// that CA signed, and an internal CRL on that CA. All three are throwaway and
// created by the config, so nothing already on the box is revoked.
const (
	testAccLiveRevokedCRLDescr  = "tftest_live_revoke_crl"
	testAccLiveRevokedCADescr   = "tftest_live_revoke_ca"
	testAccLiveRevokedCertDescr = "tftest_live_revoke_cert"
)

func testAccSystemCRLRevokedCertificateLiveConfig(t *testing.T, reason int) string {
	certs := testAccCertificates(t)
	return testAccProviderConfig() +
		testAccSystemCABlock("revoke_ca", testAccLiveRevokedCADescr, certs.CACert, certs.CAKey, 1) +
		fmt.Sprintf(`
resource "pfsense_system_certificate" "revoke_cert" {
  descr = %q
  type  = "server"
  crt   = %q
  prv   = %q

  # The API resolves `+"`caref`"+` by matching the issuer against the CAs already on
  # the box, so the CA has to exist before the certificate is imported.
  depends_on = [pfsense_system_ca.revoke_ca]
}

resource "pfsense_system_crl" "revoke_crl" {
  caref    = pfsense_system_ca.revoke_ca.refid
  descr    = %q
  method   = "internal"
  lifetime = 730

  # The CRL and the certificate are applied in parallel by Terraform (neither
  # depends on the other), and two concurrent writes to the pfSense config can
  # race and drop one of them. Serialize the CRL after the certificate so the
  # CRL is durable before the child resource resolves its parent.
  depends_on = [pfsense_system_certificate.revoke_cert]
}

resource "pfsense_system_crl_revoked_certificate" "live" {
  parent_id = pfsense_system_crl.revoke_crl.descr
  certref   = pfsense_system_certificate.revoke_cert.refid
  reason    = %d
}
`, testAccLiveRevokedCertDescr, certs.LeafCert, certs.LeafKey, testAccLiveRevokedCRLDescr, reason)
}

func TestAccSystemCRLRevokedCertificateResourceLive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckLiveObjectAbsent(t, "CRL", systemCRLPlural, "descr", testAccLiveRevokedCRLDescr)
			testAccPreCheckLiveObjectAbsent(t, "certificate authority", systemCAPlural, "descr", testAccLiveRevokedCADescr)
			testAccPreCheckLiveObjectAbsent(t, "certificate", systemCertificatePlural, "descr", testAccLiveRevokedCertDescr)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: testAccCheckLiveObjectsAbsent(
			testAccCheckLiveCRLRevokedCertificateCountAbsent(testAccLiveRevokedCRLDescr),
			testAccCheckLiveObjectAbsent("CRL", systemCRLPlural, "descr", testAccLiveRevokedCRLDescr),
			testAccCheckLiveObjectAbsent("certificate", systemCertificatePlural, "descr", testAccLiveRevokedCertDescr),
			testAccCheckLiveObjectAbsent("certificate authority", systemCAPlural, "descr", testAccLiveRevokedCADescr),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccSystemCRLRevokedCertificateLiveConfig(t, 0),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_system_crl_revoked_certificate.live", "parent_id", testAccLiveRevokedCRLDescr),
					resource.TestCheckResourceAttr("pfsense_system_crl_revoked_certificate.live", "reason", "0"),
					resource.TestCheckResourceAttrPair(
						"pfsense_system_crl_revoked_certificate.live", "certref",
						"pfsense_system_certificate.revoke_cert", "refid",
					),
					// `revoke_time` is stamped by the box when the entry is created.
					resource.TestCheckResourceAttrSet("pfsense_system_crl_revoked_certificate.live", "revoke_time"),
					testAccCheckLiveCRLRevokedCertificateCount(testAccLiveRevokedCRLDescr, 1),
				),
			},
			{
				// Both key components (the parent CRL and the certref) are
				// unchanged, so this is an in-place update; only the reason moves.
				Config: testAccSystemCRLRevokedCertificateLiveConfig(t, 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_system_crl_revoked_certificate.live", "parent_id", testAccLiveRevokedCRLDescr),
					resource.TestCheckResourceAttr("pfsense_system_crl_revoked_certificate.live", "reason", "1"),
					resource.TestCheckResourceAttrSet("pfsense_system_crl_revoked_certificate.live", "revoke_time"),
					testAccCheckLiveCRLRevokedCertificateCount(testAccLiveRevokedCRLDescr, 1),
				),
			},
			{
				// The import ID is the parent CRL description and the certref the
				// API assigned to the certificate, joined by a pipe.
				ResourceName: "pfsense_system_crl_revoked_certificate.live",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					certref, err := testAccImportStateIDFromAttr("pfsense_system_crl_revoked_certificate.live", "certref")(s)
					if err != nil {
						return "", err
					}
					return testAccLiveRevokedCRLDescr + "|" + certref, nil
				},
				ImportStateVerify: true,
			},
		},
	})
}

// testAccLiveCRLRevokedCertificates returns the revoked-certificate entries a CRL
// currently holds. The model has no plural endpoint, so the parent's `cert`
// collection is read directly, the same way the resource locates its object.
func testAccLiveCRLRevokedCertificates(crlDescr string) ([]map[string]any, error) {
	c, err := testAccClient()
	if err != nil {
		return nil, fmt.Errorf("building verification client: %w", err)
	}
	_, crl, found, err := findByKey(context.Background(), c, systemCRLPlural, "descr", crlDescr)
	if err != nil {
		return nil, fmt.Errorf("looking up CRL %q: %w", crlDescr, err)
	}
	if !found {
		return nil, nil
	}
	return getSliceMap(crl, "cert"), nil
}

func testAccCheckLiveCRLRevokedCertificateCount(crlDescr string, want int) resource.TestCheckFunc {
	return func(*terraform.State) error {
		certs, err := testAccLiveCRLRevokedCertificates(crlDescr)
		if err != nil {
			return err
		}
		if len(certs) != want {
			return fmt.Errorf("CRL %q: box reports %d revoked certificates, want %d", crlDescr, len(certs), want)
		}
		return nil
	}
}

// testAccCheckLiveCRLRevokedCertificateCountAbsent asserts the child is gone. It
// runs before the parent's own absence check, so it tolerates the parent CRL
// already having been destroyed.
func testAccCheckLiveCRLRevokedCertificateCountAbsent(crlDescr string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		certs, err := testAccLiveCRLRevokedCertificates(crlDescr)
		if err != nil {
			return err
		}
		if len(certs) != 0 {
			return fmt.Errorf("CRL %q still holds %d revoked certificates after destroy", crlDescr, len(certs))
		}
		return nil
	}
}

// ---------------------------------------------------------------------------
// pfsense_user_auth_server
// ---------------------------------------------------------------------------

// A throwaway RADIUS server pointing at a documentation-range address with a
// dummy secret. Nothing selects it as an authentication backend, so local
// `admin` authentication — which these tests depend on — is untouched, and the
// address is unreachable so no credential ever leaves the box.
const (
	testAccLiveAuthServerName = "tftest_live_auth"
	testAccLiveAuthServerHost = "192.0.2.50"
)

// testAccUserAuthServerLiveConfig renders the auth server for one step.
// `radius_nasip_attribute` is required by the API for RADIUS servers.
func testAccUserAuthServerLiveConfig(host string, timeout int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "pfsense_user_auth_server" "live" {
  name                   = %q
  type                   = "radius"
  host                   = %q
  radius_secret          = "tftest-dummy-secret"
  radius_auth_port       = "1812"
  radius_acct_port       = "1813"
  radius_protocol        = "PAP"
  radius_timeout         = %d
  radius_nasip_attribute = "wan"
}
`, testAccLiveAuthServerName, host, timeout)
}

func TestAccUserAuthServerResourceLive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckLiveObjectAbsent(t, "authentication server", userAuthServerPlural, "name", testAccLiveAuthServerName)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckLiveObjectAbsent("authentication server", userAuthServerPlural, "name", testAccLiveAuthServerName),
		Steps: []resource.TestStep{
			{
				Config: testAccUserAuthServerLiveConfig(testAccLiveAuthServerHost, 5),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_user_auth_server.live", "id", testAccLiveAuthServerName),
					resource.TestCheckResourceAttr("pfsense_user_auth_server.live", "name", testAccLiveAuthServerName),
					resource.TestCheckResourceAttr("pfsense_user_auth_server.live", "type", "radius"),
					resource.TestCheckResourceAttr("pfsense_user_auth_server.live", "host", testAccLiveAuthServerHost),
					resource.TestCheckResourceAttr("pfsense_user_auth_server.live", "radius_protocol", "PAP"),
					resource.TestCheckResourceAttr("pfsense_user_auth_server.live", "radius_timeout", "5"),
					// The secret is write-only in the API; the resource must keep
					// the configured value rather than null it out on read.
					resource.TestCheckResourceAttr("pfsense_user_auth_server.live", "radius_secret", "tftest-dummy-secret"),
					resource.TestCheckResourceAttrSet("pfsense_user_auth_server.live", "refid"),
				),
			},
			{
				// `name` (the natural key) is unchanged; the host and the timeout
				// move.
				Config: testAccUserAuthServerLiveConfig("198.51.100.50", 10),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_user_auth_server.live", "id", testAccLiveAuthServerName),
					resource.TestCheckResourceAttr("pfsense_user_auth_server.live", "host", "198.51.100.50"),
					resource.TestCheckResourceAttr("pfsense_user_auth_server.live", "radius_timeout", "10"),
					resource.TestCheckResourceAttr("pfsense_user_auth_server.live", "radius_secret", "tftest-dummy-secret"),
				),
			},
			{
				ResourceName:      "pfsense_user_auth_server.live",
				ImportState:       true,
				ImportStateId:     testAccLiveAuthServerName,
				ImportStateVerify: true,
				// The API never echoes the shared secret back, so an imported
				// server has none to compare against.
				ImportStateVerifyIgnore: []string{"radius_secret"},
			},
		},
	})
}

// ---------------------------------------------------------------------------
// pfsense_system_restapi_access_list_entry
// ---------------------------------------------------------------------------

// This resource edits the access list guarding the very API the tests run over,
// so the shape of the change matters more than usual. The box ships two default
// entries — `allow 0.0.0.0/0` and `allow 0::/0`, both weight 10 — and the test
// neither touches nor reorders them; it adds one extra `allow` entry for a
// documentation-range network at a higher weight (evaluated after the defaults).
// An entry that does not match the management client cannot deny it, no `deny`
// entry is ever created, and every step asserts the default allow-all entries are
// still in place. If a step did lock the client out, the next API call would fail
// and the test would report it rather than leave the box unreachable.
const (
	testAccLiveACLEntryNetwork = "192.0.2.0/24"
	testAccLiveACLEntryDescr   = "tftest_live_acl_entry"
)

// testAccRESTAPIAccessListEntryLiveConfig renders the entry for one step. `users`
// and `sched` are pinned to the empty values the API reports for an entry that
// applies to everyone at all times, so the plan settles.
func testAccRESTAPIAccessListEntryLiveConfig(weight int, descr string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "pfsense_system_restapi_access_list_entry" "live" {
  type    = "allow"
  weight  = %d
  network = %q
  users   = []
  sched   = ""
  descr   = %q
}
`, weight, testAccLiveACLEntryNetwork, descr)
}

func TestAccSystemRESTAPIAccessListEntryResourceLive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckLiveACLEntryAbsent(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: testAccCheckLiveObjectsAbsent(
			testAccCheckLiveACLEntryAbsent(),
			testAccCheckLiveACLDefaultsIntact(),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccRESTAPIAccessListEntryLiveConfig(20, testAccLiveACLEntryDescr),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_system_restapi_access_list_entry.live", "id", testAccLiveACLEntryNetwork),
					resource.TestCheckResourceAttr("pfsense_system_restapi_access_list_entry.live", "network", testAccLiveACLEntryNetwork),
					resource.TestCheckResourceAttr("pfsense_system_restapi_access_list_entry.live", "type", "allow"),
					resource.TestCheckResourceAttr("pfsense_system_restapi_access_list_entry.live", "weight", "20"),
					resource.TestCheckResourceAttr("pfsense_system_restapi_access_list_entry.live", "descr", testAccLiveACLEntryDescr),
					testAccCheckLiveACLDefaultsIntact(),
				),
			},
			{
				// `network` (the natural key) is unchanged; the weight and the
				// description move.
				Config: testAccRESTAPIAccessListEntryLiveConfig(30, testAccLiveACLEntryDescr+" (updated)"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_system_restapi_access_list_entry.live", "id", testAccLiveACLEntryNetwork),
					resource.TestCheckResourceAttr("pfsense_system_restapi_access_list_entry.live", "weight", "30"),
					resource.TestCheckResourceAttr("pfsense_system_restapi_access_list_entry.live", "descr", testAccLiveACLEntryDescr+" (updated)"),
					testAccCheckLiveACLDefaultsIntact(),
				),
			},
			{
				ResourceName:      "pfsense_system_restapi_access_list_entry.live",
				ImportState:       true,
				ImportStateId:     testAccLiveACLEntryNetwork,
				ImportStateVerify: true,
			},
		},
	})
}

// testAccLiveACLEntries lists the REST API access list. The entry model has no
// plural endpoint of its own; the access list endpoint returns the whole array.
func testAccLiveACLEntries() ([]map[string]any, error) {
	c, err := testAccClient()
	if err != nil {
		return nil, fmt.Errorf("building verification client: %w", err)
	}
	items, err := c.List(context.Background(), restapiAccessListPlural, nil)
	if err != nil {
		return nil, fmt.Errorf("reading the REST API access list: %w", err)
	}
	entries := make([]map[string]any, 0, len(items))
	for _, item := range items {
		obj, err := decodeObject(item)
		if err != nil {
			return nil, fmt.Errorf("decoding a REST API access list entry: %w", err)
		}
		entries = append(entries, obj)
	}
	return entries, nil
}

func testAccLiveACLEntryExists() (bool, error) {
	entries, err := testAccLiveACLEntries()
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		if objectKey(entry, "network") == testAccLiveACLEntryNetwork {
			return true, nil
		}
	}
	return false, nil
}

func testAccPreCheckLiveACLEntryAbsent(t *testing.T) {
	t.Helper()

	found, err := testAccLiveACLEntryExists()
	if err != nil {
		t.Fatalf("checking REST API access list entry %q does not already exist: %v", testAccLiveACLEntryNetwork, err)
	}
	if found {
		t.Fatalf("REST API access list entry %q already exists on the box; remove it before running the live test", testAccLiveACLEntryNetwork)
	}
	if err := testAccLiveACLDefaultsIntact(); err != nil {
		t.Fatalf("refusing to touch the REST API access list: %v", err)
	}
}

func testAccCheckLiveACLEntryAbsent() resource.TestCheckFunc {
	return func(*terraform.State) error {
		found, err := testAccLiveACLEntryExists()
		if err != nil {
			return err
		}
		if found {
			return fmt.Errorf("REST API access list entry %q still exists after destroy", testAccLiveACLEntryNetwork)
		}
		return nil
	}
}

// testAccLiveACLDefaultsIntact asserts the box still allows the management client
// through: both default allow-all entries present, and no deny entry anywhere in
// the list.
func testAccLiveACLDefaultsIntact() error {
	entries, err := testAccLiveACLEntries()
	if err != nil {
		return err
	}
	defaults := map[string]bool{"0.0.0.0/0": false, "0::/0": false}
	for _, entry := range entries {
		network := objectKey(entry, "network")
		entryType := objectKey(entry, "type")
		if entryType == "deny" {
			return fmt.Errorf("REST API access list holds a deny entry for %q; the tests never create one", network)
		}
		if _, ok := defaults[network]; ok && entryType == "allow" {
			defaults[network] = true
		}
	}
	for network, present := range defaults {
		if !present {
			return fmt.Errorf("the default REST API allow entry for %q is missing", network)
		}
	}
	return nil
}

func testAccCheckLiveACLDefaultsIntact() resource.TestCheckFunc {
	return func(*terraform.State) error { return testAccLiveACLDefaultsIntact() }
}

// ---------------------------------------------------------------------------
// pfsense_system_package
// ---------------------------------------------------------------------------

// TestAccSystemPackageResourceLive documents why this resource has no live CRUD
// coverage instead of pretending it does.
//
// Creating the resource is a real `pkg install`: the reference box sits on an
// isolated bridge with no route to pkg.pfsense.org (a fetch from it times out),
// so an install would hang and then fail. The fallback — importing a package that
// is already installed — is not available either: the box's only package is
// pfSense-pkg-RESTAPI (the API package itself), and `/api/v2/system/packages`
// reports an empty inventory, so there is nothing for the resource to resolve by
// name. The check below reads that inventory rather than assuming it, so the skip
// reports what the box actually says.
func TestAccSystemPackageResourceLive(t *testing.T) {
	testAccPreCheck(t)

	c, err := testAccClient()
	if err != nil {
		t.Fatalf("building verification client: %v", err)
	}
	installed, err := c.List(context.Background(), systemPackagePlural, nil)
	if err != nil {
		t.Fatalf("reading the installed package inventory: %v", err)
	}

	t.Skipf("skipping live CRUD for pfsense_system_package: installing a package needs a route to "+
		"pkg.pfsense.org and the reference box is on an isolated bridge, and %s reports %d installed "+
		"packages, so there is nothing to import instead", systemPackagePlural, len(installed))
}

// ---------------------------------------------------------------------------
// pfsense_system_hostname (singleton)
// ---------------------------------------------------------------------------

// Singletons have no create/destroy semantics: Delete is a no-op and the ID is
// always `system`. Each test therefore captures what the box holds, applies a
// reversible change, puts the captured value back in a later step, and asserts
// over the API that it stuck.
//
// The hostname change appends a marker suffix to the existing hostname label.
// Management is by IP, so the box stays reachable throughout, and the suffix is
// what makes the guard idempotent: a hostname still carrying it is a leftover
// from a run that died mid-test and is rolled back before this one starts.
const testAccLiveHostnameSuffix = "-tftest"

var (
	testAccOrigHostname string
	testAccOrigDomain   string
)

func testAccPreCheckLiveHostname(t *testing.T) {
	t.Helper()

	obj, err := testAccLiveGetObject(systemHostnamePath)
	if err != nil {
		t.Fatalf("reading the current hostname: %v", err)
	}
	hostname, domain := objectKey(obj, "hostname"), objectKey(obj, "domain")
	if hostname == "" {
		t.Fatalf("the box reports an empty hostname; refusing to run the live test")
	}

	if strings.HasSuffix(hostname, testAccLiveHostnameSuffix) {
		hostname = strings.TrimSuffix(hostname, testAccLiveHostnameSuffix)
		t.Logf("restoring hostname %q left behind by an earlier run", hostname)
		if err := testAccLiveUpdateObject(systemHostnamePath, map[string]any{
			"hostname": hostname,
			"domain":   domain,
		}); err != nil {
			t.Fatalf("restoring the hostname left behind by an earlier run: %v", err)
		}
	}

	testAccOrigHostname, testAccOrigDomain = hostname, domain
}

func testAccSystemHostnameLiveConfig(hostname, domain string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "pfsense_system_hostname" "live" {
  hostname = %q
  domain   = %q
}
`, hostname, domain)
}

func TestAccSystemHostnameSingletonLive(t *testing.T) {
	// The captured hostname is interpolated into the step configurations, which
	// the test case renders as it is built — so the capture has to happen before
	// resource.Test, not in its PreCheck hook.
	testAccPreCheck(t)
	testAccPreCheckLiveHostname(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		// Delete is a no-op, so this asserts what really matters: the box is back
		// on the hostname it started with.
		CheckDestroy: testAccCheckLiveSingletonRestored(systemHostnamePath, func() map[string]string {
			return map[string]string{"hostname": testAccOrigHostname, "domain": testAccOrigDomain}
		}),
		Steps: []resource.TestStep{
			{
				Config: testAccSystemHostnameLiveConfig(testAccOrigHostname+testAccLiveHostnameSuffix, testAccOrigDomain),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_system_hostname.live", "id", "system"),
					resource.TestCheckResourceAttrWith("pfsense_system_hostname.live", "hostname", func(value string) error {
						if want := testAccOrigHostname + testAccLiveHostnameSuffix; value != want {
							return fmt.Errorf("hostname is %q, want %q", value, want)
						}
						return nil
					}),
					testAccCheckLiveSingletonValue(systemHostnamePath, "hostname", func() string {
						return testAccOrigHostname + testAccLiveHostnameSuffix
					}),
				),
			},
			{
				// Restore: the box goes back to the hostname the precheck captured.
				Config: testAccSystemHostnameLiveConfig(testAccOrigHostname, testAccOrigDomain),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_system_hostname.live", "id", "system"),
					testAccCheckLiveSingletonValue(systemHostnamePath, "hostname", func() string { return testAccOrigHostname }),
					testAccCheckLiveSingletonValue(systemHostnamePath, "domain", func() string { return testAccOrigDomain }),
				),
			},
			{
				ResourceName:      "pfsense_system_hostname.live",
				ImportState:       true,
				ImportStateId:     "system",
				ImportStateVerify: true,
			},
		},
	})
}

// testAccCheckLiveSingletonValue asserts a single field of a singleton endpoint,
// read back over the API rather than out of Terraform state. The wanted value is
// a closure because the precheck only captures it at run time.
func testAccCheckLiveSingletonValue(path, field string, want func() string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		obj, err := testAccLiveGetObject(path)
		if err != nil {
			return err
		}
		if got := objectKey(obj, field); got != want() {
			return fmt.Errorf("%s: box reports %s %q, want %q", path, field, got, want())
		}
		return nil
	}
}

// testAccCheckLiveSingletonRestored is the final gate on a singleton test: it
// fails loudly when the box was left holding anything other than the values the
// precheck captured.
func testAccCheckLiveSingletonRestored(path string, want func() map[string]string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		obj, err := testAccLiveGetObject(path)
		if err != nil {
			return err
		}
		for field, value := range want() {
			if got := objectKey(obj, field); got != value {
				return fmt.Errorf("%s was not restored: box reports %s %q, want %q", path, field, got, value)
			}
		}
		return nil
	}
}

// ---------------------------------------------------------------------------
// pfsense_system_dns (singleton)
// ---------------------------------------------------------------------------

// The DNS change appends a documentation-range nameserver to the list the box
// already has, rather than replacing it: the original servers keep working
// throughout, and the appended address is what makes the guard idempotent — a
// list still carrying it is a leftover that is stripped before this run starts.
// `dnsallowoverride` is pinned to the captured value in every step (asserted, but
// never changed) because a flipped boolean could not be told apart from the
// box's own setting on the next run.
const testAccLiveDNSServer = "192.0.2.53"

var (
	testAccOrigDNSServers       []string
	testAccOrigDNSAllowOverride bool
	testAccOrigDNSLocalhost     string
)

func testAccPreCheckLiveDNS(t *testing.T) {
	t.Helper()

	obj, err := testAccLiveGetObject(systemDNSPath)
	if err != nil {
		t.Fatalf("reading the current DNS settings: %v", err)
	}
	servers := getStringSlice(obj, "dnsserver")

	kept := make([]string, 0, len(servers))
	for _, server := range servers {
		if server != testAccLiveDNSServer {
			kept = append(kept, server)
		}
	}
	if len(kept) != len(servers) {
		t.Logf("removing DNS server %q left behind by an earlier run", testAccLiveDNSServer)
		if err := testAccLiveUpdateObject(systemDNSPath, map[string]any{"dnsserver": kept}); err != nil {
			t.Fatalf("removing the DNS server left behind by an earlier run: %v", err)
		}
	}

	testAccOrigDNSServers = kept
	testAccOrigDNSLocalhost = objectKey(obj, "dnslocalhost")
	if allow := getBool(obj, "dnsallowoverride"); allow != nil {
		testAccOrigDNSAllowOverride = *allow
	}
}

// testAccSystemDNSLiveConfig renders the DNS settings for one step.
// `dnslocalhost` is only managed when the box has a value for it: the attribute
// is nullable, and writing a value the box does not have would be a change the
// test cannot reverse from its own captured state.
func testAccSystemDNSLiveConfig(servers []string) string {
	localhost := ""
	if testAccOrigDNSLocalhost != "" {
		localhost = fmt.Sprintf("  dnslocalhost     = %q\n", testAccOrigDNSLocalhost)
	}
	return testAccProviderConfig() + fmt.Sprintf(`
resource "pfsense_system_dns" "live" {
  dnsallowoverride = %t
  dnsserver        = %s
%s}
`, testAccOrigDNSAllowOverride, testAccHCLStringList(servers), localhost)
}

func TestAccSystemDNSSingletonLive(t *testing.T) {
	// The captured nameserver list is interpolated into the step configurations,
	// so it has to be read before the test case is built.
	testAccPreCheck(t)
	testAccPreCheckLiveDNS(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckLiveDNSServersRestored(),
		Steps: []resource.TestStep{
			{
				Config: testAccSystemDNSLiveConfig(append(append([]string{}, testAccOrigDNSServers...), testAccLiveDNSServer)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_system_dns.live", "id", "system"),
					resource.TestCheckResourceAttrWith("pfsense_system_dns.live", "dnsserver.#", func(value string) error {
						if want := fmt.Sprintf("%d", len(testAccOrigDNSServers)+1); value != want {
							return fmt.Errorf("dnsserver count is %s, want %s", value, want)
						}
						return nil
					}),
					testAccCheckLiveDNSServerPresent(testAccLiveDNSServer, true),
				),
			},
			{
				// Restore: the appended nameserver goes away and the box is back on
				// the list the precheck captured.
				Config: testAccSystemDNSLiveConfig(testAccOrigDNSServers),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_system_dns.live", "id", "system"),
					testAccCheckLiveDNSServerPresent(testAccLiveDNSServer, false),
					testAccCheckLiveDNSServersRestored(),
				),
			},
			{
				ResourceName:      "pfsense_system_dns.live",
				ImportState:       true,
				ImportStateId:     "system",
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckLiveDNSServerPresent(server string, want bool) resource.TestCheckFunc {
	return func(*terraform.State) error {
		obj, err := testAccLiveGetObject(systemDNSPath)
		if err != nil {
			return err
		}
		found := false
		for _, configured := range getStringSlice(obj, "dnsserver") {
			if configured == server {
				found = true
				break
			}
		}
		if found != want {
			return fmt.Errorf("DNS server %q present on the box: %t, want %t", server, found, want)
		}
		return nil
	}
}

// testAccCheckLiveDNSServersRestored is the final gate: the box's nameserver list
// must match the one the precheck captured, entry for entry.
func testAccCheckLiveDNSServersRestored() resource.TestCheckFunc {
	return func(*terraform.State) error {
		obj, err := testAccLiveGetObject(systemDNSPath)
		if err != nil {
			return err
		}
		got := getStringSlice(obj, "dnsserver")
		if len(got) != len(testAccOrigDNSServers) {
			return fmt.Errorf("DNS servers were not restored: box reports %v, want %v", got, testAccOrigDNSServers)
		}
		for i, server := range testAccOrigDNSServers {
			if got[i] != server {
				return fmt.Errorf("DNS servers were not restored: box reports %v, want %v", got, testAccOrigDNSServers)
			}
		}
		return nil
	}
}

// ---------------------------------------------------------------------------
// pfsense_system_timezone (singleton)
// ---------------------------------------------------------------------------

// `Etc/UTC` is deliberately chosen as the changed value: it is a different zone
// name (so the resource has real work to do) but the same actual zone as the
// box's `UTC` default, which keeps the guard idempotent and harmless — a box
// found still on `Etc/UTC` is a leftover from a run that died mid-test, and
// rolling it back to `UTC` changes the name without moving the clock.
const (
	testAccLiveTimezone         = "Etc/UTC"
	testAccLiveTimezoneFallback = "UTC"
)

var testAccOrigTimezone string

func testAccPreCheckLiveTimezone(t *testing.T) {
	t.Helper()

	obj, err := testAccLiveGetObject(systemTimezonePath)
	if err != nil {
		t.Fatalf("reading the current timezone: %v", err)
	}
	timezone := objectKey(obj, "timezone")
	if timezone == "" {
		t.Fatalf("the box reports an empty timezone; refusing to run the live test")
	}

	if timezone == testAccLiveTimezone {
		timezone = testAccLiveTimezoneFallback
		t.Logf("restoring timezone %q left behind by an earlier run", timezone)
		if err := testAccLiveUpdateObject(systemTimezonePath, map[string]any{"timezone": timezone}); err != nil {
			t.Fatalf("restoring the timezone left behind by an earlier run: %v", err)
		}
	}

	testAccOrigTimezone = timezone
}

func testAccSystemTimezoneLiveConfig(timezone string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "pfsense_system_timezone" "live" {
  timezone = %q
}
`, timezone)
}

func TestAccSystemTimezoneSingletonLive(t *testing.T) {
	// The captured timezone is interpolated into the restore step, so it has to
	// be read before the test case is built.
	testAccPreCheck(t)
	testAccPreCheckLiveTimezone(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: testAccCheckLiveSingletonRestored(systemTimezonePath, func() map[string]string {
			return map[string]string{"timezone": testAccOrigTimezone}
		}),
		Steps: []resource.TestStep{
			{
				Config: testAccSystemTimezoneLiveConfig(testAccLiveTimezone),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_system_timezone.live", "id", "system"),
					resource.TestCheckResourceAttr("pfsense_system_timezone.live", "timezone", testAccLiveTimezone),
					testAccCheckLiveSingletonValue(systemTimezonePath, "timezone", func() string { return testAccLiveTimezone }),
				),
			},
			{
				// Restore: the box goes back to the timezone the precheck captured.
				Config: testAccSystemTimezoneLiveConfig(testAccOrigTimezone),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_system_timezone.live", "id", "system"),
					testAccCheckLiveSingletonValue(systemTimezonePath, "timezone", func() string { return testAccOrigTimezone }),
				),
			},
			{
				ResourceName:      "pfsense_system_timezone.live",
				ImportState:       true,
				ImportStateId:     "system",
				ImportStateVerify: true,
			},
		},
	})
}
