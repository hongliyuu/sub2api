package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

func (r *userRepository) FindExternalIdentity(ctx context.Context, provider, providerUserID string) (*service.UserExternalIdentity, error) {
	exec := r.profileSQLExecutor(ctx)
	if exec == nil {
		return nil, errors.New("sql executor is not configured")
	}
	query := `
		SELECT
			id, user_id, provider, provider_user_id, provider_union_id,
			provider_username, display_name, profile_url, avatar_url, metadata,
			created_at, updated_at
		FROM user_external_identities
		WHERE provider = $1
		  AND (provider_user_id = $2 OR provider_union_id = $2)
		ORDER BY updated_at DESC
		LIMIT 1`

	rows, err := exec.QueryContext(ctx, query, provider, providerUserID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return nil, service.ErrExternalIdentityNotFound
	}

	var (
		identity    service.UserExternalIdentity
		unionID     sql.NullString
		metadataRaw []byte
	)
	if err := rows.Scan(
		&identity.ID,
		&identity.UserID,
		&identity.Provider,
		&identity.ProviderUserID,
		&unionID,
		&identity.ProviderUsername,
		&identity.DisplayName,
		&identity.ProfileURL,
		&identity.AvatarURL,
		&metadataRaw,
		&identity.CreatedAt,
		&identity.UpdatedAt,
	); err != nil {
		return nil, err
	}
	identity.ProviderUnionID = nullStringPtr(unionID)
	identity.Metadata = unmarshalMetadata(metadataRaw)
	return &identity, nil
}

func (r *userRepository) ListExternalIdentities(ctx context.Context, userID int64) ([]service.UserExternalIdentity, error) {
	identitiesByUser, err := r.loadExternalIdentities(ctx, []int64{userID})
	if err != nil {
		return nil, err
	}
	return identitiesByUser[userID], nil
}

