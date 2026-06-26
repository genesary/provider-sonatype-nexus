package controller

import (
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/genesary/provider-sonatype-nexus/internal/controller/config"
	contentcleanuppolicy "github.com/genesary/provider-sonatype-nexus/internal/controller/content/cleanuppolicy"
	contentcontentselector "github.com/genesary/provider-sonatype-nexus/internal/controller/content/contentselector"
	contentrepository "github.com/genesary/provider-sonatype-nexus/internal/controller/content/repository"
	contentroutingrule "github.com/genesary/provider-sonatype-nexus/internal/controller/content/routingrule"
	iamanonymousaccess "github.com/genesary/provider-sonatype-nexus/internal/controller/iam/anonymousaccess"
	iamldap "github.com/genesary/provider-sonatype-nexus/internal/controller/iam/ldap"
	iamlicense "github.com/genesary/provider-sonatype-nexus/internal/controller/iam/license"
	iamprivilege "github.com/genesary/provider-sonatype-nexus/internal/controller/iam/privilege"
	iamrole "github.com/genesary/provider-sonatype-nexus/internal/controller/iam/role"
	iamsaml "github.com/genesary/provider-sonatype-nexus/internal/controller/iam/saml"
	iamsecurityrealm "github.com/genesary/provider-sonatype-nexus/internal/controller/iam/securityrealm"
	iamssltruststore "github.com/genesary/provider-sonatype-nexus/internal/controller/iam/securityssltruststore"
	iamuser "github.com/genesary/provider-sonatype-nexus/internal/controller/iam/user"
	iamutc "github.com/genesary/provider-sonatype-nexus/internal/controller/iam/usertokenconfiguration"
	instanceblobstore "github.com/genesary/provider-sonatype-nexus/internal/controller/instance/blobstore"
	instancecapability "github.com/genesary/provider-sonatype-nexus/internal/controller/instance/capability"
	instanceiqserver "github.com/genesary/provider-sonatype-nexus/internal/controller/instance/iqserverconfiguration"
)

// Setup creates all Nexus controllers and adds them to the supplied manager.
func Setup(mgr ctrl.Manager, opts controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		instanceblobstore.Setup,
		config.Setup,
		instancecapability.Setup,
		instanceiqserver.Setup,
		contentcleanuppolicy.Setup,
		contentcontentselector.Setup,
		contentroutingrule.Setup,
		iamanonymousaccess.Setup,
		iamlicense.Setup,
		iamldap.Setup,
		iamprivilege.Setup,
		iamrole.Setup,
		iamsaml.Setup,
		iamsecurityrealm.Setup,
		iamssltruststore.Setup,
		iamuser.Setup,
		iamutc.Setup,
		contentrepository.Setup,
	} {
		err := setup(mgr, opts)
		if err != nil {
			return errors.Wrap(err, "cannot setup controller")
		}
	}

	return nil
}
