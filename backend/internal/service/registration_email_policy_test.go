//go:build unit

package service

import (
	"context"
	"reflect"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type registrationPolicySettingRepoStub struct {
	values map[string]string
}

func (s *registrationPolicySettingRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *registrationPolicySettingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", ErrSettingNotFound
}

func (s *registrationPolicySettingRepoStub) Set(ctx context.Context, key, value string) error {
	panic("unexpected Set call")
}

func (s *registrationPolicySettingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *registrationPolicySettingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *registrationPolicySettingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *registrationPolicySettingRepoStub) Delete(ctx context.Context, key string) error {
	panic("unexpected Delete call")
}

func TestNormalizeRegistrationEmailSuffixWhitelist(t *testing.T) {
	got, err := NormalizeRegistrationEmailSuffixWhitelist([]string{"example.com", "@EXAMPLE.COM", " @foo.bar "})
	require.NoError(t, err)
	require.Equal(t, []string{"@example.com", "@foo.bar"}, got)
}

func TestNormalizeRegistrationEmailSuffixWhitelist_Invalid(t *testing.T) {
	_, err := NormalizeRegistrationEmailSuffixWhitelist([]string{"@invalid_domain"})
	require.Error(t, err)
}

func TestParseRegistrationEmailSuffixWhitelist(t *testing.T) {
	got := ParseRegistrationEmailSuffixWhitelist(`["example.com","@foo.bar","@invalid_domain"]`)
	require.Equal(t, []string{"@example.com", "@foo.bar"}, got)
}

func TestIsRegistrationEmailSuffixAllowed(t *testing.T) {
	require.True(t, IsRegistrationEmailSuffixAllowed("user@example.com", []string{"@example.com"}))
	require.False(t, IsRegistrationEmailSuffixAllowed("user@sub.example.com", []string{"@example.com"}))
	require.True(t, IsRegistrationEmailSuffixAllowed("user@any.com", []string{}))
}

func TestNormalizeSignupSource(t *testing.T) {
	require.Equal(t, SignupSourceEmail, NormalizeSignupSource(" email "))
	require.Equal(t, SignupSourceLinuxDo, NormalizeSignupSource("LINUXDO"))
	require.Equal(t, SignupSourceOIDC, NormalizeSignupSource("oidc"))
	require.Equal(t, SignupSourceWeChat, NormalizeSignupSource("wechat"))
	require.Equal(t, SignupSourceUnknown, NormalizeSignupSource("unexpected"))
}

func TestUserHasLocalIdentity(t *testing.T) {
	require.True(t, (&User{
		Email:        "user@example.com",
		PasswordHash: "hashed-password",
	}).HasLocalIdentity())

	require.False(t, (&User{
		Email:        "user" + LinuxDoConnectSyntheticEmailDomain,
		PasswordHash: "hashed-password",
	}).HasLocalIdentity())

	require.False(t, (&User{
		Email:        "user" + OIDCConnectSyntheticEmailDomain,
		PasswordHash: "hashed-password",
	}).HasLocalIdentity())

	require.False(t, (&User{
		Email:        "user" + WeChatConnectSyntheticEmailDomain,
		PasswordHash: "hashed-password",
	}).HasLocalIdentity())

	require.False(t, (&User{
		Email: "user@example.com",
	}).HasLocalIdentity())
}

func TestSettingService_GetPublicAuthPolicy(t *testing.T) {
	repo := &registrationPolicySettingRepoStub{
		values: map[string]string{
			SettingKeyRegistrationEnabled:              "true",
			SettingKeyEmailVerifyEnabled:               "true",
			SettingKeyInvitationCodeEnabled:            "true",
			SettingKeyRegistrationEmailSuffixWhitelist: `["@example.com"]`,
			SettingKeyAuthForceEmailOnThirdPartySignup: "true",
			SettingKeyLinuxDoConnectEnabled:            "true",
			SettingKeyWeChatLoginOpenEnabled:           "true",
			SettingKeyWeChatLoginOpenAppID:             "wx-open-app",
			SettingKeyWeChatLoginOpenAppSecret:         "wx-open-secret",
			SettingKeyWeChatLoginMPEnabled:             "true",
			SettingKeyWeChatLoginMPAppID:               "wx-mp-app",
			SettingKeyWeChatLoginMPAppSecret:           "wx-mp-secret",
			SettingKeyOIDCConnectEnabled:               "true",
			SettingKeyOIDCConnectProviderName:          "Example SSO",
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	policy, err := svc.GetPublicAuthPolicy(context.Background())
	require.NoError(t, err)
	require.True(t, policy.RegistrationEnabled)
	require.True(t, policy.EmailVerifyEnabled)
	require.True(t, policy.InvitationCodeEnabled)
	require.True(t, policy.ForceEmailOnThirdPartySignup)
	require.Equal(t, []string{"@example.com"}, policy.RegistrationEmailSuffixWhitelist)
	require.True(t, policy.LinuxDoOAuthEnabled)
	require.True(t, policy.WeChatLoginOpenEnabled)
	require.True(t, policy.WeChatLoginMPEnabled)
	require.Equal(t, WeChatLoginUnionIDHealthStatusOK, policy.WeChatLoginUnionIDHealthStatus)
	require.True(t, policy.OIDCOAuthEnabled)
	require.Equal(t, "Example SSO", policy.OIDCOAuthProviderName)
}

func TestSettingService_GetDefaultUserSettings(t *testing.T) {
	repo := &registrationPolicySettingRepoStub{
		values: map[string]string{
			SettingKeyDefaultBalance:              "12.5",
			SettingKeyDefaultConcurrency:          "7",
			SettingKeyDefaultSubscriptions:        `[{"group_id":11,"validity_days":30}]`,
			"default_apply_on_bind_linuxdo":       "true",
			SettingKeyDefaultBalanceLinuxDo:       "21.5",
			SettingKeyDefaultConcurrencyLinuxDo:   "9",
			SettingKeyDefaultSubscriptionsLinuxDo: `[{"group_id":21,"validity_days":60}]`,
			"default_apply_on_bind_wechat":        "true",
			SettingKeyDefaultBalanceWeChat:        "31.5",
			SettingKeyDefaultConcurrencyWeChat:    "5",
			SettingKeyDefaultSubscriptionsWeChat:  `[{"group_id":31,"validity_days":90}]`,
			"default_apply_on_bind_oidc":          "false",
			SettingKeyDefaultBalanceOIDC:          "41.5",
			SettingKeyDefaultConcurrencyOIDC:      "3",
			SettingKeyDefaultSubscriptionsOIDC:    `[{"group_id":41,"validity_days":120}]`,
			SettingKeyDefaultBalanceEmail:         "12.5",
			SettingKeyDefaultConcurrencyEmail:     "7",
			SettingKeyDefaultSubscriptionsEmail:   `[{"group_id":11,"validity_days":30}]`,
		},
	}
	cfg := &config.Config{}
	cfg.Default.UserBalance = 1.5
	cfg.Default.UserConcurrency = 2

	svc := NewSettingService(repo, cfg)
	settings := svc.GetDefaultUserSettings(context.Background())
	linuxDoSettings := svc.GetDefaultUserSettingsBySignupSource(context.Background(), SignupSourceLinuxDo)
	weChatSettings := svc.GetDefaultUserSettingsBySignupSource(context.Background(), SignupSourceWeChat)
	oidcSettings := svc.GetDefaultUserSettingsBySignupSource(context.Background(), SignupSourceOIDC)

	require.Equal(t, 12.5, settings.Balance)
	require.Equal(t, 7, settings.Concurrency)
	require.Equal(t, []DefaultSubscriptionSetting{{GroupID: 11, ValidityDays: 30}}, settings.Subscriptions)
	require.Equal(t, 21.5, linuxDoSettings.Balance)
	require.Equal(t, 9, linuxDoSettings.Concurrency)
	require.Equal(t, []DefaultSubscriptionSetting{{GroupID: 21, ValidityDays: 60}}, linuxDoSettings.Subscriptions)
	require.Equal(t, 31.5, weChatSettings.Balance)
	require.Equal(t, 5, weChatSettings.Concurrency)
	require.Equal(t, []DefaultSubscriptionSetting{{GroupID: 31, ValidityDays: 90}}, weChatSettings.Subscriptions)
	require.Equal(t, 41.5, oidcSettings.Balance)
	require.Equal(t, 3, oidcSettings.Concurrency)
	require.Equal(t, []DefaultSubscriptionSetting{{GroupID: 41, ValidityDays: 120}}, oidcSettings.Subscriptions)

	linuxDoApplyOnBindField, ok := reflect.TypeOf(linuxDoSettings).FieldByName("ApplyOnBind")
	require.True(t, ok, "Provider default settings should expose ApplyOnBind")
	require.Equal(t, reflect.Bool, linuxDoApplyOnBindField.Type.Kind())
	require.True(t, reflect.ValueOf(linuxDoSettings).FieldByName("ApplyOnBind").Bool())
	require.True(t, reflect.ValueOf(weChatSettings).FieldByName("ApplyOnBind").Bool())
	require.False(t, reflect.ValueOf(oidcSettings).FieldByName("ApplyOnBind").Bool())
}
