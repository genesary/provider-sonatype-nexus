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
	iamsecurityrealm "github.com/genesary/provider-sonatype-nexus/internal/controller/iam/securityrealm"
	"github.com/genesary/provider-sonatype-nexus/internal/controller/ldap"
	"github.com/genesary/provider-sonatype-nexus/internal/controller/privilege"
	"github.com/genesary/provider-sonatype-nexus/internal/controller/repository"
	"github.com/genesary/provider-sonatype-nexus/internal/controller/role"
	"github.com/genesary/provider-sonatype-nexus/internal/controller/saml"
	"github.com/genesary/provider-sonatype-nexus/internal/controller/securityssltruststore"
	"github.com/genesary/provider-sonatype-nexus/internal/controller/user"
	"github.com/genesary/provider-sonatype-nexus/internal/controller/usertokenconfiguration"
)

// Setup creates all Nexus controllers and adds them to the supplied manager.
func Setup(mgr ctrl.Manager, opts controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		iamanonymousaccess.Setup,
		blobstore.Setup,
		config.Setup,
		contentcleanuppolicy.Setup,
		contentcontentselector.Setup,
		ldap.Setup,
		privilege.Setup,
		repository.Setup,
		role.Setup,
		saml.Setup,
		iamsecurityrealm.Setup,
		securityssltruststore.Setup,
		user.Setup,
		usertokenconfiguration.Setup,
	} {
		err := setup(mgr, opts)
		if err != nil {
			return errors.Wrap(err, "cannot setup controller")
		}
	}

	return nil
}
