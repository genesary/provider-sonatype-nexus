package controller

import (
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/genesary/provider-sonatype-nexus/internal/controller/config"
	contentcleanuppolicy "github.com/genesary/provider-sonatype-nexus/internal/controller/content/cleanuppolicy"
	contentcontentselector "github.com/genesary/provider-sonatype-nexus/internal/controller/content/contentselector"
	contentrepository "github.com/genesary/provider-sonatype-nexus/internal/controller/content/repository"
	contentroutingrule "github.com/genesary/provider-sonatype-nexus/internal/controller/content/routingrule"
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

// Setup adds all Nexus controllers to the supplied manager. The managed
// resource controllers each start once their CustomResourceDefinition is
// established; the ProviderConfig controllers start immediately.
func Setup(mgr ctrl.Manager, opts helpers.Options) error {
	for _, setup := range []func(ctrl.Manager, helpers.Options) error{
		config.Setup,
		contentcleanuppolicy.SetupGated,
		contentcontentselector.SetupGated,
		contentrepository.SetupGated,
		contentroutingrule.SetupGated,
		iamanonymousaccess.SetupGated,
		iamldap.SetupGated,
		iamprivilege.SetupGated,
		iamrole.SetupGated,
		iamsaml.SetupGated,
		iamuser.SetupGated,
		iamutc.SetupGated,
		instanceblobstore.SetupGated,
		instancecapability.SetupGated,
		instanceiqserver.SetupGated,
		instancelicense.SetupGated,
		instancesecurityrealm.SetupGated,
		instancessltruststore.SetupGated,
	} {
		err := setup(mgr, opts)
		if err != nil {
			return errors.Wrap(err, "cannot setup controller")
		}
	}

	return nil
}
