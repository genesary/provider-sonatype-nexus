//go:build e2e

/*
Copyright 2026 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// fileServerImage serves the file, with the httpd applet.
	fileServerImage = "docker.io/library/busybox:1.37.0"
	// fileServerPort is the port the file is served on.
	fileServerPort = 8080
	// fileServerTimeout bounds starting and stopping the server, including
	// pulling its image.
	fileServerTimeout = 2 * time.Minute
	// fileServerScript serves $CONTENT behind basic authentication.
	fileServerScript = `mkdir /tmp/www && printf %s "$CONTENT" >/tmp/www/file &&
echo "/:$USERNAME:$PASSWORD" >/tmp/httpd.conf &&
exec httpd -f -p 8080 -h /tmp/www -c /tmp/httpd.conf`
)

// FileServer serves a single file over HTTP inside the cluster, where the
// provider can fetch it, behind basic authentication.
type FileServer struct {
	kube client.Client
	name string
}

// NewFileServer creates a FileServer that serves nothing until Serve is
// called. Everything it creates is deleted when the test ends.
func (f *Framework) NewFileServer(t *testing.T, name string) *FileServer {
	t.Helper()

	server := &FileServer{kube: f.Kube, name: name}

	for _, obj := range []client.Object{
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: server.SecretName(), Namespace: metav1.NamespaceDefault},
			StringData: map[string]string{"username": "e2e", "password": "e2e-password"},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: metav1.NamespaceDefault},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{"app": name},
				Ports:    []corev1.ServicePort{{Port: fileServerPort}},
			},
		},
	} {
		f.Apply(t, obj)
		t.Cleanup(func() { f.Delete(t, obj) })
	}

	t.Cleanup(func() {
		err := server.Stop()
		if err != nil {
			t.Logf("cleanup: stopping file server %s: %v", name, err)
		}
	})

	return server
}

// URL is the URL of the file inside the cluster.
func (s *FileServer) URL() string {
	return fmt.Sprintf("http://%s.%s.svc.cluster.local:%d/file", s.name, metav1.NamespaceDefault, fileServerPort)
}

// SecretName names the Secret, in the default namespace, that holds the
// username and the password the server requires.
func (s *FileServer) SecretName() string {
	return s.name + "-credentials"
}

// Serve serves content as the file, replacing what was served before. The
// server is unavailable while it is replaced.
func (s *FileServer) Serve(content string) error {
	ctx := context.Background()

	err := s.Stop()
	if err != nil {
		return err
	}

	fromSecret := func(env, key string) corev1.EnvVar {
		return corev1.EnvVar{Name: env, ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: s.SecretName()},
			Key:                  key,
		}}}
	}

	pod := s.pod()
	pod.Spec = corev1.PodSpec{
		TerminationGracePeriodSeconds: ptr.To[int64](0),
		Containers: []corev1.Container{{
			Name:    "httpd",
			Image:   fileServerImage,
			Command: []string{"sh", "-c", fileServerScript},
			Env: []corev1.EnvVar{
				{Name: "CONTENT", Value: content},
				fromSecret("USERNAME", "username"),
				fromSecret("PASSWORD", "password"),
			},
			ReadinessProbe: &corev1.Probe{
				ProbeHandler:  corev1.ProbeHandler{TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromInt32(fileServerPort)}},
				PeriodSeconds: 1,
			},
		}},
	}

	err = s.kube.Create(ctx, pod)
	if err != nil {
		return fmt.Errorf("creating the file server: %w", err)
	}

	return wait.PollUntilContextTimeout(ctx, pollInterval, fileServerTimeout, true, func(ctx context.Context) (bool, error) {
		err := s.kube.Get(ctx, client.ObjectKeyFromObject(pod), pod)
		if err != nil {
			return false, err
		}

		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady {
				return condition.Status == corev1.ConditionTrue, nil
			}
		}

		return false, nil
	})
}

// Stop makes the server unavailable until Serve is called.
func (s *FileServer) Stop() error {
	ctx := context.Background()
	pod := s.pod()

	err := s.kube.Delete(ctx, pod)
	if apierrors.IsNotFound(err) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("deleting the file server: %w", err)
	}

	return wait.PollUntilContextTimeout(ctx, pollInterval, fileServerTimeout, true, func(ctx context.Context) (bool, error) {
		err := s.kube.Get(ctx, client.ObjectKeyFromObject(pod), pod)

		return apierrors.IsNotFound(err), client.IgnoreNotFound(err)
	})
}

// pod returns the pod that serves the file, without its spec.
func (s *FileServer) pod() *corev1.Pod {
	return &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name:      s.name,
		Namespace: metav1.NamespaceDefault,
		Labels:    map[string]string{"app": s.name},
	}}
}
