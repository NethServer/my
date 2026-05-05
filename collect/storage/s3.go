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
// The client is configured via BACKUP_S3_* environment variables and exposes
// a singleton for use by the backup ingest and restore handlers. Virtual-
// hosted addressing is the default (required by Spaces and most managed S3
// providers); path-style can be forced for local S3 emulators via
// BACKUP_S3_USE_PATH_STYLE=true.
package storage

import (
	"context"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/nethesis/my/collect/configuration"
)

var (
	backupClient     *s3.Client
	backupPresigner  *s3.PresignClient
	backupClientErr  error
	backupClientOnce sync.Once
)

// BackupClient returns a singleton S3 client wired to the configured backup
// store, along with a presigner for generating short-lived download URLs.
// The first call initializes both; subsequent calls return the cached values.
func BackupClient(ctx context.Context) (*s3.Client, *s3.PresignClient, error) {
	backupClientOnce.Do(func() {
		backupClient, backupPresigner, backupClientErr = buildBackupClient(ctx)
	})
	return backupClient, backupPresigner, backupClientErr
}

func buildBackupClient(ctx context.Context) (*s3.Client, *s3.PresignClient, error) {
	if configuration.Config.S3Endpoint == "" {
		return nil, nil, fmt.Errorf("S3_ENDPOINT is not set")
	}
	if configuration.Config.S3AccessKey == "" || configuration.Config.S3SecretKey == "" {
		return nil, nil, fmt.Errorf("S3_ACCESS_KEY and S3_SECRET_KEY must be set")
	}

	creds := credentials.NewStaticCredentialsProvider(
		configuration.Config.S3AccessKey,
		configuration.Config.S3SecretKey,
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
		o.BaseEndpoint = aws.String(configuration.Config.S3Endpoint)
		o.UsePathStyle = configuration.Config.BackupS3UsePathStyle
	})

	presigner := s3.NewPresignClient(client)

	return client, presigner, nil
}
