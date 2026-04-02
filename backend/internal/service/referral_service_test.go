//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =====================
// stubReferralRepo
// =====================

type stubReferralRepo struct {
	createProfile          func(ctx context.Context, userID int64, code string) (*UserReferralProfile, error)
	getProfileByUserID     func(ctx context.Context, userID int64) (*UserReferralProfile, error)
	getProfileByCode       func(ctx context.Context, code string) (*UserReferralProfile, error)
	createRelation         func(ctx context.Context, relation *ReferralRelation) error
	getRelationByInviteeID func(ctx context.Context, inviteeID int64) (*ReferralRelation, error)
	markRewardGranted      func(ctx context.Context, id int64) error
	listByInviterID        func(ctx context.Context, inviterID int64, params pagination.PaginationParams) ([]ReferralRelation, *pagination.PaginationResult, error)
	countByInviterID       func(ctx context.Context, inviterID int64) (int64, error)
	sumRewardsByInviterID  func(ctx context.Context, inviterID int64) (float64, error)
	getPlatformStats       func(ctx context.Context) (*ReferralStats, error)
}

func (s *stubReferralRepo) CreateProfile(ctx context.Context, userID int64, code string) (*UserReferralProfile, error) {
	if s.createProfile != nil {
		return s.createProfile(ctx, userID, code)
	}
	panic("unexpected CreateProfile call")
}
func (s *stubReferralRepo) GetProfileByUserID(ctx context.Context, userID int64) (*UserReferralProfile, error) {
	if s.getProfileByUserID != nil {
		return s.getProfileByUserID(ctx, userID)
	}
	panic("unexpected GetProfileByUserID call")
}
func (s *stubReferralRepo) GetProfileByCode(ctx context.Context, code string) (*UserReferralProfile, error) {
	if s.getProfileByCode != nil {
		return s.getProfileByCode(ctx, code)
	}
	panic("unexpected GetProfileByCode call")
}
func (s *stubReferralRepo) CreateRelation(ctx context.Context, relation *ReferralRelation) error {
	if s.createRelation != nil {
		return s.createRelation(ctx, relation)
	}
	panic("unexpected CreateRelation call")
}
func (s *stubReferralRepo) GetRelationByInviteeID(ctx context.Context, inviteeID int64) (*ReferralRelation, error) {
	if s.getRelationByInviteeID != nil {
		return s.getRelationByInviteeID(ctx, inviteeID)
	}
	panic("unexpected GetRelationByInviteeID call")
}
func (s *stubReferralRepo) MarkRewardGranted(ctx context.Context, id int64) error {
	if s.markRewardGranted != nil {
		return s.markRewardGranted(ctx, id)
	}
	panic("unexpected MarkRewardGranted call")
}
func (s *stubReferralRepo) ListByInviterID(ctx context.Context, inviterID int64, params pagination.PaginationParams) ([]ReferralRelation, *pagination.PaginationResult, error) {
	if s.listByInviterID != nil {
		return s.listByInviterID(ctx, inviterID, params)
	}
	panic("unexpected ListByInviterID call")
}
func (s *stubReferralRepo) CountByInviterID(ctx context.Context, inviterID int64) (int64, error) {
	if s.countByInviterID != nil {
		return s.countByInviterID(ctx, inviterID)
	}
	panic("unexpected CountByInviterID call")
}
func (s *stubReferralRepo) SumRewardsByInviterID(ctx context.Context, inviterID int64) (float64, error) {
	if s.sumRewardsByInviterID != nil {
		return s.sumRewardsByInviterID(ctx, inviterID)
	}
	panic("unexpected SumRewardsByInviterID call")
}
func (s *stubReferralRepo) GetPlatformStats(ctx context.Context) (*ReferralStats, error) {
	if s.getPlatformStats != nil {
		return s.getPlatformStats(ctx)
	}
	panic("unexpected GetPlatformStats call")
}

// stubUserRepoForReferral 支持按 ID 查找用户
type stubUserRepoForReferral struct {
	users map[int64]*User
}

func newStubUserRepoForReferral(users ...*User) *stubUserRepoForReferral {
	m := make(map[int64]*User, len(users))
	for _, u := range users {
		m[u.ID] = u
	}
	return &stubUserRepoForReferral{users: m}
}

