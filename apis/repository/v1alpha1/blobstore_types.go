package v1alpha1

import (
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BlobStoreParameters defines the desired state of a BlobStore.
type BlobStoreParameters struct {
	// Name of the blob store.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Type of the blob store (File or S3).
	// +kubebuilder:validation:Enum=File;S3
	// +kubebuilder:default=File
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
	// +kubebuilder:validation:Optional
	ExpirationDays *int32 `json:"expirationDays,omitempty"`

	// AccessKeyIDSecretRef is a reference to a secret containing the access key ID.
	// +kubebuilder:validation:Optional
	AccessKeyIDSecretRef *xpv2.SecretKeySelector `json:"accessKeyIdSecretRef,omitempty"`

	// SecretAccessKeySecretRef is a reference to a secret containing the secret access key.
	// +kubebuilder:validation:Optional
	SecretAccessKeySecretRef *xpv2.SecretKeySelector `json:"secretAccessKeySecretRef,omitempty"`

	// AssumeRole for S3 access.
	// +kubebuilder:validation:Optional
	AssumeRole *string `json:"assumeRole,omitempty"`

	// SessionToken for S3 access.
	// +kubebuilder:validation:Optional
	SessionTokenSecretRef *xpv2.SecretKeySelector `json:"sessionTokenSecretRef,omitempty"`

	// ForcePathStyle enables path-style access for S3.
	// +kubebuilder:validation:Optional
	ForcePathStyle *bool `json:"forcePathStyle,omitempty"`
}

// BlobStoreObservation represents the observed state of a BlobStore.
type BlobStoreObservation struct {
	// AvailableSpaceInBytes is the available space in bytes.
	AvailableSpaceInBytes *int64 `json:"availableSpaceInBytes,omitempty"`

	// TotalSizeInBytes is the total size in bytes.
	TotalSizeInBytes *int64 `json:"totalSizeInBytes,omitempty"`

	// BlobCount is the number of blobs in the store.
	BlobCount *int64 `json:"blobCount,omitempty"`

	// Path is the observed filesystem path (File type only).
	Path *string `json:"path,omitempty"`

	// SoftQuotaType is the observed soft quota type.
	SoftQuotaType *string `json:"softQuotaType,omitempty"`

	// SoftQuotaLimit is the observed soft quota limit in bytes.
	SoftQuotaLimit *int64 `json:"softQuotaLimit,omitempty"`

	// BucketName is the observed S3 bucket name (S3 type only).
	BucketName *string `json:"bucketName,omitempty"`

	// BucketRegion is the observed S3 bucket region (S3 type only).
	BucketRegion *string `json:"bucketRegion,omitempty"`

	// BucketPrefix is the observed S3 bucket prefix (S3 type only).
	BucketPrefix *string `json:"bucketPrefix,omitempty"`
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

// init registers this type with the SchemeBuilder.
func init() {
	SchemeBuilder.Register(&BlobStore{}, &BlobStoreList{})
}