func (r *userRepository) UpsertExternalIdentity(ctx context.Context, userID int64, input service.UpsertUserExternalIdentityInput) (*service.UserExternalIdentity, error) {
	exec := r.profileSQLExecutor(ctx)
	if exec == nil {
		return nil, errors.New("sql executor is not configured")
	}

	metadataJSON, err := json.Marshal(normalizeMetadata(input.Metadata))
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO user_external_identities (
			user_id, provider, provider_user_id, provider_union_id,
			provider_username, display_name, profile_url, avatar_url, metadata
		) VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6, $7, $8, $9::jsonb)
		ON CONFLICT (user_id, provider) DO UPDATE SET
			provider_user_id = EXCLUDED.provider_user_id,
			provider_union_id = EXCLUDED.provider_union_id,
			provider_username = EXCLUDED.provider_username,
			display_name = EXCLUDED.display_name,
			profile_url = EXCLUDED.profile_url,
			avatar_url = EXCLUDED.avatar_url,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
		RETURNING
			id, user_id, provider, provider_user_id, provider_union_id,
			provider_username, display_name, profile_url, avatar_url, metadata,
			created_at, updated_at`

	var (
		identity    service.UserExternalIdentity
		unionID     sql.NullString
		metadataRaw []byte
	)
	if err := scanSingleRow(ctx, exec, query, []any{
		userID,
		input.Provider,
		input.ProviderUserID,
		stringPtrValue(input.ProviderUnionID),
		input.ProviderUsername,
		input.DisplayName,
		input.ProfileURL,
		input.AvatarURL,
		string(metadataJSON),
	},
		&identity.ID,
		&identity.UserID,
		&identity.Provider,
		&identity.ProviderUserID,
		&unionID,
		&identity.ProviderUsername,
		&identity.DisplayName,
		&identity.ProfileURL,
		&identity.AvatarURL,
		&metadataRaw,
		&identity.CreatedAt,
		&identity.UpdatedAt,
	); err != nil {
		return nil, translatePersistenceError(err, nil, service.ErrExternalIdentityAlreadyBound)
	}
	identity.ProviderUnionID = nullStringPtr(unionID)
	identity.Metadata = unmarshalMetadata(metadataRaw)
	return &identity, nil
}

func (r *userRepository) DeleteExternalIdentity(ctx context.Context, userID int64, provider string) error {
	exec := r.profileSQLExecutor(ctx)
	if exec == nil {
		return errors.New("sql executor is not configured")
	}
	result, err := exec.ExecContext(ctx,
		`DELETE FROM user_external_identities WHERE user_id = $1 AND provider = $2`,
		userID, provider,
	)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrExternalIdentityNotFound
	}
	return nil
}

func (r *userRepository) GetAvatar(ctx context.Context, userID int64) (*service.UserAvatar, error) {
	exec := r.profileSQLExecutor(ctx)
	if exec == nil {
		return nil, errors.New("sql executor is not configured")
	}
	query := `
		SELECT
			id, user_id, storage_provider, storage_key, url, content_type,
			byte_size, sha256, width, height, created_at, updated_at
		FROM user_avatars
		WHERE user_id = $1`

	avatar, err := scanUserAvatar(ctx, exec, query, userID)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrUserAvatarNotFound, nil)
	}
	return avatar, nil
}

func (r *userRepository) UpsertAvatar(ctx context.Context, userID int64, input service.UpsertUserAvatarInput) (*service.UserAvatar, error) {
	exec := r.profileSQLExecutor(ctx)
	if exec == nil {
		return nil, errors.New("sql executor is not configured")
	}

	query := `
		INSERT INTO user_avatars (
			user_id, storage_provider, storage_key, url, content_type,
			byte_size, sha256, width, height
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (user_id) DO UPDATE SET
			storage_provider = EXCLUDED.storage_provider,
			storage_key = EXCLUDED.storage_key,
			url = EXCLUDED.url,
			content_type = EXCLUDED.content_type,
			byte_size = EXCLUDED.byte_size,
			sha256 = EXCLUDED.sha256,
			width = EXCLUDED.width,
			height = EXCLUDED.height,
			updated_at = NOW()
		RETURNING
			id, user_id, storage_provider, storage_key, url, content_type,
			byte_size, sha256, width, height, created_at, updated_at`

	avatar, err := scanUserAvatar(ctx, exec, query,
		userID,
		input.StorageProvider,
		input.StorageKey,
		input.URL,
		input.ContentType,
		input.ByteSize,
		input.SHA256,
		input.Width,
		input.Height,
	)
	if err != nil {
		return nil, err
	}
	return avatar, nil
}

func (r *userRepository) DeleteAvatar(ctx context.Context, userID int64) error {
	exec := r.profileSQLExecutor(ctx)
	if exec == nil {
		return errors.New("sql executor is not configured")
	}
	result, err := exec.ExecContext(ctx, `DELETE FROM user_avatars WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrUserAvatarNotFound
	}
	return nil
}

func (r *userRepository) hydrateProfileData(ctx context.Context, users ...*service.User) error {
	userIDs := make([]int64, 0, len(users))
	userMap := make(map[int64]*service.User, len(users))
	for _, user := range users {
		if user == nil || user.ID == 0 {
			continue
		}
		userIDs = append(userIDs, user.ID)
		userMap[user.ID] = user
	}
	if len(userIDs) == 0 {
		return nil
	}

	identitiesByUser, err := r.loadExternalIdentities(ctx, userIDs)
	if err != nil {
		return err
	}
	avatarsByUser, err := r.loadAvatars(ctx, userIDs)
	if err != nil {
		return err
	}

	for userID, user := range userMap {
		service.ApplyIdentityProfileData(user, avatarsByUser[userID], identitiesByUser[userID])
	}
	return nil
}