func (r *stubUserRepoForReferral) Create(context.Context, *User) error { return nil }
func (r *stubUserRepoForReferral) GetByID(_ context.Context, id int64) (*User, error) {
	u, ok := r.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	clone := *u
	return &clone, nil
}
func (r *stubUserRepoForReferral) GetByEmail(context.Context, string) (*User, error) {
	return nil, ErrUserNotFound
}
func (r *stubUserRepoForReferral) GetFirstAdmin(context.Context) (*User, error) {
	return nil, ErrUserNotFound
}
func (r *stubUserRepoForReferral) Update(context.Context, *User) error        { return nil }
func (r *stubUserRepoForReferral) Delete(context.Context, int64) error        { return nil }
func (r *stubUserRepoForReferral) UpdateBalance(context.Context, int64, float64) error {
	return nil
}
func (r *stubUserRepoForReferral) DeductBalance(context.Context, int64, float64) error {
	return nil
}
func (r *stubUserRepoForReferral) UpdateConcurrency(context.Context, int64, int) error {
	return nil
}
func (r *stubUserRepoForReferral) ExistsByEmail(context.Context, string) (bool, error) {
	return false, nil
}
func (r *stubUserRepoForReferral) List(context.Context, pagination.PaginationParams) ([]User, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (r *stubUserRepoForReferral) ListWithFilters(context.Context, pagination.PaginationParams, UserListFilters) ([]User, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (r *stubUserRepoForReferral) RemoveGroupFromAllowedGroups(context.Context, int64) (int64, error) {
	return 0, nil
}
func (r *stubUserRepoForReferral) AddGroupToAllowedGroups(context.Context, int64, int64) error {
	return nil
}
func (r *stubUserRepoForReferral) UpdateTotpSecret(context.Context, int64, *string) error {
	return nil
}
func (r *stubUserRepoForReferral) EnableTotp(context.Context, int64) error  { return nil }
func (r *stubUserRepoForReferral) DisableTotp(context.Context, int64) error { return nil }
func (r *stubUserRepoForReferral) RemoveGroupFromUserAllowedGroups(context.Context, int64, int64) error {
	return nil
}

// =====================
// 辅助函数测试
// =====================

func TestMaskEmail(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"空字符串", "", ""},
		{"短local部分(<=3)", "ab@example.com", "ab***@example.com"},
		{"恰好3位local", "abc@example.com", "abc***@example.com"},
		{"正常邮箱", "alice@example.com", "ali***@example.com"},
		{"无@符号", "notanemail", "notanemail"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, maskEmail(tt.input))
		})
	}
}

func TestBuildReferralLink(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		code    string
		want    string
	}{
		{"空baseURL", "", "CODE1234", "?ref=CODE1234"},
		{"有baseURL", "https://example.com", "CODE1234", "https://example.com/register?ref=CODE1234"},
		{"末尾有斜杠", "https://example.com/", "CODE1234", "https://example.com/register?ref=CODE1234"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, buildReferralLink(tt.baseURL, tt.code))
		})
	}
}

func TestIsUniqueConflict(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"duplicate key", errors.New("pq: duplicate key value violates unique constraint"), true},
		{"unique constraint", errors.New("ERROR: unique constraint violation"), true},
		{"duplicate entry", errors.New("Error 1062: Duplicate entry '...' for key"), true},
		{"普通错误", errors.New("connection refused"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isUniqueConflict(tt.err))
		})
	}
}

func TestGenerateReferralCode(t *testing.T) {
	validChars := map[byte]struct{}{}
	for _, c := range referralCodeCharset {
		validChars[byte(c)] = struct{}{}
	}

	for i := 0; i < 20; i++ {
		code, err := generateReferralCode()
		require.NoError(t, err)
		assert.Len(t, code, referralCodeLength, "code should be %d chars", referralCodeLength)
		for _, b := range []byte(code) {
			_, ok := validChars[b]
			assert.True(t, ok, "char %q not in charset", b)
		}
	}
}

// =====================
// GetOrCreateProfile 测试
// =====================

