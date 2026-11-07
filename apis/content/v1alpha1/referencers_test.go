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

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
)

// named sets the metadata of obj and its external name, and returns it.
func named[T client.Object](obj T, namespace, name, externalName string) T {
	obj.SetNamespace(namespace)
	obj.SetName(name)
	meta.SetExternalName(obj, externalName)

	return obj
}

// TestRepositoryResolveReferences checks that the blob store, cleanup
// policies, group members and routing rule resolve to the external names of
// the managed resources they reference, in the Repository's namespace only.
func TestRepositoryResolveReferences(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()

	for _, add := range []func(*runtime.Scheme) error{SchemeBuilder.AddToScheme, instancev1alpha1.SchemeBuilder.AddToScheme} {
		err := add(scheme)
		if err != nil {
			t.Fatalf("AddToScheme() error = %v", err)
		}
	}

	reader := clientfake.NewClientBuilder().WithScheme(scheme).WithObjects(
		named(&instancev1alpha1.BlobStore{}, "team", "store", "team-store"),
		named(&CleanupPolicy{}, "team", "weekly", "team-weekly"),
		named(&Repository{}, "team", "releases", "maven-releases"),
		named(&Repository{}, "team", "central", "maven-central"),
		named(&RoutingRule{}, "team", "allow-org", "team-allow-org"),
		named(&instancev1alpha1.BlobStore{}, "other", "private", "other-private"),
	).Build()

	repository := &Repository{ObjectMeta: metav1.ObjectMeta{Name: "group", Namespace: "team"}}
	repository.Spec.ForProvider = RepositoryParameters{
		Storage: &RepositoryStorage{BlobStoreNameRef: &xpv2.Reference{Name: "store"}},
		Cleanup: &RepositoryCleanup{PolicyNameRefs: []xpv2.Reference{{Name: "weekly"}}},
		Group: &GroupConfig{MemberNameRefs: []xpv2.Reference{
			{Name: "releases"},
			{Name: "central"},
		}},
		RoutingRuleRef: &xpv2.Reference{Name: "allow-org"},
	}

	err := repository.ResolveReferences(context.Background(), reader)
	if err != nil {
		t.Fatalf("ResolveReferences() error = %v", err)
	}

	params := repository.Spec.ForProvider

	if params.Storage.BlobStoreName != "team-store" {
		t.Errorf("storage.blobStoreName = %q, want %q", params.Storage.BlobStoreName, "team-store")
	}

	if !slices.Equal(params.Cleanup.PolicyNames, []string{"team-weekly"}) {
		t.Errorf("cleanup.policyNames = %v, want [team-weekly]", params.Cleanup.PolicyNames)
	}

	// Group members keep the order of their references, which sets the
	// order in which Nexus searches them.
	if !slices.Equal(params.Group.MemberNames, []string{"maven-releases", "maven-central"}) {
		t.Errorf("group.memberNames = %v, want [maven-releases maven-central]", params.Group.MemberNames)
	}

	if ptr.Deref(params.RoutingRule, "") != "team-allow-org" {
		t.Errorf("routingRule = %v, want %q", ptr.Deref(params.RoutingRule, "<nil>"), "team-allow-org")
	}

	other := &Repository{ObjectMeta: metav1.ObjectMeta{Name: "hosted", Namespace: "team"}}
	other.Spec.ForProvider.Storage = &RepositoryStorage{BlobStoreNameRef: &xpv2.Reference{Name: "private"}}

	err = other.ResolveReferences(context.Background(), reader)
	if err == nil {
		t.Errorf("ResolveReferences() of a BlobStore in another namespace error = nil, want an error")
	}
}

// TestRepositoryResolveReferencesWithoutReferences checks that a Repository
// that references nothing keeps its values.
func TestRepositoryResolveReferencesWithoutReferences(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()

	err := SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("AddToScheme() error = %v", err)
	}

	repository := &Repository{ObjectMeta: metav1.ObjectMeta{Name: "hosted", Namespace: "team"}}
	repository.Spec.ForProvider = RepositoryParameters{
		Storage: &RepositoryStorage{BlobStoreName: "default"},
		Cleanup: &RepositoryCleanup{PolicyNames: []string{"weekly"}},
	}
	want := repository.Spec.ForProvider.DeepCopy()

	err = repository.ResolveReferences(context.Background(), clientfake.NewClientBuilder().WithScheme(scheme).Build())
	if err != nil {
		t.Fatalf("ResolveReferences() error = %v", err)
	}

	if repository.Spec.ForProvider.Storage.BlobStoreName != want.Storage.BlobStoreName ||
		!slices.Equal(repository.Spec.ForProvider.Cleanup.PolicyNames, want.Cleanup.PolicyNames) ||
		repository.Spec.ForProvider.RoutingRule != nil {
		t.Errorf("ResolveReferences() changed the parameters to %+v, want %+v", repository.Spec.ForProvider, *want)
	}
}
