package iam

import (
	"errors"
	"strings"
	"testing"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
)

// testMetadata is minimal SAML identity provider metadata.
const testMetadata = `<?xml version="1.0" encoding="UTF-8"?>
<!-- an identity provider -->
<md:EntityDescriptor xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata" entityID="https://idp.example.com"/>`

// TestParseIdPMetadata checks that only SAML metadata is accepted, and that it
// is normalized.
func TestParseIdPMetadata(t *testing.T) {
	t.Parallel()

	const defaultNamespace = `<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata"></EntityDescriptor>`

	const latin1 = `<?xml version="1.0" encoding="ISO-8859-1"?><EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata"/>`

	tests := map[string]struct {
		data    string
		want    string
		wantErr bool
	}{
		"Metadata":            {data: testMetadata, want: testMetadata},
		"DefaultNamespace":    {data: defaultNamespace, want: defaultNamespace},
		"DeclaredLatin1":      {data: latin1, want: latin1},
		"ByteOrderMark":       {data: "\xef\xbb\xbf" + testMetadata, want: testMetadata},
		"SurroundingSpace":    {data: "\n  " + testMetadata + "\r\n", want: testMetadata},
		"Empty":               {data: "", wantErr: true},
		"NotXML":              {data: "-----BEGIN PRIVATE KEY-----", wantErr: true},
		"LoginPage":           {data: "<!DOCTYPE html><html><body>Log in</body></html>", wantErr: true},
		"OtherXML":            {data: `<Response xmlns="urn:oasis:names:tc:SAML:2.0:protocol"/>`, wantErr: true},
		"UTF16":               {data: "<\x00E\x00\xff\xfe", wantErr: true},
		"UndeclaredNamespace": {data: "<EntityDescriptor/>", wantErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseIdPMetadata([]byte(tc.data))
			if tc.wantErr {
				if !errors.Is(err, ErrNotIdPMetadata) {
					t.Fatalf("ParseIdPMetadata() error = %v, want %v", err, ErrNotIdPMetadata)
				}

				return
			}

			if err != nil {
				t.Fatalf("ParseIdPMetadata() error = %v", err)
			}

			if string(got) != tc.want {
				t.Errorf("ParseIdPMetadata() = %q, want %q", got, tc.want)
			}

			// The cached document is the parsed one, and is parsed again.
			again, err := ParseIdPMetadata(got)
			if err != nil || string(again) != tc.want {
				t.Errorf("ParseIdPMetadata() of its output = %q, %v, want it unchanged", again, err)
			}
		})
	}
}

// TestSAMLObservation_MetadataDigest checks that the observation records the
// digest of the metadata, not the metadata, which may be read from a Secret,
// and that a change of metadata is still drift.
func TestSAMLObservation_MetadataDigest(t *testing.T) {
	t.Parallel()

	cr := &iamv1alpha1.SAML{}
	cr.Spec.ForProvider = iamv1alpha1.SAMLParameters{EntityId: "nexus", UsernameAttribute: "username"}
	cr.Status.AtProvider = GenerateSAMLObservation(&v395.SamlConfigurationXO{
		IdpMetadata:       testMetadata,
		EntityId:          new("nexus"),
		UsernameAttribute: "username",
	})

	if got := cr.Status.AtProvider.IdpMetadataSHA256; got != IdPMetadataSHA256(testMetadata) || strings.Contains(got, "EntityDescriptor") {
		t.Errorf("IdpMetadataSHA256 = %q, want the digest of the metadata", got)
	}

	if !IsSAMLUpToDate(cr, testMetadata) {
		t.Error("IsSAMLUpToDate() with the observed metadata = false, want true")
	}

	if IsSAMLUpToDate(cr, testMetadata+" ") {
		t.Error("IsSAMLUpToDate() with other metadata = true, want false")
	}

	cr.Spec.ForProvider.UsernameAttribute = "uid"
	if IsSAMLUpToDate(cr, testMetadata) {
		t.Error("IsSAMLUpToDate() with another username attribute = true, want false")
	}
}