func TestGetOrCreateProfile_ExistingProfile(t *testing.T) {
	existing := &UserReferralProfile{ID: 1, UserID: 42, ReferralCode: "ABCD1234"}
	repo := &stubReferralRepo{
		getProfileByUserID: func(_ context.Context, userID int64) (*UserReferralProfile, error) {
			return existing, nil
		},
	}
	svc := NewReferralService(repo, newStubUserRepoForReferral(), nil, nil, nil)

	profile, err := svc.GetOrCreateProfile(context.Background(), 42)
	require.NoError(t, err)
	assert.Equal(t, existing, profile)
}

func TestGetOrCreateProfile_NewProfile(t *testing.T) {
	created := &UserReferralProfile{ID: 2, UserID: 7, ReferralCode: "NEWCODE1"}
	repo := &stubReferralRepo{
		getProfileByUserID: func(_ context.Context, userID int64) (*UserReferralProfile, error) {
			return nil, nil
		},
		createProfile: func(_ context.Context, userID int64, code string) (*UserReferralProfile, error) {
			return created, nil
		},
	}
	svc := NewReferralService(repo, newStubUserRepoForReferral(), nil, nil, nil)

	profile, err := svc.GetOrCreateProfile(context.Background(), 7)
	require.NoError(t, err)
	assert.Equal(t, created, profile)
}

func TestGetOrCreateProfile_RetryOnUniqueConflict(t *testing.T) {
	attempts := 0
	created := &UserReferralProfile{ID: 3, UserID: 9, ReferralCode: "RETRY123"}
	repo := &stubReferralRepo{
		getProfileByUserID: func(_ context.Context, userID int64) (*UserReferralProfile, error) {
			return nil, nil
		},
		createProfile: func(_ context.Context, userID int64, code string) (*UserReferralProfile, error) {
			attempts++
			if attempts == 1 {
				return nil, errors.New("pq: duplicate key value violates unique constraint")
			}
			return created, nil
		},
	}
	svc := NewReferralService(repo, newStubUserRepoForReferral(), nil, nil, nil)

	profile, err := svc.GetOrCreateProfile(context.Background(), 9)
	require.NoError(t, err)
	assert.Equal(t, created, profile)
	assert.Equal(t, 2, attempts)
}

func TestGetOrCreateProfile_MaxRetries(t *testing.T) {
	repo := &stubReferralRepo{
		getProfileByUserID: func(_ context.Context, userID int64) (*UserReferralProfile, error) {
			return nil, nil
		},
		createProfile: func(_ context.Context, userID int64, code string) (*UserReferralProfile, error) {
			return nil, errors.New("pq: duplicate key value violates unique constraint")
		},
	}
	svc := NewReferralService(repo, newStubUserRepoForReferral(), nil, nil, nil)

	_, err := svc.GetOrCreateProfile(context.Background(), 10)
	require.Error(t, err)
}

func TestGetOrCreateProfile_RepoError(t *testing.T) {
	repo := &stubReferralRepo{
		getProfileByUserID: func(_ context.Context, userID int64) (*UserReferralProfile, error) {
			return nil, errors.New("db connection lost")
		},
	}
	svc := NewReferralService(repo, newStubUserRepoForReferral(), nil, nil, nil)

	_, err := svc.GetOrCreateProfile(context.Background(), 1)
	require.Error(t, err)
}

// =====================
// ValidateReferralCode 测试
// =====================

func TestValidateReferralCode_Empty(t *testing.T) {
	repo := &stubReferralRepo{}
	svc := NewReferralService(repo, newStubUserRepoForReferral(), nil, nil, nil)

	profile, err := svc.ValidateReferralCode(context.Background(), "", 1)
	assert.NoError(t, err)
	assert.Nil(t, profile)
}

func TestValidateReferralCode_LowercaseAutoUpper(t *testing.T) {
	expected := &UserReferralProfile{ID: 5, UserID: 99, ReferralCode: "ABCD1234"}
	repo := &stubReferralRepo{
		getProfileByCode: func(_ context.Context, code string) (*UserReferralProfile, error) {
			assert.Equal(t, "ABCD1234", code) // 验证已转大写
			return expected, nil
		},
	}
	svc := NewReferralService(repo, newStubUserRepoForReferral(), nil, nil, nil)

	profile, err := svc.ValidateReferralCode(context.Background(), "abcd1234", 42)
	require.NoError(t, err)
	assert.Equal(t, expected, profile)
}

