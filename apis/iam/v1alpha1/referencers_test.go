package v1alpha1

import (
	"context"
	"slices"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	contentv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
)

// named sets the metadata of obj, its external name and its labels, and
// returns it.
func named[T client.Object](obj T, namespace, name, externalName string, labels map[string]string) T {
	obj.SetNamespace(namespace)
	obj.SetName(name)
	obj.SetLabels(labels)
	meta.SetExternalName(obj, externalName)

	return obj
}

// newReader returns a reader holding objects.
func newReader(t *testing.T, objects ...client.Object) client.Reader {
	t.Helper()

	scheme := runtime.NewScheme()

	for _, add := range []func(*runtime.Scheme) error{SchemeBuilder.AddToScheme, contentv1alpha1.SchemeBuilder.AddToScheme} {
		err := add(scheme)
		if err != nil {
			t.Fatalf("AddToScheme() error = %v", err)
		}
	}

	return clientfake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
}

// teamLabels select the managed resources of the team.
var teamLabels = map[string]string{"team": "platform"}

// TestRoleResolveReferences checks that privileges and roles resolve to the
// external names of the managed resources they reference, in the Role's
// namespace only.
func TestRoleResolveReferences(t *testing.T) {
	t.Parallel()

	reader := newReader(t,
		named(&Privilege{}, "team", "read", "nx-team-read", teamLabels),
		named(&Privilege{}, "team", "write", "nx-team-write", teamLabels),
		named(&Privilege{}, "other", "admin", "nx-other-admin", teamLabels),
		named(&Role{}, "team", "base", "team-base", nil),
		named(&Role{}, "other", "base", "other-base", nil),
	)

	tests := map[string]struct {
		params         RoleParameters
		wantPrivileges []string
		wantRoles      []string
		wantErr        bool
	}{
		"References": {
			params: RoleParameters{
				PrivilegeRefs: []xpv2.Reference{{Name: "write"}, {Name: "read"}},
				RoleRefs:      []xpv2.Reference{{Name: "base"}},
			},
			wantPrivileges: []string{"nx-team-write", "nx-team-read"},
			wantRoles:      []string{"team-base"},
		},
		"SelectorStaysInTheNamespace": {
			params: RoleParameters{
				PrivilegeSelector: &xpv2.Selector{MatchLabels: teamLabels},
			},
			wantPrivileges: []string{"nx-team-read", "nx-team-write"},
		},
		"ValuesWithoutReferencesAreKept": {
			params: RoleParameters{
				Privileges: []string{"nx-repository-view-*-*-browse"},
				Roles:      []string{"nx-anonymous"},
			},
			wantPrivileges: []string{"nx-repository-view-*-*-browse"},
			wantRoles:      []string{"nx-anonymous"},
		},
		"ReferenceToAnotherNamespaceFails": {
			params: RoleParameters{
				PrivilegeRefs: []xpv2.Reference{{Name: "admin"}},
			},
			wantErr: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			role := &Role{ObjectMeta: metav1.ObjectMeta{Name: "role", Namespace: "team"}}
			role.Spec.ForProvider = test.params

			err := role.ResolveReferences(context.Background(), reader)
			if (err != nil) != test.wantErr {
				t.Fatalf("ResolveReferences() error = %v, want error %t", err, test.wantErr)
			}

			if test.wantErr {
				return
			}

			if !slices.Equal(role.Spec.ForProvider.Privileges, test.wantPrivileges) {
				t.Errorf("privileges = %v, want %v", role.Spec.ForProvider.Privileges, test.wantPrivileges)
			}

			if !slices.Equal(role.Spec.ForProvider.Roles, test.wantRoles) {
				t.Errorf("roles = %v, want %v", role.Spec.ForProvider.Roles, test.wantRoles)
			}
		})
	}
}

// TestUserResolveReferences checks that roles resolve to the external names
// of the Roles they reference.
func TestUserResolveReferences(t *testing.T) {
	t.Parallel()

	reader := newReader(t, named(&Role{}, "team", "developers", "team-developers", nil))

	user := &User{ObjectMeta: metav1.ObjectMeta{Name: "user", Namespace: "team"}}
	user.Spec.ForProvider.RoleRefs = []xpv2.Reference{{Name: "developers"}}

	err := user.ResolveReferences(context.Background(), reader)
	if err != nil {
		t.Fatalf("ResolveReferences() error = %v", err)
	}

	if !slices.Equal(user.Spec.ForProvider.Roles, []string{"team-developers"}) {
		t.Errorf("roles = %v, want [team-developers]", user.Spec.ForProvider.Roles)
	}
}

// TestPrivilegeResolveReferences checks that the repository and content
// selector resolve to the external names of the managed resources they
// reference, in the Privilege's namespace only.
func TestPrivilegeResolveReferences(t *testing.T) {
	t.Parallel()

	reader := newReader(t,
		named(&contentv1alpha1.Repository{}, "team", "releases", "maven-releases", teamLabels),
		named(&contentv1alpha1.ContentSelector{}, "team", "artifacts", "team-artifacts", nil),
		named(&contentv1alpha1.Repository{}, "other", "private", "other-private", nil),
	)

	tests := map[string]struct {
		params              PrivilegeParameters
		wantRepository      *string
		wantContentSelector *string
		wantErr             bool
	}{
		"References": {
			params: PrivilegeParameters{
				RepositoryRef:      &xpv2.Reference{Name: "releases"},
				ContentSelectorRef: &xpv2.Reference{Name: "artifacts"},
			},
			wantRepository:      new("maven-releases"),
			wantContentSelector: new("team-artifacts"),
		},
		"Selector": {
			params: PrivilegeParameters{
				RepositorySelector: &xpv2.Selector{MatchLabels: teamLabels},
			},
			wantRepository: new("maven-releases"),
		},
		"ValueWithoutReferenceIsKept": {
			params:         PrivilegeParameters{Repository: new("*")},
			wantRepository: new("*"),
		},
		"ReferenceToAnotherNamespaceFails": {
			params:  PrivilegeParameters{RepositoryRef: &xpv2.Reference{Name: "private"}},
			wantErr: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			privilege := &Privilege{ObjectMeta: metav1.ObjectMeta{Name: "privilege", Namespace: "team"}}
			privilege.Spec.ForProvider = test.params

			err := privilege.ResolveReferences(context.Background(), reader)
			if (err != nil) != test.wantErr {
				t.Fatalf("ResolveReferences() error = %v, want error %t", err, test.wantErr)
			}

			if test.wantErr {
				return
			}

			if ptr.Deref(privilege.Spec.ForProvider.Repository, "") != ptr.Deref(test.wantRepository, "") {
				t.Errorf("repository = %v, want %v", ptr.Deref(privilege.Spec.ForProvider.Repository, "<nil>"), ptr.Deref(test.wantRepository, "<nil>"))
			}

			if ptr.Deref(privilege.Spec.ForProvider.ContentSelector, "") != ptr.Deref(test.wantContentSelector, "") {
				t.Errorf("contentSelector = %v, want %v", ptr.Deref(privilege.Spec.ForProvider.ContentSelector, "<nil>"), ptr.Deref(test.wantContentSelector, "<nil>"))
			}
		})
	}
}
