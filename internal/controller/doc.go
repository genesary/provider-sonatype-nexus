// Package controller contains all Crossplane controllers for managing
// Sonatype Nexus resources.
//
// Each subdirectory contains a controller for a specific resource type:
//   - anonymousaccess: AnonymousAccess resource
//   - blobstore: BlobStore resource
//   - contentselector: ContentSelector resource
//   - ldap: LDAP resource
//   - privilege: Privilege resource
//   - repository: Repository resource (supports multiple formats)
//   - role: Role resource
//   - saml: SAML resource
//   - securityrealm: SecurityRealm resource
//   - user: User resource
//   - usertokenconfiguration: UserTokenConfiguration resource
//
// To add a new resource:
//  1. Create a new subdirectory with the resource name
//  2. Implement the controller following existing patterns
//  3. Add the Setup function to Setup() in register.go
package controller