func (r *userRepository) loadExternalIdentities(ctx context.Context, userIDs []int64) (map[int64][]service.UserExternalIdentity, error) {
	out := make(map[int64][]service.UserExternalIdentity, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	exec := r.profileSQLExecutor(ctx)
	if exec == nil {
		return out, nil
	}

	rows, err := exec.QueryContext(ctx, `
		SELECT
			id, user_id, provider, provider_user_id, provider_union_id,
			provider_username, display_name, profile_url, avatar_url, metadata,
			created_at, updated_at
		FROM user_external_identities
		WHERE user_id = ANY($1)
		ORDER BY user_id ASC, provider ASC
	`, pq.Array(userIDs))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			identity    service.UserExternalIdentity
			unionID     sql.NullString
			metadataRaw []byte
		)
		if err := rows.Scan(
			&identity.ID,
			&identity.UserID,
			&identity.Provider,
			&identity.ProviderUserID,
			&unionID,
			&identity.ProviderUsername,
			&identity.DisplayName,
			&identity.ProfileURL,
			&identity.AvatarURL,
			&metadataRaw,
			&identity.CreatedAt,
			&identity.UpdatedAt,
		); err != nil {
			return nil, err
		}
		identity.ProviderUnionID = nullStringPtr(unionID)
		identity.Metadata = unmarshalMetadata(metadataRaw)
		out[identity.UserID] = append(out[identity.UserID], identity)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *userRepository) loadAvatars(ctx context.Context, userIDs []int64) (map[int64]*service.UserAvatar, error) {
	out := make(map[int64]*service.UserAvatar, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	exec := r.profileSQLExecutor(ctx)
	if exec == nil {
		return out, nil
	}

	rows, err := exec.QueryContext(ctx, `
		SELECT
			id, user_id, storage_provider, storage_key, url, content_type,
			byte_size, sha256, width, height, created_at, updated_at
		FROM user_avatars
		WHERE user_id = ANY($1)
	`, pq.Array(userIDs))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		avatar, err := scanUserAvatarRow(rows)
		if err != nil {
			return nil, err
		}
		out[avatar.UserID] = avatar
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *userRepository) profileSQLExecutor(ctx context.Context) sqlExecutor {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return tx.Client()
	}
	return r.sql
}

func scanUserAvatar(ctx context.Context, exec sqlExecutor, query string, args ...any) (*service.UserAvatar, error) {
	avatar := &service.UserAvatar{}
	var width, height sql.NullInt64
	if err := scanSingleRow(ctx, exec, query, args, &avatar.ID, &avatar.UserID, &avatar.StorageProvider,
		&avatar.StorageKey, &avatar.URL, &avatar.ContentType, &avatar.ByteSize, &avatar.SHA256,
		&width, &height, &avatar.CreatedAt, &avatar.UpdatedAt); err != nil {
		return nil, err
	}
	avatar.Width = nullInt64Ptr(width)
	avatar.Height = nullInt64Ptr(height)
	return avatar, nil
}

func scanUserAvatarRow(scanner interface{ Scan(dest ...any) error }) (*service.UserAvatar, error) {
	avatar := &service.UserAvatar{}
	var width, height sql.NullInt64
	if err := scanner.Scan(
		&avatar.ID,
		&avatar.UserID,
		&avatar.StorageProvider,
		&avatar.StorageKey,
		&avatar.URL,
		&avatar.ContentType,
		&avatar.ByteSize,
		&avatar.SHA256,
		&width,
		&height,
		&avatar.CreatedAt,
		&avatar.UpdatedAt,
	); err != nil {
		return nil, err
	}
	avatar.Width = nullInt64Ptr(width)
	avatar.Height = nullInt64Ptr(height)
	return avatar, nil
}

func normalizeMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		return map[string]any{}
	}
	return metadata
}

func unmarshalMetadata(raw []byte) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var metadata map[string]any
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return nil
	}
	if len(metadata) == 0 {
		return nil
	}
	return metadata
}

func stringPtrValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func nullStringPtr(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	value := v.String
	return &value
}

func nullInt64Ptr(v sql.NullInt64) *int {
	if !v.Valid {
		return nil
	}
	value := int(v.Int64)
	return &value
}
