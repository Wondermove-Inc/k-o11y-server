// Package k8s provides utilities for creating and managing Kubernetes Jobs
// from within a K8s pod (in-cluster configuration).
package k8s

import (
	"context"
	"fmt"
	"os"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// S3ActivateJobConfig holds the configuration for creating an S3 activate Job.
type S3ActivateJobConfig struct {
	Namespace          string // K8s namespace (default: current pod namespace)
	Image              string // Job container image
	CHHost             string // ClickHouse VM host
	CHSSHUser          string // SSH user on CH VM
	CHSSHPort          string // SSH port
	CHPassword         string // ClickHouse password
	SSHKeySecretName   string // K8s Secret name containing SSH private key
	EncryptionKey      string // K_O11Y_ENCRYPTION_KEY
	Mode               string // "activate" or "apply"
}

// JobStatus represents the current state of a K8s Job.
type JobStatus struct {
	Name      string `json:"name"`
	Active    bool   `json:"active"`
	Succeeded bool   `json:"succeeded"`
	Failed    bool   `json:"failed"`
	Message   string `json:"message"`
}

// Client wraps the Kubernetes clientset.
type Client struct {
	clientset *kubernetes.Clientset
	namespace string
}

// NewInClusterClient creates a K8s client using in-cluster ServiceAccount token.
func NewInClusterClient() (*Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s clientset: %w", err)
	}

	ns := os.Getenv("POD_NAMESPACE")
	if ns == "" {
		nsBytes, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
		if err != nil {
			ns = "signoz"
		} else {
			ns = string(nsBytes)
		}
	}

	return &Client{clientset: clientset, namespace: ns}, nil
}

// CreateS3ActivateJob creates a K8s Job that SSHes to the CH VM and activates S3.
func (c *Client) CreateS3ActivateJob(ctx context.Context, cfg S3ActivateJobConfig) (string, error) {
	if cfg.Namespace == "" {
		cfg.Namespace = c.namespace
	}
	if cfg.CHSSHUser == "" {
		cfg.CHSSHUser = "ko11y"
	}
	if cfg.CHSSHPort == "" {
		cfg.CHSSHPort = "22"
	}
	if cfg.Mode == "" {
		cfg.Mode = "activate"
	}

	jobName := fmt.Sprintf("s3-%s-%s", cfg.Mode, time.Now().Format("20060102-150405"))

	// Check if a similar job is already running
	existing, err := c.GetS3JobStatus(ctx, cfg.Namespace)
	if err == nil && existing != nil && existing.Active {
		return "", fmt.Errorf("s3 activation job already running: %s", existing.Name)
	}

	backoffLimit := int32(0)
	ttlSeconds := int32(600) // Job auto-cleanup after 10 minutes
	sshKeyMode := int32(0400)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: cfg.Namespace,
			Labels: map[string]string{
				"app":       "ko11y-s3-activate",
				"ko11y/s3": cfg.Mode,
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttlSeconds,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":       "ko11y-s3-activate",
						"ko11y/s3": cfg.Mode,
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Volumes: []corev1.Volume{
						{
							Name: "ssh-key",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName:  cfg.SSHKeySecretName,
									DefaultMode: &sshKeyMode,
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "s3-activate",
							Image: cfg.Image,
							Env: []corev1.EnvVar{
								{Name: "CH_HOST", Value: cfg.CHHost},
								{Name: "CH_SSH_USER", Value: cfg.CHSSHUser},
								{Name: "CH_SSH_PORT", Value: cfg.CHSSHPort},
								{Name: "CH_PASSWORD", Value: cfg.CHPassword},
								{Name: "K_O11Y_ENCRYPTION_KEY", Value: cfg.EncryptionKey},
								{Name: "SSH_KEY_PATH", Value: "/etc/ssh-key/ssh-privatekey"},
								{Name: "MODE", Value: cfg.Mode},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "ssh-key",
									MountPath: "/etc/ssh-key",
									ReadOnly:  true,
								},
							},
						},
					},
				},
			},
		},
	}

	created, err := c.clientset.BatchV1().Jobs(cfg.Namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create job: %w", err)
	}

	return created.Name, nil
}

// GetS3JobStatus returns the status of the most recent s3-activate Job.
func (c *Client) GetS3JobStatus(ctx context.Context, namespace string) (*JobStatus, error) {
	if namespace == "" {
		namespace = c.namespace
	}

	jobs, err := c.clientset.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app=ko11y-s3-activate",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	if len(jobs.Items) == 0 {
		return nil, nil
	}

	// Find the most recent job
	var latest *batchv1.Job
	for i := range jobs.Items {
		j := &jobs.Items[i]
		if latest == nil || j.CreationTimestamp.After(latest.CreationTimestamp.Time) {
			latest = j
		}
	}

	status := &JobStatus{
		Name:   latest.Name,
		Active: latest.Status.Active > 0,
	}

	if latest.Status.Succeeded > 0 {
		status.Succeeded = true
		status.Message = "S3 activation completed successfully"
	} else if latest.Status.Failed > 0 {
		status.Failed = true
		status.Message = "S3 activation failed"
		// Try to get failure reason from conditions
		for _, cond := range latest.Status.Conditions {
			if cond.Type == batchv1.JobFailed {
				status.Message = fmt.Sprintf("S3 activation failed: %s", cond.Message)
			}
		}
	} else if status.Active {
		status.Message = "S3 activation in progress..."
	}

	return status, nil
}

// DeleteS3Jobs deletes completed s3-activate Jobs.
func (c *Client) DeleteS3Jobs(ctx context.Context, namespace string) error {
	if namespace == "" {
		namespace = c.namespace
	}

	propagation := metav1.DeletePropagationBackground
	return c.clientset.BatchV1().Jobs(namespace).DeleteCollection(ctx,
		metav1.DeleteOptions{PropagationPolicy: &propagation},
		metav1.ListOptions{LabelSelector: "app=ko11y-s3-activate"},
	)
}
