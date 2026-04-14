/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

// Package storage provides an S3 client for the DigitalOcean Spaces bucket
// that holds appliance configuration backups.
//
// The backend consumes the bucket read-only: it lists backups for a system,
// fetches object metadata for rich UI rendering, and generates short-lived
// presigned URLs so user browsers can download objects directly from Spaces
// without streaming through the API.
//
// When BACKUP_S3_PRESIGN_ENDPOINT is set, the presigner is built on a
// separate client that signs URLs with that endpoint instead of the one
// used for server-side calls. This matters only for local development
// where backend runs inside a compose network and talks to MinIO over the
// internal hostname, while the browser can only reach MinIO via a host
// port mapping; using two endpoints keeps signatures valid in both worlds.
package storage

import (
	"context"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/nethesis/my/backend/configuration"
)

var (
	backupClient     *s3.Client
	backupPresigner  *s3.PresignClient
	backupClientErr  error
	backupClientOnce sync.Once
)

// BackupClient returns a singleton S3 client + presigner wired to the
// configured backup store.
func BackupClient(ctx context.Context) (*s3.Client, *s3.PresignClient, error) {
	backupClientOnce.Do(func() {
		backupClient, backupPresigner, backupClientErr = buildBackupClient(ctx)
	})
	return backupClient, backupPresigner, backupClientErr
}

func buildBackupClient(ctx context.Context) (*s3.Client, *s3.PresignClient, error) {
	if configuration.Config.BackupS3Endpoint == "" {
		return nil, nil, fmt.Errorf("BACKUP_S3_ENDPOINT is not set")
	}
	if configuration.Config.BackupS3AccessKey == "" || configuration.Config.BackupS3SecretKey == "" {
		return nil, nil, fmt.Errorf("BACKUP_S3_ACCESS_KEY and BACKUP_S3_SECRET_KEY must be set")
	}

	creds := credentials.NewStaticCredentialsProvider(
		configuration.Config.BackupS3AccessKey,
		configuration.Config.BackupS3SecretKey,
		"",
	)

	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(configuration.Config.BackupS3Region),
		awsconfig.WithCredentialsProvider(creds),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("load S3 config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(configuration.Config.BackupS3Endpoint)
		o.UsePathStyle = configuration.Config.BackupS3UsePathStyle
	})

	presignClient := client
	if override := configuration.Config.BackupS3PresignEndpoint; override != "" {
		presignClient = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(override)
			o.UsePathStyle = configuration.Config.BackupS3UsePathStyle
		})
	}
	presigner := s3.NewPresignClient(presignClient)

	return client, presigner, nil
}