func TestValidateReferralCode_NotFound(t *testing.T) {
	repo := &stubReferralRepo{
		getProfileByCode: func(_ context.Context, code string) (*UserReferralProfile, error) {
			return nil, ErrReferralCodeNotFound
		},
	}
	svc := NewReferralService(repo, newStubUserRepoForReferral(), nil, nil, nil)

	_, err := svc.ValidateReferralCode(context.Background(), "BADCODE1", 42)
	assert.ErrorIs(t, err, ErrReferralCodeNotFound)
}

func TestValidateReferralCode_SelfReferral(t *testing.T) {
	profile := &UserReferralProfile{ID: 5, UserID: 42, ReferralCode: "MYCODE12"}
	repo := &stubReferralRepo{
		getProfileByCode: func(_ context.Context, code string) (*UserReferralProfile, error) {
			return profile, nil
		},
	}
	svc := NewReferralService(repo, newStubUserRepoForReferral(), nil, nil, nil)

	_, err := svc.ValidateReferralCode(context.Background(), "MYCODE12", 42) // inviteeID == profile.UserID
	assert.ErrorIs(t, err, ErrSelfReferralNotAllowed)
}

func TestValidateReferralCode_Valid(t *testing.T) {
	profile := &UserReferralProfile{ID: 5, UserID: 99, ReferralCode: "VALID123"}
	repo := &stubReferralRepo{
		getProfileByCode: func(_ context.Context, code string) (*UserReferralProfile, error) {
			return profile, nil
		},
	}
	svc := NewReferralService(repo, newStubUserRepoForReferral(), nil, nil, nil)

	result, err := svc.ValidateReferralCode(context.Background(), "VALID123", 42)
	require.NoError(t, err)
	assert.Equal(t, profile, result)
}

// =====================
// GrantRewardsInTx 测试
// =====================

func TestGrantRewardsInTx_Normal(t *testing.T) {
	var createdRelation *ReferralRelation
	var inviterUpdated, inviteeUpdated bool
	var markedID int64

	userRepo := newStubUserRepoForReferral(&User{ID: 1}, &User{ID: 2})
	updateBalance := func(_ context.Context, id int64, _ float64) error {
		if id == 1 {
			inviterUpdated = true
		}
		if id == 2 {
			inviteeUpdated = true
		}
		return nil
	}
	userRepo.users[1].ID = 1
	userRepo.users[2].ID = 2

	repo := &stubReferralRepo{
		createRelation: func(_ context.Context, rel *ReferralRelation) error {
			rel.ID = 100
			createdRelation = rel
			return nil
		},
		markRewardGranted: func(_ context.Context, id int64) error {
			markedID = id
			return nil
		},
	}
	// Override UpdateBalance in stubUserRepoForReferral
	ur := newStubUserRepoForReferral()
	var inviterUpdateCalled, inviteeUpdateCalled int64
	_ = updateBalance
	customRepo := &stubReferralRepo{
		createRelation: func(_ context.Context, rel *ReferralRelation) error {
			rel.ID = 100
			createdRelation = rel
			return nil
		},
		markRewardGranted: func(_ context.Context, id int64) error {
			markedID = id
			return nil
		},
	}

	balanceCallCount := map[int64]int{}
	mur := &mockUserRepo{
		updateBalanceFn: func(_ context.Context, id int64, amount float64) error {
			balanceCallCount[id]++
			if id == 1 {
				inviterUpdated = true
			}
			if id == 2 {
				inviteeUpdated = true
			}
			return nil
		},
	}
	_ = repo
	_ = ur
	_ = inviterUpdated
	_ = inviteeUpdated
	_ = inviterUpdateCalled
	_ = inviteeUpdateCalled

	svc := NewReferralService(customRepo, mur, nil, nil, nil)
	err := svc.GrantRewardsInTx(context.Background(), 1, 2, 5.0, 3.0)

	require.NoError(t, err)
	require.NotNil(t, createdRelation)
	assert.Equal(t, int64(1), createdRelation.InviterID)
	assert.Equal(t, int64(2), createdRelation.InviteeID)
	assert.Equal(t, 1, balanceCallCount[1])
	assert.Equal(t, 1, balanceCallCount[2])
	assert.Equal(t, int64(100), markedID)
}

