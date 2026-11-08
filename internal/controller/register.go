package controller

import (
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"

	contentv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/controller/config"
	contentcleanuppolicy "github.com/genesary/provider-sonatype-nexus/internal/controller/content/cleanuppolicy"
	contentcontentselector "github.com/genesary/provider-sonatype-nexus/internal/controller/content/contentselector"
	contentrepository "github.com/genesary/provider-sonatype-nexus/internal/controller/content/repository"
	contentroutingrule "github.com/genesary/provider-sonatype-nexus/internal/controller/content/routingrule"
	"github.com/genesary/provider-sonatype-nexus/internal/controller/crdgate"
	iamanonymousaccess "github.com/genesary/provider-sonatype-nexus/internal/controller/iam/anonymousaccess"
	iamldap "github.com/genesary/provider-sonatype-nexus/internal/controller/iam/ldap"
	iamprivilege "github.com/genesary/provider-sonatype-nexus/internal/controller/iam/privilege"
	iamrole "github.com/genesary/provider-sonatype-nexus/internal/controller/iam/role"
	iamsaml "github.com/genesary/provider-sonatype-nexus/internal/controller/iam/saml"
	iamuser "github.com/genesary/provider-sonatype-nexus/internal/controller/iam/user"
	iamutc "github.com/genesary/provider-sonatype-nexus/internal/controller/iam/usertokenconfiguration"
	instanceblobstore "github.com/genesary/provider-sonatype-nexus/internal/controller/instance/blobstore"
	instancecapability "github.com/genesary/provider-sonatype-nexus/internal/controller/instance/capability"
	instanceiqserver "github.com/genesary/provider-sonatype-nexus/internal/controller/instance/iqserverconfiguration"
	instancelicense "github.com/genesary/provider-sonatype-nexus/internal/controller/instance/license"
	instancesecurityrealm "github.com/genesary/provider-sonatype-nexus/internal/controller/instance/securityrealm"
	instancessltruststore "github.com/genesary/provider-sonatype-nexus/internal/controller/instance/securityssltruststore"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
)

// managedController is a managed resource controller, with the kind it
// reconciles.
type managedController struct {
	setup helpers.SetupFn
	gvk   schema.GroupVersionKind
}

// managedControllers lists the managed resource controllers.
var managedControllers = []managedController{
	{contentcleanuppolicy.Setup, contentv1alpha1.CleanupPolicyGroupVersionKind},
	{contentcontentselector.Setup, contentv1alpha1.ContentSelectorGroupVersionKind},
	{contentrepository.Setup, contentv1alpha1.RepositoryGroupVersionKind},
	{contentroutingrule.Setup, contentv1alpha1.RoutingRuleGroupVersionKind},
	{iamanonymousaccess.Setup, iamv1alpha1.AnonymousAccessGroupVersionKind},
	{iamldap.Setup, iamv1alpha1.LDAPGroupVersionKind},
	{iamprivilege.Setup, iamv1alpha1.PrivilegeGroupVersionKind},
	{iamrole.Setup, iamv1alpha1.RoleGroupVersionKind},
	{iamsaml.Setup, iamv1alpha1.SAMLGroupVersionKind},
	{iamuser.Setup, iamv1alpha1.UserGroupVersionKind},
	{iamutc.Setup, iamv1alpha1.UserTokenConfigurationGroupVersionKind},
	{instanceblobstore.Setup, instancev1alpha1.BlobStoreGroupVersionKind},
	{instancecapability.Setup, instancev1alpha1.CapabilityGroupVersionKind},
	{instanceiqserver.Setup, instancev1alpha1.IQServerConfigurationGroupVersionKind},
	{instancelicense.Setup, instancev1alpha1.LicenseGroupVersionKind},
	{instancesecurityrealm.Setup, instancev1alpha1.SecurityRealmGroupVersionKind},
	{instancessltruststore.Setup, instancev1alpha1.SecuritySSLTruststoreGroupVersionKind},
}

// Setup adds all Nexus controllers to the supplied manager. The ProviderConfig
// controllers start immediately. With a gate in opts, each managed resource
// controller only starts once its CustomResourceDefinition is established,
// which a gate controller watching those definitions reports.
func Setup(mgr ctrl.Manager, opts helpers.Options) error {
	err := config.Setup(mgr, opts)
	if err != nil {
		return errors.Wrap(err, "cannot setup the ProviderConfig controllers")
	}

	if opts.HasGate() {
		err = crdgate.Setup(mgr, opts.Options, managedKinds())
		if err != nil {
			return errors.Wrap(err, "cannot setup the CustomResourceDefinition gate controller")
		}
	}

	for _, managed := range managedControllers {
		err = helpers.SetupGated(mgr, opts, managed.setup, managed.gvk)
		if err != nil {
			return errors.Wrapf(err, "cannot setup the controller of %s", managed.gvk)
		}
	}

	return nil
}

// managedKinds returns the kinds of the managed resource controllers.
func managedKinds() []schema.GroupVersionKind {
	kinds := make([]schema.GroupVersionKind, 0, len(managedControllers))
	for _, managed := range managedControllers {
		kinds = append(kinds, managed.gvk)
	}

	return kinds
}
