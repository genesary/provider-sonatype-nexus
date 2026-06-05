package controller

import (
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/genesary/provider-sonatype-nexus/internal/controller/blobstore"
	"github.com/genesary/provider-sonatype-nexus/internal/controller/config"
	contentcleanuppolicy "github.com/genesary/provider-sonatype-nexus/internal/controller/content/cleanuppolicy"
	contentcontentselector "github.com/genesary/provider-sonatype-nexus/internal/controller/content/contentselector"
	iamanonymousaccess "github.com/genesary/provider-sonatype-nexus/internal/controller/iam/anonymousaccess"
	iamrole "github.com/genesary/provider-sonatype-nexus/internal/controller/iam/role"
	iamsecurityrealm "github.com/genesary/provider-sonatype-nexus/internal/controller/iam/securityrealm"
	iamutc "github.com/genesary/provider-sonatype-nexus/internal/controller/iam/usertokenconfiguration"
	"github.com/genesary/provider-sonatype-nexus/internal/controller/ldap"
	"github.com/genesary/provider-sonatype-nexus/internal/controller/privilege"
	"github.com/genesary/provider-sonatype-nexus/internal/controller/repository"
	"github.com/genesary/provider-sonatype-nexus/internal/controller/saml"
	"github.com/genesary/provider-sonatype-nexus/internal/controller/securityssltruststore"
	"github.com/genesary/provider-sonatype-nexus/internal/controller/user"
)

// Setup creates all Nexus controllers and adds them to the supplied manager.
func Setup(mgr ctrl.Manager, opts controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		blobstore.Setup,
		config.Setup,
		contentcleanuppolicy.Setup,
		contentcontentselector.Setup,
		iamanonymousaccess.Setup,
		iamsecurityrealm.Setup,
		iamutc.Setup,
		ldap.Setup,
		privilege.Setup,
		repository.Setup,
		iamrole.Setup,
		saml.Setup,
		securityssltruststore.Setup,
		user.Setup,
	} {
		err := setup(mgr, opts)
		if err != nil {
			return errors.Wrap(err, "cannot setup controller")
		}
	}

	return nil
}