func TestGrantRewardsInTx_SkipInviterRewardWhenZero(t *testing.T) {
	balanceCallCount := map[int64]int{}
	mur := &mockUserRepo{
		updateBalanceFn: func(_ context.Context, id int64, amount float64) error {
			balanceCallCount[id]++
			return nil
		},
	}
	repo := &stubReferralRepo{
		createRelation: func(_ context.Context, rel *ReferralRelation) error {
			rel.ID = 1
			return nil
		},
		markRewardGranted: func(_ context.Context, id int64) error { return nil },
	}
	svc := NewReferralService(repo, mur, nil, nil, nil)

	err := svc.GrantRewardsInTx(context.Background(), 1, 2, 0, 3.0)
	require.NoError(t, err)
	assert.Equal(t, 0, balanceCallCount[1], "inviter reward should be skipped when 0")
	assert.Equal(t, 1, balanceCallCount[2])
}

func TestGrantRewardsInTx_SkipInviteeRewardWhenZero(t *testing.T) {
	balanceCallCount := map[int64]int{}
	mur := &mockUserRepo{
		updateBalanceFn: func(_ context.Context, id int64, amount float64) error {
			balanceCallCount[id]++
			return nil
		},
	}
	repo := &stubReferralRepo{
		createRelation: func(_ context.Context, rel *ReferralRelation) error {
			rel.ID = 1
			return nil
		},
		markRewardGranted: func(_ context.Context, id int64) error { return nil },
	}
	svc := NewReferralService(repo, mur, nil, nil, nil)

	err := svc.GrantRewardsInTx(context.Background(), 1, 2, 5.0, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, balanceCallCount[1])
	assert.Equal(t, 0, balanceCallCount[2], "invitee reward should be skipped when 0")
}

func TestGrantRewardsInTx_CreateRelationError(t *testing.T) {
	repo := &stubReferralRepo{
		createRelation: func(_ context.Context, rel *ReferralRelation) error {
			return errors.New("db error")
		},
	}
	svc := NewReferralService(repo, &mockUserRepo{}, nil, nil, nil)
	err := svc.GrantRewardsInTx(context.Background(), 1, 2, 5.0, 3.0)
	require.Error(t, err)
}

func TestGrantRewardsInTx_InviterBalanceError(t *testing.T) {
	repo := &stubReferralRepo{
		createRelation: func(_ context.Context, rel *ReferralRelation) error {
			rel.ID = 1
			return nil
		},
		markRewardGranted: func(_ context.Context, id int64) error { return nil },
	}
	mur := &mockUserRepo{
		updateBalanceFn: func(_ context.Context, id int64, amount float64) error {
			if id == 1 {
				return errors.New("inviter balance error")
			}
			return nil
		},
	}
	svc := NewReferralService(repo, mur, nil, nil, nil)
	err := svc.GrantRewardsInTx(context.Background(), 1, 2, 5.0, 3.0)
	require.Error(t, err)
}

func TestGrantRewardsInTx_InviteeBalanceError(t *testing.T) {
	repo := &stubReferralRepo{
		createRelation: func(_ context.Context, rel *ReferralRelation) error {
			rel.ID = 1
			return nil
		},
		markRewardGranted: func(_ context.Context, id int64) error { return nil },
	}
	mur := &mockUserRepo{
		updateBalanceFn: func(_ context.Context, id int64, amount float64) error {
			if id == 2 {
				return errors.New("invitee balance error")
			}
			return nil
		},
	}
	svc := NewReferralService(repo, mur, nil, nil, nil)
	err := svc.GrantRewardsInTx(context.Background(), 1, 2, 5.0, 3.0)
	require.Error(t, err)
}

