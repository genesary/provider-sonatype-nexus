package controller

import (
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/AYDEV-FR/provider-sonatype-nexus/internal/controller/anonymousaccess"
	"github.com/AYDEV-FR/provider-sonatype-nexus/internal/controller/blobstore"
	"github.com/AYDEV-FR/provider-sonatype-nexus/internal/controller/contentselector"
	"github.com/AYDEV-FR/provider-sonatype-nexus/internal/controller/ldap"
	"github.com/AYDEV-FR/provider-sonatype-nexus/internal/controller/privilege"
	"github.com/AYDEV-FR/provider-sonatype-nexus/internal/controller/repository"
	"github.com/AYDEV-FR/provider-sonatype-nexus/internal/controller/role"
	"github.com/AYDEV-FR/provider-sonatype-nexus/internal/controller/saml"
	"github.com/AYDEV-FR/provider-sonatype-nexus/internal/controller/securityrealm"
	"github.com/AYDEV-FR/provider-sonatype-nexus/internal/controller/user"
	"github.com/AYDEV-FR/provider-sonatype-nexus/internal/controller/usertokenconfiguration"
)

// Setup creates all Nexus controllers and adds them to the supplied manager.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	// List of all controller setup functions
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		anonymousaccess.Setup,
		blobstore.Setup,
		contentselector.Setup,
		ldap.Setup,
		privilege.Setup,
		repository.Setup,
		role.Setup,
		saml.Setup,
		securityrealm.Setup,
		user.Setup,
		usertokenconfiguration.Setup,
	} {
		if err := setup(mgr, o); err != nil {
			return errors.Wrap(err, "cannot setup controller")
		}
	}
	return nil
}
