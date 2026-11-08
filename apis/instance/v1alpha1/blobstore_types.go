package v1alpha1

import (
	"reflect"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// BlobStoreParameters defines the desired state of a BlobStore.
// +kubebuilder:validation:XValidation:rule="self.type != 'Azure' || has(self.azureConfig)",message="azureConfig is required when type is Azure."
// +kubebuilder:validation:XValidation:rule="self.type == 'Azure' || !has(self.azureConfig)",message="azureConfig is only allowed when type is Azure."
type BlobStoreParameters struct {
	// Name of the blob store.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Name is immutable."
	Name string `json:"name"`

	// Type of the blob store (File, S3 or Azure). Azure blob stores require
	// Nexus Repository Pro.
	// +kubebuilder:validation:Enum=File;S3;Azure
	// +kubebuilder:default=File
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Type is immutable."
	Type string `json:"type"`

	// Path for File type blob store.
	// +kubebuilder:validation:Optional
	Path *string `json:"path,omitempty"`

	// SoftQuota defines the soft quota configuration.
	// +kubebuilder:validation:Optional
	SoftQuota *SoftQuota `json:"softQuota,omitempty"`

	// S3Config defines S3 configuration for S3 type blob store.
	// +kubebuilder:validation:Optional
	S3Config *S3Config `json:"s3Config,omitempty"`

	// AzureConfig defines the Azure Blob Storage configuration of an Azure
	// type blob store. Required when the type is Azure.
	// +kubebuilder:validation:Optional
	AzureConfig *AzureConfig `json:"azureConfig,omitempty"`
}

// AzureConfig defines the Azure Blob Storage configuration of an Azure blob
// store.
// +kubebuilder:validation:XValidation:rule="self.authenticationMethod != 'ACCOUNTKEY' || has(self.accountKeySecretRef)",message="accountKeySecretRef is required when authenticationMethod is ACCOUNTKEY."
// +kubebuilder:validation:XValidation:rule="self.authenticationMethod == 'ACCOUNTKEY' || !has(self.accountKeySecretRef)",message="accountKeySecretRef is only allowed when authenticationMethod is ACCOUNTKEY."
type AzureConfig struct {
	// AccountName is the Azure storage account name.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	AccountName string `json:"accountName"`

	// ContainerName is the name of an existing container of the storage
	// account, which holds the blobs.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[a-z0-9][a-z0-9-]{2,62}$`
	ContainerName string `json:"containerName"`

	// AuthenticationMethod selects how Nexus authenticates to Azure:
	// ACCOUNTKEY with the account key in accountKeySecretRef,
	// MANAGEDIDENTITY with the managed identity of the Nexus host, or
	// ENVIRONMENTVARIABLE with the credentials in the Nexus environment.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=ACCOUNTKEY;MANAGEDIDENTITY;ENVIRONMENTVARIABLE
	AuthenticationMethod string `json:"authenticationMethod"`

	// AccountKeySecretRef references the Kubernetes Secret key holding the
	// storage account key. The secret must be in the same namespace as the
	// BlobStore. Required when the authentication method is ACCOUNTKEY.
	// +kubebuilder:validation:Optional
	AccountKeySecretRef *xpv2.LocalSecretKeySelector `json:"accountKeySecretRef,omitempty"`

	// PreSignedURLEnabled enables direct downloads from Azure through SAS
	// URLs. If unset, Nexus applies its default and the setting is not
	// checked for drift.
	// +kubebuilder:validation:Optional
	PreSignedURLEnabled *bool `json:"preSignedUrlEnabled,omitempty"`
}

// SoftQuota defines a soft quota configuration for blob stores.
type SoftQuota struct {
	// Type of the soft quota.
	// +kubebuilder:validation:Enum=spaceRemainingQuota;spaceUsedQuota
	// +kubebuilder:validation:Optional
	Type *string `json:"type,omitempty"`

	// Limit in bytes.
	// +kubebuilder:validation:Optional
	Limit *int64 `json:"limit,omitempty"`
}

// S3Config defines S3 configuration for S3 type blob stores.
type S3Config struct {
	// Bucket name.
	// +kubebuilder:validation:Required
	Bucket string `json:"bucket"`

	// Prefix for objects in the bucket.
	// +kubebuilder:validation:Optional
	Prefix *string `json:"prefix,omitempty"`

	// Region of the S3 bucket.
	// +kubebuilder:validation:Optional
	Region *string `json:"region,omitempty"`

	// Endpoint URL for S3 compatible storage.
	// +kubebuilder:validation:Optional
	Endpoint *string `json:"endpoint,omitempty"`

	// Expiration days for objects.
	//
	// Deprecated: Nexus no longer accepts a bucket expiration, so this field
	// is ignored.
	//
	// +kubebuilder:validation:Optional
	ExpirationDays *int32 `json:"expirationDays,omitempty"`

	// AccessKeyIDSecretRef is a reference to a secret containing the access key ID.
	// +kubebuilder:validation:Optional
	AccessKeyIDSecretRef *xpv2.LocalSecretKeySelector `json:"accessKeyIdSecretRef,omitempty"`

	// SecretAccessKeySecretRef is a reference to a secret containing the secret access key.
	// +kubebuilder:validation:Optional
	SecretAccessKeySecretRef *xpv2.LocalSecretKeySelector `json:"secretAccessKeySecretRef,omitempty"`

	// AssumeRole for S3 access.
	// +kubebuilder:validation:Optional
	AssumeRole *string `json:"assumeRole,omitempty"`

	// SessionToken for S3 access.
	// +kubebuilder:validation:Optional
	SessionTokenSecretRef *xpv2.LocalSecretKeySelector `json:"sessionTokenSecretRef,omitempty"`

	// ForcePathStyle enables path-style access for S3.
	// +kubebuilder:validation:Optional
	ForcePathStyle *bool `json:"forcePathStyle,omitempty"`
}

// BlobStoreObservation represents the observed state of a BlobStore.
type BlobStoreObservation struct {
	// Name is the observed blob store name.
	Name string `json:"name,omitempty"`

	// Type is the observed blob store type, as reported by the blob store
	// listing.
	Type string `json:"type,omitempty"`

	// AvailableSpaceInBytes is the free space reported by the blob store.
	AvailableSpaceInBytes *int64 `json:"availableSpaceInBytes,omitempty"`

	// TotalSizeInBytes is the total size reported by the blob store.
	TotalSizeInBytes *int64 `json:"totalSizeInBytes,omitempty"`

	// BlobCount is the number of blobs held by the blob store.
	BlobCount *int64 `json:"blobCount,omitempty"`

	// Path is the observed path of a File blob store.
	Path *string `json:"path,omitempty"`

	// SoftQuotaType is the observed soft quota type.
	SoftQuotaType *string `json:"softQuotaType,omitempty"`

	// SoftQuotaLimit is the observed soft quota limit.
	SoftQuotaLimit *int64 `json:"softQuotaLimit,omitempty"`

	// BucketName is the observed bucket name of an S3 blob store.
	BucketName *string `json:"bucketName,omitempty"`

	// BucketRegion is the observed bucket region of an S3 blob store.
	BucketRegion *string `json:"bucketRegion,omitempty"`

	// BucketPrefix is the observed bucket prefix of an S3 blob store.
	BucketPrefix *string `json:"bucketPrefix,omitempty"`

	// BucketExpirationDays is the observed number of days before deleted
	// blobs expire from the S3 bucket.
	//
	// Deprecated: Nexus no longer reports a bucket expiration, so this field
	// is never set.
	BucketExpirationDays *int32 `json:"bucketExpirationDays,omitempty"`

	// AccountName is the observed storage account of an Azure blob store.
	AccountName *string `json:"accountName,omitempty"`

	// ContainerName is the observed container of an Azure blob store.
	ContainerName *string `json:"containerName,omitempty"`

	// AuthenticationMethod is the observed authentication method of an Azure
	// blob store.
	AuthenticationMethod *string `json:"authenticationMethod,omitempty"`

	// PreSignedURLEnabled is whether the observed Azure blob store serves
	// downloads through SAS URLs.
	PreSignedURLEnabled *bool `json:"preSignedUrlEnabled,omitempty"`

	// CredentialsFingerprint identifies the Azure account key last applied
	// to Nexus, which Nexus never reports back. It changes when the
	// referenced Secret changes or is repointed, which triggers an update.
	CredentialsFingerprint string `json:"credentialsFingerprint,omitempty"`
}

// BlobStoreSpec defines the desired state of BlobStore.
type BlobStoreSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`

	ForProvider BlobStoreParameters `json:"forProvider"`
}

// BlobStoreStatus defines the observed state of BlobStore.
type BlobStoreStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`

	AtProvider BlobStoreObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,nexus}

// BlobStore is the Schema for the blobstores API.
type BlobStore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BlobStoreSpec   `json:"spec"`
	Status BlobStoreStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BlobStoreList contains a list of BlobStore.
type BlobStoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []BlobStore `json:"items"`
}

// BlobStore type metadata.
var (
	BlobStoreKind             = reflect.TypeFor[BlobStore]().Name()
	BlobStoreGroupKind        = schema.GroupKind{Group: APIGroup, Kind: BlobStoreKind}.String()
	BlobStoreKindAPIVersion   = BlobStoreKind + "." + SchemeGroupVersion.String()
	BlobStoreGroupVersionKind = SchemeGroupVersion.WithKind(BlobStoreKind)
)

// init registers this type with the SchemeBuilder.
func init() {
	SchemeBuilder.Register(&BlobStore{}, &BlobStoreList{})
}