func TestGrantRewardsInTx_MarkRewardGrantedError(t *testing.T) {
	repo := &stubReferralRepo{
		createRelation: func(_ context.Context, rel *ReferralRelation) error {
			rel.ID = 1
			return nil
		},
		markRewardGranted: func(_ context.Context, id int64) error {
			return errors.New("mark error")
		},
	}
	svc := NewReferralService(repo, &mockUserRepo{}, nil, nil, nil)
	err := svc.GrantRewardsInTx(context.Background(), 1, 2, 0, 0)
	require.Error(t, err)
}

// =====================
// GetMyReferralInfo 测试
// =====================

func TestGetMyReferralInfo_Normal_NoInviter(t *testing.T) {
	profile := &UserReferralProfile{ID: 1, UserID: 10, ReferralCode: "CODE1234"}
	repo := &stubReferralRepo{
		getProfileByUserID: func(_ context.Context, userID int64) (*UserReferralProfile, error) {
			return profile, nil
		},
		countByInviterID: func(_ context.Context, inviterID int64) (int64, error) {
			return 5, nil
		},
		sumRewardsByInviterID: func(_ context.Context, inviterID int64) (float64, error) {
			return 12.5, nil
		},
		getRelationByInviteeID: func(_ context.Context, inviteeID int64) (*ReferralRelation, error) {
			return nil, nil // 没有被邀请
		},
	}
	svc := NewReferralService(repo, newStubUserRepoForReferral(), nil, nil, nil)

	info, err := svc.GetMyReferralInfo(context.Background(), 10, "https://example.com")
	require.NoError(t, err)
	assert.Equal(t, "CODE1234", info.ReferralCode)
	assert.Equal(t, "https://example.com/register?ref=CODE1234", info.ReferralLink)
	assert.Equal(t, int64(5), info.TotalInvitees)
	assert.Equal(t, 12.5, info.TotalRewardEarned)
	assert.Nil(t, info.InviterInfo)
}

func TestGetMyReferralInfo_WithInviter(t *testing.T) {
	profile := &UserReferralProfile{ID: 1, UserID: 20, ReferralCode: "MYCODE12"}
	inviterUser := &User{ID: 99, Email: "inviter@example.com"}
	repo := &stubReferralRepo{
		getProfileByUserID: func(_ context.Context, userID int64) (*UserReferralProfile, error) {
			return profile, nil
		},
		countByInviterID: func(_ context.Context, inviterID int64) (int64, error) {
			return 0, nil
		},
		sumRewardsByInviterID: func(_ context.Context, inviterID int64) (float64, error) {
			return 0, nil
		},
		getRelationByInviteeID: func(_ context.Context, inviteeID int64) (*ReferralRelation, error) {
			return &ReferralRelation{ID: 1, InviterID: 99, InviteeID: inviteeID}, nil
		},
	}
	svc := NewReferralService(repo, newStubUserRepoForReferral(inviterUser), nil, nil, nil)

	info, err := svc.GetMyReferralInfo(context.Background(), 20, "")
	require.NoError(t, err)
	require.NotNil(t, info.InviterInfo)
	assert.Equal(t, "inv***@example.com", info.InviterInfo.EmailMasked)
}

func TestGetMyReferralInfo_ProfileError(t *testing.T) {
	repo := &stubReferralRepo{
		getProfileByUserID: func(_ context.Context, userID int64) (*UserReferralProfile, error) {
			return nil, errors.New("db error")
		},
	}
	svc := NewReferralService(repo, newStubUserRepoForReferral(), nil, nil, nil)

	_, err := svc.GetMyReferralInfo(context.Background(), 1, "")
	require.Error(t, err)
}

// =====================
// ListMyInvitees 测试
// =====================

