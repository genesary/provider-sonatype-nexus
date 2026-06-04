package v1alpha1

import (
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LDAPParameters defines the desired state of an LDAP server configuration.
type LDAPParameters struct {
	// Name of the LDAP server.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Protocol to use for connecting to the LDAP server.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=ldap;ldaps
	Protocol string `json:"protocol"`

	// Host is the LDAP server hostname.
	// +kubebuilder:validation:Required
	Host string `json:"host"`

	// Port is the LDAP server port.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int32 `json:"port"`

	// SearchBase is the LDAP location to be added to the connection URL.
	// +kubebuilder:validation:Required
	SearchBase string `json:"searchBase"`

	// AuthScheme is the authentication scheme.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=none;simple;DIGEST_MD5;CRAM_MD5
	AuthScheme string `json:"authScheme"`

	// AuthUsername is the username for LDAP authentication (for simple auth).
	// +optional
	AuthUsername *string `json:"authUsername,omitempty"`

	// AuthPasswordSecretRef is a reference to a secret containing the auth password.
	// +optional
	AuthPasswordSecretRef *xpv1.SecretKeySelector `json:"authPasswordSecretRef,omitempty"`

	// AuthRealm is the SASL realm for DIGEST-MD5/CRAM-MD5 authentication.
	// +optional
	AuthRealm *string `json:"authRealm,omitempty"`

	// ConnectionTimeoutSeconds is the timeout for LDAP connections.
	// +kubebuilder:default=30
	// +optional
	ConnectionTimeoutSeconds *int32 `json:"connectionTimeoutSeconds,omitempty"`

	// ConnectionRetryDelaySeconds is the delay between connection retries.
	// +kubebuilder:default=300
	// +optional
	ConnectionRetryDelaySeconds *int32 `json:"connectionRetryDelaySeconds,omitempty"`

	// MaxIncidentCount is the maximum number of connection retries.
	// +kubebuilder:default=3
	// +optional
	MaxIncidentCount *int32 `json:"maxIncidentCount,omitempty"`

	// UseTrustStore determines if the truststore should be used.
	// +optional
	UseTrustStore *bool `json:"useTrustStore,omitempty"`

	// UserBaseDN is the relative DN where user objects are found.
	// +kubebuilder:validation:Required
	UserBaseDN string `json:"userBaseDn"`

	// UserSubtree determines if users are located in structures below the user base DN.
	// +optional
	UserSubtree *bool `json:"userSubtree,omitempty"`

	// UserObjectClass is the LDAP class for user objects.
	// +kubebuilder:default="inetOrgPerson"
	// +optional
	UserObjectClass *string `json:"userObjectClass,omitempty"`

	// UserIDAttribute is the attribute used to identify users.
	// +kubebuilder:default="uid"
	// +optional
	UserIDAttribute *string `json:"userIdAttribute,omitempty"`

	// UserRealNameAttribute is the attribute for user real names.
	// +kubebuilder:default="cn"
	// +optional
	UserRealNameAttribute *string `json:"userRealNameAttribute,omitempty"`

	// UserEmailAddressAttribute is the attribute for user email addresses.
	// +kubebuilder:default="mail"
	// +optional
	UserEmailAddressAttribute *string `json:"userEmailAddressAttribute,omitempty"`

	// UserPasswordAttribute is the attribute for user passwords.
	// +optional
	UserPasswordAttribute *string `json:"userPasswordAttribute,omitempty"`

	// UserMemberOfAttribute is the attribute storing group DNs in user objects.
	// +optional
	UserMemberOfAttribute *string `json:"userMemberOfAttribute,omitempty"`

	// UserLDAPFilter is an additional filter to limit user results.
	// +optional
	UserLDAPFilter *string `json:"userLdapFilter,omitempty"`

	// LDAPGroupsAsRoles determines if LDAP groups should be used as Nexus roles.
	// +optional
	LDAPGroupsAsRoles *bool `json:"ldapGroupsAsRoles,omitempty"`

	// GroupType is the type of group mapping (static or dynamic).
	// +kubebuilder:validation:Enum=static;dynamic
	// +optional
	GroupType *string `json:"groupType,omitempty"`

	// GroupBaseDN is the relative DN where group objects are found.
	// +optional
	GroupBaseDN *string `json:"groupBaseDn,omitempty"`

	// GroupSubtree determines if groups are located in structures below the group base DN.
	// +optional
	GroupSubtree *bool `json:"groupSubtree,omitempty"`

	// GroupObjectClass is the LDAP class for group objects.
	// +kubebuilder:default="groupOfUniqueNames"
	// +optional
	GroupObjectClass *string `json:"groupObjectClass,omitempty"`

	// GroupIDAttribute is the attribute defining the Group ID.
	// +kubebuilder:default="cn"
	// +optional
	GroupIDAttribute *string `json:"groupIdAttribute,omitempty"`

	// GroupMemberAttribute is the attribute containing group member usernames.
	// +kubebuilder:default="uniqueMember"
	// +optional
	GroupMemberAttribute *string `json:"groupMemberAttribute,omitempty"`

	// GroupMemberFormat is the format of user IDs in the group member attribute.
	// +kubebuilder:default="uid=${username},ou=people,dc=example,dc=com"
	// +optional
	GroupMemberFormat *string `json:"groupMemberFormat,omitempty"`
}

// LDAPObservation represents the observed state of an LDAP server configuration.
type LDAPObservation struct {
	// ID is the internal LDAP server ID.
	ID *string `json:"id,omitempty"`
}

// LDAPSpec defines the desired state of LDAP.
type LDAPSpec struct {
	xpv1.ResourceSpec `json:",inline"`

	ForProvider LDAPParameters `json:"forProvider"`
}

// LDAPStatus defines the observed state of LDAP.
type LDAPStatus struct {
	xpv1.ResourceStatus `json:",inline"`

	AtProvider LDAPObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="HOST",type="string",JSONPath=".spec.forProvider.host"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,nexus}

// LDAP is the Schema for the ldaps API.
type LDAP struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LDAPSpec   `json:"spec"`
	Status LDAPStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LDAPList contains a list of LDAP.
type LDAPList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []LDAP `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LDAP{}, &LDAPList{})
}
