package instance

import (
	"errors"
	"strings"
	"testing"
)

// sameCertPem is a self-signed test certificate.
const sameCertPem = `-----BEGIN CERTIFICATE-----
MIIDETCCAfmgAwIBAgIULjaoxQ6pRnkQYrHMqHCOoa4I6wYwDQYJKoZIhvcNAQEL
BQAwGDEWMBQGA1UEAwwNY2xhdWRlLXRlc3QtYTAeFw0yNjExMDgwMzAyMjFaFw0y
NjEyMDgwMzAyMjFaMBgxFjAUBgNVBAMMDWNsYXVkZS10ZXN0LWEwggEiMA0GCSqG
SIb3DQEBAQUAA4IBDwAwggEKAoIBAQDSGh1igB4rw6maggGQQdXq4f8lqeD7RjoZ
QKqsa+JCZ/RIc4+atN3g1+JHA6QQXYUGr+d0DUVWrmeQFj7sSesqAC/9qwezbR2e
MmzoKqV07c7gw3Ac+EnV0QhDjp6Mvvfjykrg557/XaPKUIbaX9m/DbaOMg2vhP4r
DBmwWOiZm7Wp8Nf3lpZz9jbTbHEmGpA5SNY/Gr1TCMbeGEs8SmU7gBAer7mE/V6e
NK0SqWiPXatGWtNUfIWpfbvCgfPIsZwjEHCmV+iQx80jVZMII3NBB/L4NTlMU3AU
JxKslM/rL1cNmdB/F/sv8tNBErqkyhSs0ediGYQuWSJNHhktWuUhAgMBAAGjUzBR
MB0GA1UdDgQWBBTUCQ4lU4cnI6VYGMEfofgiDiXatDAfBgNVHSMEGDAWgBTUCQ4l
U4cnI6VYGMEfofgiDiXatDAPBgNVHRMBAf8EBTADAQH/MA0GCSqGSIb3DQEBCwUA
A4IBAQBIGoUHTikABdyG1tBhzq/8r2pw31avfBynq2ry+k4jQyojXGC3lhNufew8
9/+nF3yJeAHk6vGheotEMed7w7fuozqnQu5qOspKeXpD2m7FAHd7MFuc8XgYVUdK
daF9FDTxB1NQRfVNsfkpkWfsJlD5NcKYjKz9xyyL81KxY8g30KMNbfzpLx3Gor5G
vbAQC9aTRnkdDVWTBDDUHtyCmpA94hXPIihWi/P4cypNsgFiDUK8YPemmie0UVK2
VuZXvnBuK6wxqcy1Y1MK3N6OneCXqKlOZn2RWR4PllApmqbWGSxZmzRb4D6TR43j
qknoPzJD0I9euF9C+l9ggWuRMDF0
-----END CERTIFICATE-----`

// TestSameCertificate checks that certificates compare by content.
func TestSameCertificate(t *testing.T) {
	t.Parallel()

	lines := strings.Split(sameCertPem, "\n")
	body := strings.Join(lines[1:len(lines)-1], "")
	oneLine := lines[0] + "\n" + body + "\n" + lines[len(lines)-1]
	differentBody := strings.Replace(sameCertPem, "MIIDETCC", "MIIDETCD", 1)

	tests := []struct {
		name string
		a, b string
		want bool
	}{
		{name: "Identical", a: sameCertPem, b: sameCertPem, want: true},
		{name: "CRLF", a: sameCertPem, b: strings.ReplaceAll(sameCertPem, "\n", "\r\n"), want: true},
		{name: "Rewrapped", a: sameCertPem, b: oneLine, want: true},
		{name: "SurroundingWhitespace", a: sameCertPem, b: "\n  " + sameCertPem + "\n\n", want: true},
		{name: "DifferentCertificate", a: sameCertPem, b: differentBody, want: false},
		{name: "NotPEMSameText", a: " not a pem ", b: "not a pem", want: true},
		{name: "NotPEMVersusPEM", a: "not a pem", b: sameCertPem, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := SameCertificate(tt.a, tt.b); got != tt.want {
				t.Errorf("SameCertificate() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParseCertificate checks that the first certificate, and nothing else, is
// returned.
func TestParseCertificate(t *testing.T) {
	t.Parallel()

	const key = "-----BEGIN PRIVATE KEY-----\nc2VjcmV0\n-----END PRIVATE KEY-----\n"

	const otherCert = "-----BEGIN CERTIFICATE-----\nb3RoZXI=\n-----END CERTIFICATE-----\n"

	tests := map[string]struct {
		data                   string
		wantErr                bool
		wantInvalidCertificate bool
	}{
		"Certificate":          {data: sameCertPem},
		"Bundle":               {data: sameCertPem + "\n" + otherCert},
		"CRLF":                 {data: strings.ReplaceAll(sameCertPem, "\n", "\r\n")},
		"Preamble":             {data: "# Some CA\nsubject=CN = test\n" + sameCertPem},
		"TrustedCertificate":   {data: strings.ReplaceAll(sameCertPem, "CERTIFICATE", "TRUSTED CERTIFICATE")},
		"PrivateKeyFirst":      {data: key + sameCertPem},
		"PrivateKeyLast":       {data: sameCertPem + "\n" + key},
		"Empty":                {wantErr: true},
		"PrivateKeyOnly":       {data: key, wantErr: true},
		"LoginPage":            {data: "<html>log in</html>", wantErr: true},
		"TruncatedOnTheWire":   {data: sameCertPem[:len(sameCertPem)/2], wantErr: true},
		"MalformedCertificate": {data: otherCert, wantInvalidCertificate: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseCertificate([]byte(tc.data))
			if tc.wantInvalidCertificate {
				if err == nil || errors.Is(err, ErrNoCertificate) {
					t.Fatalf("ParseCertificate() error = %v, want certificate parsing error", err)
				}

				return
			}

			if tc.wantErr {
				if !errors.Is(err, ErrNoCertificate) {
					t.Fatalf("ParseCertificate() error = %v, want %v", err, ErrNoCertificate)
				}

				return
			}

			if err != nil {
				t.Fatalf("ParseCertificate() error = %v", err)
			}

			if want := sameCertPem + "\n"; string(got) != want {
				t.Errorf("ParseCertificate() = %q, want %q", got, want)
			}
		})
	}
}