func TestListMyInvitees_Normal(t *testing.T) {
	now := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	relations := []ReferralRelation{
		{InviteeEmail: "alice@example.com", InviterReward: 5.0, CreatedAt: now},
		{InviteeEmail: "bo@example.com", InviterReward: 3.0, CreatedAt: now},
	}
	paginationResult := &pagination.PaginationResult{Total: 2, Page: 1, PageSize: 10, Pages: 1}

	repo := &stubReferralRepo{
		listByInviterID: func(_ context.Context, inviterID int64, params pagination.PaginationParams) ([]ReferralRelation, *pagination.PaginationResult, error) {
			return relations, paginationResult, nil
		},
	}
	svc := NewReferralService(repo, newStubUserRepoForReferral(), nil, nil, nil)

	params := pagination.PaginationParams{Page: 1, PageSize: 10}
	invitees, result, err := svc.ListMyInvitees(context.Background(), 1, params)
	require.NoError(t, err)
	require.Len(t, invitees, 2)
	assert.Equal(t, "ali***@example.com", invitees[0].EmailMasked)
	assert.Equal(t, "bo***@example.com", invitees[1].EmailMasked)
	assert.Equal(t, 5.0, invitees[0].RewardEarned)
	assert.Equal(t, now, invitees[0].RegisteredAt)
	assert.Equal(t, paginationResult, result)
}

func TestListMyInvitees_Empty(t *testing.T) {
	repo := &stubReferralRepo{
		listByInviterID: func(_ context.Context, inviterID int64, params pagination.PaginationParams) ([]ReferralRelation, *pagination.PaginationResult, error) {
			return []ReferralRelation{}, &pagination.PaginationResult{Total: 0, Page: 1, PageSize: 10, Pages: 1}, nil
		},
	}
	svc := NewReferralService(repo, newStubUserRepoForReferral(), nil, nil, nil)

	invitees, _, err := svc.ListMyInvitees(context.Background(), 1, pagination.PaginationParams{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Empty(t, invitees)
}

func TestListMyInvitees_RepoError(t *testing.T) {
	repo := &stubReferralRepo{
		listByInviterID: func(_ context.Context, inviterID int64, params pagination.PaginationParams) ([]ReferralRelation, *pagination.PaginationResult, error) {
			return nil, nil, errors.New("db error")
		},
	}
	svc := NewReferralService(repo, newStubUserRepoForReferral(), nil, nil, nil)

	_, _, err := svc.ListMyInvitees(context.Background(), 1, pagination.PaginationParams{Page: 1, PageSize: 10})
	require.Error(t, err)
}

// =====================
// GetPlatformStats 测试
// =====================

func TestGetPlatformStats_Normal(t *testing.T) {
	expected := &ReferralStats{TotalRelations: 100, TotalInviterRewardGiven: 50.0, TotalInviteeRewardGiven: 30.0}
	repo := &stubReferralRepo{
		getPlatformStats: func(_ context.Context) (*ReferralStats, error) {
			return expected, nil
		},
	}
	svc := NewReferralService(repo, newStubUserRepoForReferral(), nil, nil, nil)

	stats, err := svc.GetPlatformStats(context.Background())
	require.NoError(t, err)
	assert.Equal(t, expected, stats)
}

func TestGetPlatformStats_Error(t *testing.T) {
	repo := &stubReferralRepo{
		getPlatformStats: func(_ context.Context) (*ReferralStats, error) {
			return nil, errors.New("db error")
		},
	}
	svc := NewReferralService(repo, newStubUserRepoForReferral(), nil, nil, nil)

	_, err := svc.GetPlatformStats(context.Background())
	require.Error(t, err)
}

// =====================
// InvalidateRewardCaches 测试
// =====================

func TestInvalidateRewardCaches_NilService(t *testing.T) {
	repo := &stubReferralRepo{}
	svc := NewReferralService(repo, newStubUserRepoForReferral(), nil, nil, nil)
	// nil billingCacheService 时不 panic
	assert.NotPanics(t, func() {
		svc.InvalidateRewardCaches(1, 2)
	})
}

func TestInvalidateRewardCaches_NonNil(t *testing.T) {
	repo := &stubReferralRepo{}
	// BillingCacheService with nil internal cache: InvalidateUserBalance returns nil immediately
	billingCache := &BillingCacheService{cache: nil}
	svc := NewReferralService(repo, newStubUserRepoForReferral(), nil, billingCache, nil)

	assert.NotPanics(t, func() {
		svc.InvalidateRewardCaches(1, 2)
	})
	// 等待异步 goroutine 完成
	time.Sleep(50 * time.Millisecond)
}
